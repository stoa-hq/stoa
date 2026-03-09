package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// Sentinel errors for the media domain.
var (
	ErrNotFound     = errors.New("media not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Hook name constants for the media domain.
const (
	HookBeforeMediaUpload = "media.before_upload"
	HookAfterMediaUpload  = "media.after_upload"
	HookBeforeMediaDelete = "media.before_delete"
	HookAfterMediaDelete  = "media.after_delete"
)

// StorageBackend is the interface for persisting file bytes.
// Implementations may write to disk, S3, GCS, etc.
type StorageBackend interface {
	// Store writes src to storage and returns the storage path.
	Store(ctx context.Context, filename string, src io.Reader) (storagePath string, err error)

	// Delete removes the file at the given storage path.
	Delete(ctx context.Context, storagePath string) error

	// URL returns the public URL for the given storage path.
	URL(storagePath string) string
}

// MediaService defines the business-logic interface for the media domain.
type MediaService interface {
	// Upload stores the file bytes via the StorageBackend and persists a Media record.
	Upload(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error)

	// List returns a paginated, filtered list of media records.
	List(ctx context.Context, filter MediaFilter) ([]Media, int, error)

	// GetByID retrieves a single media record.
	GetByID(ctx context.Context, id uuid.UUID) (*Media, error)

	// Delete removes the media record and its stored file.
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo    MediaRepository
	storage StorageBackend
	hooks   *sdk.HookRegistry
	logger  zerolog.Logger
}

// NewService creates a new MediaService.
func NewService(repo MediaRepository, storage StorageBackend, hooks *sdk.HookRegistry, logger zerolog.Logger) MediaService {
	return &service{repo: repo, storage: storage, hooks: hooks, logger: logger}
}

func (s *service) Upload(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error) {
	if filename == "" {
		return nil, fmt.Errorf("%w: filename is required", ErrInvalidInput)
	}
	if mimeType == "" {
		return nil, fmt.Errorf("%w: mime_type is required", ErrInvalidInput)
	}

	m := &Media{
		ID:        uuid.New(),
		Filename:  filename,
		MimeType:  mimeType,
		Size:      size,
		AltText:   altText,
		CreatedAt: time.Now().UTC(),
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeMediaUpload,
			Entity: m,
		}); err != nil {
			return nil, fmt.Errorf("media: before_upload hook: %w", err)
		}
	}

	storagePath, err := s.storage.Store(ctx, filename, src)
	if err != nil {
		s.logger.Error().Err(err).Msg("media: Upload storage")
		return nil, fmt.Errorf("media: Upload storage: %w", err)
	}
	m.StoragePath = storagePath
	m.URL = s.storage.URL(storagePath)

	if err := s.repo.Create(ctx, m); err != nil {
		s.logger.Error().Err(err).Msg("media: Upload persist")
		// Best-effort cleanup of the stored file.
		_ = s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterMediaUpload,
			Entity: m,
		})
	}
	return m, nil
}

func (s *service) List(ctx context.Context, filter MediaFilter) ([]Media, int, error) {
	items, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("media: List")
		return nil, 0, err
	}
	for i := range items {
		items[i].URL = s.storage.URL(items[i].StoragePath)
	}
	return items, total, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("media: GetByID")
		}
		return nil, err
	}
	m.URL = s.storage.URL(m.StoragePath)
	return m, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("media: Delete find")
		}
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeMediaDelete,
			Entity: m,
		}); err != nil {
			return fmt.Errorf("media: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("media: Delete repo")
		}
		return err
	}

	// Best-effort removal from storage; log but do not fail if it errors.
	if err := s.storage.Delete(ctx, m.StoragePath); err != nil {
		s.logger.Warn().Err(err).Str("storage_path", m.StoragePath).Msg("media: Delete storage cleanup failed")
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterMediaDelete,
			Entity: m,
		})
	}
	return nil
}
