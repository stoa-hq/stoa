package settings

import "context"

// Repository defines the persistence contract for store settings.
type Repository interface {
	Get(ctx context.Context) (*StoreSettings, error)
	Upsert(ctx context.Context, s *StoreSettings) (*StoreSettings, error)
}
