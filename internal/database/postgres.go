package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

type DB struct {
	Pool   *pgxpool.Pool
	logger zerolog.Logger
}

func New(cfg config.DatabaseConfig, logger zerolog.Logger) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	if poolCfg.ConnConfig.TLSConfig == nil {
		logger.Warn().Msg("database connection uses sslmode=disable — not recommended for production")
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	logger.Info().Msg("database connection established")

	return &DB{Pool: pool, logger: logger}, nil
}

func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func (db *DB) Close() {
	db.Pool.Close()
	db.logger.Info().Msg("database connection closed")
}
