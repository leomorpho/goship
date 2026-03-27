package storagerepo

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/domain"
	storagequeries "github.com/leomorpho/goship/framework/storage/queries"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/afero"
)

type NoImagesInFiles struct {
	Message string
}

// Error implements the error interface
func (e *NoImagesInFiles) Error() string {
	return fmt.Sprintf("NoImagesInFiles: %s", e.Message)
}

type Bucket string

const (
	BucketMainApp     Bucket = "main-app"
	BucketStaticFiles Bucket = "static-files"
)

var defaultImagePresignedURLExpiry = 48 * time.Hour

var ErrBucketDoesNotExist = errors.New("requested bucket does not exist")

// StorageClientInterface defines the interface for the storage client.
type StorageClientInterface interface {
	CreateBucket(bucketName string, location string) error
	UploadFile(bucket Bucket, objectName string, fileStream io.Reader) (*int, error)
	DeleteFile(bucket Bucket, objectName string) error
	GetPresignedURL(bucket Bucket, objectName string, expiry time.Duration) (string, error)
	GetImageObjectFromFile(file *ImageFile) (*domain.Photo, error)
	GetImageObjectsFromFiles(files []*ImageFile) ([]domain.Photo, error)
}

func (b Bucket) String() string {
	return string(b)
}

func ParseBucket(raw string) (Bucket, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "main-app", "main", "app":
		return BucketMainApp, nil
	case "static-files", "static":
		return BucketStaticFiles, nil
	default:
		return "", ErrBucketDoesNotExist
	}
}

type ImageFile struct {
	ID    int
	Sizes []ImageFileSize
}

type ImageFileSize struct {
	Size      string
	Height    int
	Width     int
	ObjectKey string
}

type StorageClient struct {
	config      *config.Config
	db          *sql.DB
	postgresql  bool
	minioClient *minio.Client
	fs          afero.Fs
}

func NewStorageClient(cfg *config.Config, db *sql.DB, dialect string) *StorageClient {
	sc := &StorageClient{
		config:     cfg,
		db:         db,
		postgresql: strings.EqualFold(strings.TrimSpace(dialect), "postgres") || strings.EqualFold(strings.TrimSpace(dialect), "postgresql") || strings.EqualFold(strings.TrimSpace(dialect), "pgx"),
	}

	if cfg.App.Environment == config.EnvTest {
		sc.fs = afero.NewMemMapFs()
		return sc
	}

	switch cfg.Storage.Driver {
	case config.StorageDriverMinIO:
		minioClient, err := minio.New(cfg.Storage.S3Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.Storage.S3AccessKey, cfg.Storage.S3SecretKey, ""),
			Secure: cfg.Storage.S3UseSSL,
		})
		if err != nil {
			slog.Error("failed to initialize minio client", "error", err)
			os.Exit(1)
		}
		sc.minioClient = minioClient
	case config.StorageDriverLocal:
		fallthrough
	default:
		base := afero.NewOsFs()
		if err := base.MkdirAll(cfg.Storage.LocalStoragePath, 0o755); err != nil {
			slog.Error("failed to create local storage path", "path", cfg.Storage.LocalStoragePath, "error", err)
			os.Exit(1)
		}
		sc.fs = afero.NewBasePathFs(base, cfg.Storage.LocalStoragePath)
	}

	return sc
}

func (sc *StorageClient) getBucketName(b Bucket) (string, error) {
	switch b {
	case BucketMainApp:
		return sc.config.Storage.AppBucketName, nil
	case BucketStaticFiles:
		return sc.config.Storage.StaticFilesBucketName, nil
	default:
		return "", ErrBucketDoesNotExist
	}
}

func (sc *StorageClient) CreateBucket(bucketName string, location string) error {
	ctx := context.Background()

	bucketName = bucketName + string(sc.config.App.Environment)

	if sc.fs != nil {
		return sc.fs.MkdirAll(bucketName, 0o755)
	}

	err := sc.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := sc.minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			slog.Info("Bucket already exists", "bucket", bucketName)
		} else {
			return err
		}
	} else {
		slog.Info("Successfully created bucket", "bucket", bucketName)
	}

	return nil
}

