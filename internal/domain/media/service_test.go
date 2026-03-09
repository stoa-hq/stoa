package media

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock MediaRepository
// ---------------------------------------------------------------------------

type mockMediaRepo struct {
	create   func(ctx context.Context, m *Media) error
	findByID func(ctx context.Context, id uuid.UUID) (*Media, error)
	findAll  func(ctx context.Context, f MediaFilter) ([]Media, int, error)
	delete   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockMediaRepo) Create(ctx context.Context, media *Media) error {
	if m.create != nil {
		return m.create(ctx, media)
	}
	return nil
}
func (m *mockMediaRepo) FindByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockMediaRepo) FindAll(ctx context.Context, f MediaFilter) ([]Media, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockMediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock StorageBackend
// ---------------------------------------------------------------------------

type mockStorage struct {
	store  func(ctx context.Context, filename string, src io.Reader) (string, error)
	delete func(ctx context.Context, storagePath string) error
}

func (m *mockStorage) Store(ctx context.Context, filename string, src io.Reader) (string, error) {
	if m.store != nil {
		return m.store(ctx, filename, src)
	}
	return "/uploads/" + filename, nil
}
func (m *mockStorage) Delete(ctx context.Context, storagePath string) error {
	if m.delete != nil {
		return m.delete(ctx, storagePath)
	}
	return nil
}
func (m *mockStorage) URL(storagePath string) string {
	return "/uploads/" + storagePath
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestMediaService(repo MediaRepository, storage StorageBackend) MediaService {
	return NewService(repo, storage, sdk.NewHookRegistry(), zerolog.Nop())
}

func defaultStorage() *mockStorage {
	return &mockStorage{}
}

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

func TestMediaService_Upload_MissingFilename(t *testing.T) {
	_, err := newTestMediaService(&mockMediaRepo{}, defaultStorage()).
		Upload(context.Background(), "", "image/jpeg", "", 0, strings.NewReader("data"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing filename, got %v", err)
	}
}

func TestMediaService_Upload_MissingMimeType(t *testing.T) {
	_, err := newTestMediaService(&mockMediaRepo{}, defaultStorage()).
		Upload(context.Background(), "photo.jpg", "", "", 0, strings.NewReader("data"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing mime_type, got %v", err)
	}
}

func TestMediaService_Upload_StorageError_CleanupSkipped(t *testing.T) {
	storageErr := errors.New("disk full")
	storage := &mockStorage{
		store: func(_ context.Context, _ string, _ io.Reader) (string, error) {
			return "", storageErr
		},
	}
	_, err := newTestMediaService(&mockMediaRepo{}, storage).
		Upload(context.Background(), "photo.jpg", "image/jpeg", "", 100, strings.NewReader("data"))
	if !errors.Is(err, storageErr) {
		t.Errorf("expected storageErr, got %v", err)
	}
}

func TestMediaService_Upload_RepoPersistError_DeletesStoredFile(t *testing.T) {
	repoErr := errors.New("db error")
	storageDeleteCalled := false
	storage := &mockStorage{
		delete: func(_ context.Context, _ string) error {
			storageDeleteCalled = true
			return nil
		},
	}
	repo := &mockMediaRepo{
		create: func(_ context.Context, _ *Media) error {
			return repoErr
		},
	}
	_, err := newTestMediaService(repo, storage).
		Upload(context.Background(), "photo.jpg", "image/jpeg", "", 100, strings.NewReader("data"))
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repoErr, got %v", err)
	}
	if !storageDeleteCalled {
		t.Error("expected storage.Delete to be called for cleanup after repo failure")
	}
}

func TestMediaService_Upload_Success(t *testing.T) {
	var saved *Media
	repo := &mockMediaRepo{
		create: func(_ context.Context, m *Media) error {
			saved = m
			return nil
		},
	}
	got, err := newTestMediaService(repo, defaultStorage()).
		Upload(context.Background(), "photo.jpg", "image/jpeg", "A photo", 1024, strings.NewReader("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if got.Filename != "photo.jpg" {
		t.Errorf("filename: got %q, want photo.jpg", got.Filename)
	}
	if got.MimeType != "image/jpeg" {
		t.Errorf("mime_type: got %q, want image/jpeg", got.MimeType)
	}
	if got.AltText != "A photo" {
		t.Errorf("alt_text: got %q, want 'A photo'", got.AltText)
	}
	if got.StoragePath == "" {
		t.Error("StoragePath should be set after upload")
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestMediaService_GetByID_NotFound(t *testing.T) {
	_, err := newTestMediaService(&mockMediaRepo{}, defaultStorage()).
		GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMediaService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockMediaRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Media, error) {
			return &Media{ID: id, Filename: "img.png"}, nil
		},
	}
	got, err := newTestMediaService(repo, defaultStorage()).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %s, want %s", got.ID, id)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestMediaService_Delete_NotFound(t *testing.T) {
	err := newTestMediaService(&mockMediaRepo{}, defaultStorage()).
		Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMediaService_Delete_Success_CallsStorageDelete(t *testing.T) {
	id := uuid.New()
	storagePath := "/uploads/img.jpg"
	storageDeleteCalled := false

	repo := &mockMediaRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Media, error) {
			return &Media{ID: id, StoragePath: storagePath}, nil
		},
	}
	storage := &mockStorage{
		delete: func(_ context.Context, path string) error {
			if path != storagePath {
				return errors.New("wrong path")
			}
			storageDeleteCalled = true
			return nil
		},
	}
	if err := newTestMediaService(repo, storage).Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !storageDeleteCalled {
		t.Error("expected storage.Delete to be called")
	}
}
