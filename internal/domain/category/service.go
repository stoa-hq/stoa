package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// Service implements business logic for the category domain.
type Service struct {
	repo   CategoryRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService constructs a Service.
func NewService(repo CategoryRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		hooks:  hooks,
		logger: logger,
	}
}

// GetByID returns a single category, including its translations.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	cat, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("service.GetByID: %w", err)
	}
	return cat, nil
}

// List returns a paginated slice of categories plus the total count.
func (s *Service) List(ctx context.Context, filter CategoryFilter) ([]Category, int, error) {
	cats, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("service.List: %w", err)
	}
	return cats, total, nil
}

// GetTree returns the active category hierarchy for the given locale.
func (s *Service) GetTree(ctx context.Context, locale string) ([]Category, error) {
	if locale == "" {
		locale = "de-DE"
	}
	tree, err := s.repo.FindTree(ctx, locale)
	if err != nil {
		return nil, fmt.Errorf("service.GetTree: %w", err)
	}
	return tree, nil
}

// GetBySlug returns the category matching slug+locale.
func (s *Service) GetBySlug(ctx context.Context, slug, locale string) (*Category, error) {
	cat, err := s.repo.FindBySlug(ctx, slug, locale)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("service.GetBySlug: %w", err)
	}
	return cat, nil
}

// Create persists a new category after running before/after hooks.
func (s *Service) Create(ctx context.Context, cat *Category) error {
	beforeEvent := &sdk.HookEvent{
		Name:   sdk.HookBeforeCategoryCreate,
		Entity: cat,
	}
	if err := s.hooks.Dispatch(ctx, beforeEvent); err != nil {
		return fmt.Errorf("service.Create before_create hook: %w", err)
	}

	if err := s.repo.Create(ctx, cat); err != nil {
		return fmt.Errorf("service.Create repo: %w", err)
	}

	afterEvent := &sdk.HookEvent{
		Name:   sdk.HookAfterCategoryCreate,
		Entity: cat,
	}
	if err := s.hooks.Dispatch(ctx, afterEvent); err != nil {
		s.logger.Warn().Err(err).Msg("after_create hook error")
	}

	s.logger.Info().Stringer("id", cat.ID).Msg("category created")
	return nil
}

// Update persists changes to an existing category after running before/after hooks.
func (s *Service) Update(ctx context.Context, cat *Category) error {
	existing, err := s.repo.FindByID(ctx, cat.ID)
	if err != nil {
		return fmt.Errorf("service.Update find existing: %w", err)
	}

	beforeEvent := &sdk.HookEvent{
		Name:   sdk.HookBeforeCategoryUpdate,
		Entity: cat,
		Changes: map[string]interface{}{
			"previous": existing,
		},
	}
	if err := s.hooks.Dispatch(ctx, beforeEvent); err != nil {
		return fmt.Errorf("service.Update before_update hook: %w", err)
	}

	if err := s.repo.Update(ctx, cat); err != nil {
		return fmt.Errorf("service.Update repo: %w", err)
	}

	afterEvent := &sdk.HookEvent{
		Name:   sdk.HookAfterCategoryUpdate,
		Entity: cat,
		Changes: map[string]interface{}{
			"previous": existing,
		},
	}
	if err := s.hooks.Dispatch(ctx, afterEvent); err != nil {
		s.logger.Warn().Err(err).Msg("after_update hook error")
	}

	s.logger.Info().Stringer("id", cat.ID).Msg("category updated")
	return nil
}

// Delete removes a category by ID after running before/after hooks.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("service.Delete find existing: %w", err)
	}

	beforeEvent := &sdk.HookEvent{
		Name:   sdk.HookBeforeCategoryDelete,
		Entity: existing,
	}
	if err := s.hooks.Dispatch(ctx, beforeEvent); err != nil {
		return fmt.Errorf("service.Delete before_delete hook: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("service.Delete repo: %w", err)
	}

	afterEvent := &sdk.HookEvent{
		Name:   sdk.HookAfterCategoryDelete,
		Entity: existing,
	}
	if err := s.hooks.Dispatch(ctx, afterEvent); err != nil {
		s.logger.Warn().Err(err).Msg("after_delete hook error")
	}

	s.logger.Info().Stringer("id", id).Msg("category deleted")
	return nil
}
