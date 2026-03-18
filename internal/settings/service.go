package settings

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
)

// Sentinel errors for the settings domain.
var (
	ErrNotFound     = errors.New("store settings not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Service provides business logic for store settings.
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new settings Service.
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Get returns the current store settings. If no row exists, returns defaults.
func (s *Service) Get(ctx context.Context) (*StoreSettings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return defaultSettings(), nil
		}
		s.logger.Error().Err(err).Msg("settings: Get")
		return nil, err
	}
	return settings, nil
}

// Update validates and persists the given settings.
func (s *Service) Update(ctx context.Context, settings *StoreSettings) (*StoreSettings, error) {
	if settings.StoreName == "" {
		return nil, ErrInvalidInput
	}
	if settings.Currency == "" {
		settings.Currency = "EUR"
	}
	if settings.Timezone == "" {
		settings.Timezone = "UTC"
	}

	result, err := s.repo.Upsert(ctx, settings)
	if err != nil {
		s.logger.Error().Err(err).Msg("settings: Update")
		return nil, err
	}
	return result, nil
}

func defaultSettings() *StoreSettings {
	return &StoreSettings{
		StoreName:        "Stoa",
		StoreDescription: "",
		Currency:         "EUR",
		Timezone:         "UTC",
		MaintenanceMode:  false,
	}
}
