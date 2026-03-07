package storagerepo

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/domain"
	storagequeries "github.com/leomorpho/goship/framework/repos/storage/queries"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type NoImagesInFiles struct {
	Message string
}

// Error implements the error interface
func (e *NoImagesInFiles) Error() string {
	return fmt.Sprintf("NoImagesInFiles: %s", e.Message)
}

// TODO: use github.com/orsinium-labs/enum
type Bucket int

const (
	BucketMainApp Bucket = iota
	BucketStaticFiles
)

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
}

func NewStorageClient(cfg *config.Config, db *sql.DB, dialect string) *StorageClient {
	minioClient, err := minio.New(cfg.Storage.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.S3AccessKey, cfg.Storage.S3SecretKey, ""),
		Secure: cfg.Storage.S3UseSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	return &StorageClient{
		config:      cfg,
		db:          db,
		postgresql:  strings.EqualFold(strings.TrimSpace(dialect), "postgres") || strings.EqualFold(strings.TrimSpace(dialect), "postgresql") || strings.EqualFold(strings.TrimSpace(dialect), "pgx"),
		minioClient: minioClient,
	}
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

	err := sc.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := sc.minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket %s already exists\n", bucketName)
		} else {
			return err
		}
	} else {
		log.Printf("Successfully created bucket %s\n", bucketName)
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
	size, err := io.Copy(hash, fileStream)
	if err != nil {
		return nil, err
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

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
	err = sc.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return err
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
	if image == nil || len(image.Sizes) == 0 {
		return nil, nil
	}

	p := &domain.Photo{
		ID: image.ID,
	}

	for _, size := range image.Sizes {
		if size.ObjectKey == "" {
			continue
		}
		// Generate a presigned URL with a specified duration
		url, err := sc.GetPresignedURL(BucketMainApp, size.ObjectKey, 2*24*time.Hour) // Adjust duration as needed
		if err != nil {
			return nil, err
		}
		switch *domain.ImageSizes.Parse(size.Size) {
		case domain.ImageSizeFull:
			p.FullURL = url
			p.FullHeight = size.Height
			p.FullWidth = size.Width
		case domain.ImageSizePreview:
			p.PreviewURL = url
			p.PreviewHeight = size.Height
			p.PreviewWidth = size.Width
		case domain.ImageSizeThumbnail:
			p.ThumbnailURL = url
			p.ThumbnailHeight = size.Height
			p.ThumbnailWidth = size.Width
		}
	}

	return p, nil
}

func (sc *StorageClient) GetImageObjectsFromFiles(
	files []*ImageFile,
) ([]domain.Photo, error) {
	if len(files) == 0 {
		return nil, &NoImagesInFiles{Message: "no images in files"}
	}
	var photos []domain.Photo
	for _, f := range files {
		photo, err := sc.GetImageObjectFromFile(f)
		if err != nil {
			return nil, err
		}
		photos = append(photos, *photo)
	}
	return photos, nil
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
