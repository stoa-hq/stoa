package media

import (
	"context"
	"io"
)

// StoredFile represents a file that has been stored.
type StoredFile struct {
	Path     string
	URL      string
	Size     int64
	MimeType string
}

// Storage defines the interface for media storage backends.
type Storage interface {
	Store(ctx context.Context, filename string, reader io.Reader, size int64) (*StoredFile, error)
	Delete(ctx context.Context, path string) error
	URL(path string) string
}
