package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTokenNotFound = errors.New("refresh token not found")
	ErrTokenRevoked  = errors.New("refresh token revoked")
	ErrTokenReuse    = errors.New("refresh token reuse detected")
)

type RefreshTokenRecord struct {
	ID        uuid.UUID
	TokenID   string
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	Revoked   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

type RefreshTokenStore struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenStore(pool *pgxpool.Pool) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

// Store persists a new refresh token record.
func (s *RefreshTokenStore) Store(ctx context.Context, tokenID string, userID, familyID uuid.UUID, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (token_id, user_id, family_id, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		tokenID, userID, familyID, expiresAt)
	return err
}

// Consume looks up a refresh token by its JWT ID (jti), validates it, and marks
// it as revoked. If the token was already revoked, the entire token family is
// revoked (reuse detection) and ErrTokenReuse is returned.
func (s *RefreshTokenStore) Consume(ctx context.Context, tokenID string) (*RefreshTokenRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var rec RefreshTokenRecord
	err = tx.QueryRow(ctx,
		`SELECT id, token_id, user_id, family_id, revoked, expires_at, created_at
		 FROM refresh_tokens WHERE token_id = $1 FOR UPDATE`, tokenID).
		Scan(&rec.ID, &rec.TokenID, &rec.UserID, &rec.FamilyID, &rec.Revoked, &rec.ExpiresAt, &rec.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	if rec.Revoked {
		// Reuse detected — revoke all tokens in this family.
		_, _ = tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked = TRUE WHERE family_id = $1`, rec.FamilyID)
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, ErrTokenReuse
	}

	// Mark current token as revoked (consumed).
	_, err = tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1`, rec.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &rec, nil
}

// RevokeAllForUser revokes every refresh token belonging to the given user.
func (s *RefreshTokenStore) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`, userID)
	return err
}

// Cleanup removes expired tokens. Should be called periodically.
func (s *RefreshTokenStore) Cleanup(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
