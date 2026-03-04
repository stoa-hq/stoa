package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type postgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL-backed AuditLogRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) AuditLogRepository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) Create(ctx context.Context, a *AuditLog) error {
	const q = `
		INSERT INTO audit_logs (id, user_id, user_type, action, entity_type, entity_id, changes, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	changesJSON, err := json.Marshal(a.Changes)
	if err != nil {
		return fmt.Errorf("audit: Create marshal changes: %w", err)
	}

	_, err = r.db.Exec(ctx, q,
		a.ID, a.UserID, a.UserType, a.Action, a.EntityType, a.EntityID,
		changesJSON, a.IPAddress, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	args := []interface{}{}
	where := "WHERE 1=1"
	idx := 1

	if filter.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", idx)
		args = append(args, *filter.UserID)
		idx++
	}
	if filter.EntityType != "" {
		where += fmt.Sprintf(" AND entity_type = $%d", idx)
		args = append(args, filter.EntityType)
		idx++
	}
	if filter.EntityID != nil {
		where += fmt.Sprintf(" AND entity_id = $%d", idx)
		args = append(args, *filter.EntityID)
		idx++
	}
	if filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", idx)
		args = append(args, filter.Action)
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, user_id, user_type, action, entity_type, entity_id, changes, ip_address::text, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: FindAll: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var a AuditLog
		var changesRaw []byte
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.UserType, &a.Action, &a.EntityType, &a.EntityID,
			&changesRaw, &a.IPAddress, &a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("audit: FindAll scan: %w", err)
		}
		if len(changesRaw) > 0 {
			if err := json.Unmarshal(changesRaw, &a.Changes); err != nil {
				return nil, 0, fmt.Errorf("audit: FindAll unmarshal changes: %w", err)
			}
		}
		logs = append(logs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit: FindAll rows: %w", err)
	}
	return logs, total, nil
}
