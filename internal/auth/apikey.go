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
	CreatedBy   *uuid.UUID   `json:"created_by,omitempty"`
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

func (m *APIKeyManager) Create(ctx context.Context, name string, permissions []Permission, createdBy uuid.UUID) (string, *APIKey, error) {
	key, hash, err := GenerateAPIKey()
	if err != nil {
		return "", nil, err
	}

	id := uuid.New()
	now := time.Now().UTC()

	permStrs := make([]string, len(permissions))
	for i, p := range permissions {
		permStrs[i] = string(p)
	}

	_, err = m.pool.Exec(ctx,
		`INSERT INTO api_keys (id, name, key_hash, permissions, active, created_at, created_by)
		 VALUES ($1, $2, $3, $4, true, $5, $6)`,
		id, name, hash, permStrs, now, createdBy)
	if err != nil {
		return "", nil, fmt.Errorf("creating API key: %w", err)
	}

	return key, &APIKey{
		ID:          id,
		Name:        name,
		Permissions: permissions,
		Active:      true,
		CreatedBy:   &createdBy,
		CreatedAt:   now,
	}, nil
}

func (m *APIKeyManager) Validate(ctx context.Context, key string) (*APIKey, error) {
	hash := hashKey(key)

	var apiKey APIKey
	var permStrs []string
	err := m.pool.QueryRow(ctx,
		`SELECT id, name, permissions, active, last_used_at, created_at, created_by
		 FROM api_keys WHERE key_hash = $1`, hash).
		Scan(&apiKey.ID, &apiKey.Name, &permStrs, &apiKey.Active, &apiKey.LastUsedAt, &apiKey.CreatedAt, &apiKey.CreatedBy)
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
	_, _ = m.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), apiKey.ID)

	return &apiKey, nil
}

func (m *APIKeyManager) List(ctx context.Context, userID *uuid.UUID) ([]APIKey, error) {
	var query string
	var args []interface{}
	if userID != nil {
		query = `SELECT id, name, permissions, active, last_used_at, created_at, created_by
		         FROM api_keys WHERE created_by = $1 ORDER BY created_at DESC`
		args = append(args, *userID)
	} else {
		query = `SELECT id, name, permissions, active, last_used_at, created_at, created_by
		         FROM api_keys ORDER BY created_at DESC`
	}

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var permStrs []string
		if err := rows.Scan(&k.ID, &k.Name, &permStrs, &k.Active, &k.LastUsedAt, &k.CreatedAt, &k.CreatedBy); err != nil {
			return nil, fmt.Errorf("scanning API key: %w", err)
		}
		k.Permissions = make([]Permission, len(permStrs))
		for i, s := range permStrs {
			k.Permissions[i] = Permission(s)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (m *APIKeyManager) Revoke(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	if userID != nil {
		_, err := m.pool.Exec(ctx, `UPDATE api_keys SET active = false WHERE id = $1 AND created_by = $2`, id, *userID)
		return err
	}
	_, err := m.pool.Exec(ctx, `UPDATE api_keys SET active = false WHERE id = $1`, id)
	return err
}

func (m *APIKeyManager) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := m.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_keys WHERE created_by = $1 AND active = true`, userID).
		Scan(&count)
	return count, err
}

func (m *APIKeyManager) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	var apiKey APIKey
	var permStrs []string
	err := m.pool.QueryRow(ctx,
		`SELECT id, name, permissions, active, last_used_at, created_at, created_by
		 FROM api_keys WHERE id = $1`, id).
		Scan(&apiKey.ID, &apiKey.Name, &permStrs, &apiKey.Active, &apiKey.LastUsedAt, &apiKey.CreatedAt, &apiKey.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("getting API key: %w", err)
	}
	apiKey.Permissions = make([]Permission, len(permStrs))
	for i, s := range permStrs {
		apiKey.Permissions[i] = Permission(s)
	}
	return &apiKey, nil
}
