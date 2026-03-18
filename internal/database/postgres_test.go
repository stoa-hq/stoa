package database

import (
	"bytes"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

func TestNew_SSLModeDisableWarning(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	cfg := config.DatabaseConfig{
		URL: "postgres://stoa:secret@localhost:5432/stoa?sslmode=disable",
	}

	// New will fail (no DB running), but the warning is logged before connection
	_, _ = New(cfg, logger)

	if !bytes.Contains(buf.Bytes(), []byte("sslmode=disable")) {
		t.Error("expected warning about sslmode=disable in log output")
	}
}

func TestNew_SSLModeRequireNoWarning(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	cfg := config.DatabaseConfig{
		URL: "postgres://stoa:secret@localhost:5432/stoa?sslmode=require",
	}

	// New will fail (no DB running), but no sslmode warning should appear
	_, _ = New(cfg, logger)

	if bytes.Contains(buf.Bytes(), []byte("sslmode=disable")) {
		t.Error("unexpected sslmode=disable warning when using sslmode=require")
	}
}

func TestPgxTLSConfig_DisableSetsNil(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://stoa:secret@localhost:5432/stoa?sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if poolCfg.ConnConfig.TLSConfig != nil {
		t.Error("expected TLSConfig to be nil for sslmode=disable")
	}
}

func TestPgxTLSConfig_RequireSetsNonNil(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://stoa:secret@localhost:5432/stoa?sslmode=require")
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if poolCfg.ConnConfig.TLSConfig == nil {
		t.Error("expected TLSConfig to be non-nil for sslmode=require")
	}
}
