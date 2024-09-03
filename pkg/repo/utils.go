package repo

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidMimeType      = errors.New("invalid MIME type")
	ErrInvalidFileExtension = errors.New("invalid file extension")
	ErrImageProcessing      = errors.New("error processing image")
)

func ValidateAndProcessImage(fileHeader *multipart.FileHeader) error {
	// TODO: need to do many other checks for file upload security: https://portswigger.net/web-security/file-upload

	// Define allowed MIME types
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/webp": true,
		"image/png":  true,
		"image/gif":  true,
	}

	// Define allowed file extensions
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".png":  true,
		".gif":  true,
	}

	// Check MIME type
	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		return ErrInvalidMimeType
	}

	// Check file extension
	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[extension] {
		return ErrInvalidFileExtension
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// TODO: verify that the file actually contains an image
	return nil
}

// daysAgo returns a time.Time object for x days ago.
func daysAgo(x int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -x)
}