func (sc *StorageClient) UploadFile(bucket Bucket, objectName string, fileStream io.Reader) (*int, error) {
	ctx := context.Background()

	bucketName, err := sc.getBucketName(bucket)
	if err != nil {
		return nil, err
	}

	// Calculate file size and hash
	hash := md5.New()
	var size int64
	var fileHash string

	if sc.fs != nil {
		// For Afero, we can use a TeeReader to calculate hash while writing
		if err := sc.fs.MkdirAll(bucketName, 0o755); err != nil {
			return nil, err
		}
		fullPath := filepath.Join(bucketName, objectName)
		file, err := sc.fs.Create(fullPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		tee := io.TeeReader(fileStream, hash)
		size, err = io.Copy(file, tee)
		if err != nil {
			return nil, err
		}
		fileHash = hex.EncodeToString(hash.Sum(nil))
	} else {
		size, err = io.Copy(hash, fileStream)
		if err != nil {
			return nil, err
		}
		fileHash = hex.EncodeToString(hash.Sum(nil))

		// Seek to the beginning of the file stream if possible
		if seeker, ok := fileStream.(io.Seeker); ok {
			_, err = seeker.Seek(0, io.SeekStart)
			if err != nil {
				return nil, err
			}
		}

		// Upload file to S3-compatible storage
		_, err = sc.minioClient.PutObject(ctx, bucketName, objectName, fileStream, size, minio.PutObjectOptions{})
		if err != nil {
			return nil, err
		}
	}

	fileID, err := sc.insertFileStorageRow(ctx, bucketName, objectName, size, fileHash)
	if err != nil {
		return nil, err
	}
	return &fileID, nil
}

func (sc *StorageClient) GetPresignedURL(bucket Bucket, objectName string, expiry time.Duration) (string, error) {
	ctx := context.Background()

	bucketName, err := sc.getBucketName(bucket)
	if err != nil {
		return "", err
	}

	if sc.fs != nil {
		// For local storage, return a relative URL that the app serves.
		// We use /uploads as the prefix which should be registered in the router.
		return fmt.Sprintf("/uploads/%s/%s", bucketName, objectName), nil
	}

	presignedURL, err := sc.minioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (sc *StorageClient) DeleteFile(bucket Bucket, objectName string) error {
	ctx := context.Background()
	bucketName, err := sc.getBucketName(bucket)
	if err != nil {
		return err
	}

	if sc.fs != nil {
		fullPath := filepath.Join(bucketName, objectName)
		err = sc.fs.Remove(fullPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	} else {
		err = sc.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{ForceDelete: true})
		if err != nil {
			return err
		}
	}

	queryName := "delete_file_storage_by_object_key_sqlite"
	if sc.postgresql {
		queryName = "delete_file_storage_by_object_key_postgres"
	}
	query, err := storagequeries.Get(queryName)
	if err != nil {
		return err
	}
	_, err = sc.db.ExecContext(ctx, query, objectName)

	if err != nil {
		return err
	}
	return nil
}

// TODO: GetImageObjectFromFile and GetImageObjectsFromFiles can be standardized
// to return a specific file object. Expiration can also be parametrized if necessary.
// getPhotoObjectFromFile generates a signed URL for a single file.
func (sc *StorageClient) GetImageObjectFromFile(image *ImageFile) (*domain.Photo, error) {
	photo, err := sc.hydratePhoto(image, sc.imagePresignedURLExpiry())
	if err != nil {
		return nil, err
	}
	return photo, nil
}

func (sc *StorageClient) GetImageObjectsFromFiles(
	files []*ImageFile,
) ([]domain.Photo, error) {
	return sc.hydratePhotos(files, sc.imagePresignedURLExpiry())
}

func (sc *StorageClient) imagePresignedURLExpiry() time.Duration {
	return defaultImagePresignedURLExpiry
}

func (sc *StorageClient) hydratePhotos(files []*ImageFile, expiry time.Duration) ([]domain.Photo, error) {
	if len(files) == 0 {
		return nil, &NoImagesInFiles{Message: "no images in files"}
	}
	photos := make([]domain.Photo, 0, len(files))
	for _, file := range files {
		photo, err := sc.hydratePhoto(file, expiry)
		if err != nil {
			return nil, err
		}
		if photo == nil {
			continue
		}
		photos = append(photos, *photo)
	}
	return photos, nil
}

func (sc *StorageClient) hydratePhoto(image *ImageFile, expiry time.Duration) (*domain.Photo, error) {
	if image == nil || len(image.Sizes) == 0 {
		return nil, nil
	}

	photo := &domain.Photo{ID: image.ID}
	for _, size := range image.Sizes {
		if size.ObjectKey == "" {
			continue
		}
		url, err := sc.GetPresignedURL(BucketMainApp, size.ObjectKey, expiry)
		if err != nil {
			return nil, err
		}
		switch *domain.ImageSizes.Parse(size.Size) {
		case domain.ImageSizeFull:
			photo.FullURL = url
			photo.FullHeight = size.Height
			photo.FullWidth = size.Width
		case domain.ImageSizePreview:
			photo.PreviewURL = url
			photo.PreviewHeight = size.Height
			photo.PreviewWidth = size.Width
		case domain.ImageSizeThumbnail:
			photo.ThumbnailURL = url
			photo.ThumbnailHeight = size.Height
			photo.ThumbnailWidth = size.Width
		}
	}
	return photo, nil
}

func (sc *StorageClient) insertFileStorageRow(
	ctx context.Context,
	bucketName string,
	objectName string,
	size int64,
	fileHash string,
) (int, error) {
	now := time.Now().UTC()
	if sc.postgresql {
		query, err := storagequeries.Get("insert_file_storage_returning_id_postgres")
		if err != nil {
			return 0, err
		}
		var id int
		err = sc.db.QueryRowContext(ctx, query, now, now, bucketName, objectName, size, fileHash).Scan(&id)
		return id, err
	}

	query, err := storagequeries.Get("insert_file_storage_sqlite")
	if err != nil {
		return 0, err
	}
	result, err := sc.db.ExecContext(ctx, query, now, now, bucketName, objectName, size, fileHash)
	if err != nil {
		return 0, err
	}
	id64, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id64), nil
}
