package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
)

type Migrator struct {
	m      *migrate.Migrate
	logger zerolog.Logger
}

func NewMigrator(dbURL, migrationsPath string, logger zerolog.Logger) (*Migrator, error) {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dbURL,
	)
	if err != nil {
		return nil, fmt.Errorf("creating migrator: %w", err)
	}

	return &Migrator{m: m, logger: logger}, nil
}

func (m *Migrator) Up() error {
	m.logger.Info().Msg("running migrations up")
	if err := m.m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info().Msg("no pending migrations")
			return nil
		}
		return fmt.Errorf("running migrations up: %w", err)
	}
	m.logger.Info().Msg("migrations completed")
	return nil
}

func (m *Migrator) Down() error {
	m.logger.Info().Msg("rolling back last migration")
	if err := m.m.Steps(-1); err != nil {
		return fmt.Errorf("rolling back migration: %w", err)
	}
	m.logger.Info().Msg("rollback completed")
	return nil
}

func (m *Migrator) Version() (uint, bool, error) {
	return m.m.Version()
}

func (m *Migrator) Close() {
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		m.logger.Error().Err(srcErr).Msg("closing migration source")
	}
	if dbErr != nil {
		m.logger.Error().Err(dbErr).Msg("closing migration database")
	}
}
