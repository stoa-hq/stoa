package tag

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// Sentinel errors for the tag domain.
var (
	ErrNotFound     = errors.New("tag not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Hook name constants for the tag domain.
const (
	HookBeforeTagCreate = "tag.before_create"
	HookAfterTagCreate  = "tag.after_create"
	HookBeforeTagUpdate = "tag.before_update"
	HookAfterTagUpdate  = "tag.after_update"
	HookBeforeTagDelete = "tag.before_delete"
	HookAfterTagDelete  = "tag.after_delete"
)

// TagService defines the business-logic interface for the tag domain.
type TagService interface {
	List(ctx context.Context, filter TagFilter) ([]Tag, int, error)
	Create(ctx context.Context, t *Tag) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tag, error)
	Update(ctx context.Context, t *Tag) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo   TagRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new TagService with the given repository and optional hook registry.
func NewService(repo TagRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) TagService {
	return &service{repo: repo, hooks: hooks, logger: logger}
}

func (s *service) List(ctx context.Context, filter TagFilter) ([]Tag, int, error) {
	tags, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("tag: List")
		return nil, 0, err
	}
	return tags, total, nil
}

func (s *service) Create(ctx context.Context, t *Tag) error {
	if t.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if t.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidInput)
	}
	t.ID = uuid.New()

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTagCreate,
			Entity: t,
		}); err != nil {
			return fmt.Errorf("tag: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, t); err != nil {
		s.logger.Error().Err(err).Msg("tag: Create")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTagCreate,
			Entity: t,
		})
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Tag, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tag: GetByID")
		}
		return nil, err
	}
	return t, nil
}

func (s *service) Update(ctx context.Context, t *Tag) error {
	if t.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if t.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidInput)
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTagUpdate,
			Entity: t,
		}); err != nil {
			return fmt.Errorf("tag: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, t); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tag: Update")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTagUpdate,
			Entity: t,
		})
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTagDelete,
			Entity: &Tag{ID: id},
		}); err != nil {
			return fmt.Errorf("tag: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tag: Delete")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTagDelete,
			Entity: &Tag{ID: id},
		})
	}
	return nil
}
