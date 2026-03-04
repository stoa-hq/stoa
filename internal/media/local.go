package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	basePath string
	baseURL  string
}

func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("creating upload directory: %w", err)
	}
	return &LocalStorage{basePath: basePath, baseURL: baseURL}, nil
}

func (s *LocalStorage) Store(ctx context.Context, filename string, reader io.Reader, size int64) (*StoredFile, error) {
	// Generate unique path: YYYY/MM/uuid-filename
	now := time.Now()
	dir := filepath.Join(s.basePath, fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	uniqueName := fmt.Sprintf("%s-%s", uuid.New().String()[:8], filename)
	fullPath := filepath.Join(dir, uniqueName)

	f, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("writing file: %w", err)
	}

	relPath, _ := filepath.Rel(s.basePath, fullPath)

	return &StoredFile{
		Path:     relPath,
		URL:      s.URL(relPath),
		Size:     written,
	}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

func (s *LocalStorage) URL(path string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, path)
}
