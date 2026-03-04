package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKey struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	KeyHash     string       `json:"-"`
	Permissions []Permission `json:"permissions"`
	Active      bool         `json:"active"`
	LastUsedAt  *time.Time   `json:"last_used_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type APIKeyManager struct {
	pool *pgxpool.Pool
}

func NewAPIKeyManager(pool *pgxpool.Pool) *APIKeyManager {
	return &APIKeyManager{pool: pool}
}

func GenerateAPIKey() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generating API key: %w", err)
	}
	key := "ck_" + hex.EncodeToString(bytes)
	hash := hashKey(key)
	return key, hash, nil
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (m *APIKeyManager) Create(ctx context.Context, name string, permissions []Permission) (string, *APIKey, error) {
	key, hash, err := GenerateAPIKey()
	if err != nil {
		return "", nil, err
	}

	id := uuid.New()
	now := time.Now()

	permStrs := make([]string, len(permissions))
	for i, p := range permissions {
		permStrs[i] = string(p)
	}

	_, err = m.pool.Exec(ctx,
		`INSERT INTO api_keys (id, name, key_hash, permissions, active, created_at)
		 VALUES ($1, $2, $3, $4, true, $5)`,
		id, name, hash, permStrs, now)
	if err != nil {
		return "", nil, fmt.Errorf("creating API key: %w", err)
	}

	return key, &APIKey{
		ID:          id,
		Name:        name,
		Permissions: permissions,
		Active:      true,
		CreatedAt:   now,
	}, nil
}

func (m *APIKeyManager) Validate(ctx context.Context, key string) (*APIKey, error) {
	hash := hashKey(key)

	var apiKey APIKey
	var permStrs []string
	err := m.pool.QueryRow(ctx,
		`SELECT id, name, permissions, active, last_used_at, created_at
		 FROM api_keys WHERE key_hash = $1`, hash).
		Scan(&apiKey.ID, &apiKey.Name, &permStrs, &apiKey.Active, &apiKey.LastUsedAt, &apiKey.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("validating API key: %w", err)
	}

	if !apiKey.Active {
		return nil, fmt.Errorf("API key is inactive")
	}

	apiKey.Permissions = make([]Permission, len(permStrs))
	for i, s := range permStrs {
		apiKey.Permissions[i] = Permission(s)
	}

	// Update last used
	_, _ = m.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now(), apiKey.ID)

	return &apiKey, nil
}

func (m *APIKeyManager) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := m.pool.Exec(ctx, `UPDATE api_keys SET active = false WHERE id = $1`, id)
	return err
}
