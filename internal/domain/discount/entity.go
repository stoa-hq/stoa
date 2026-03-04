package discount

import (
	"time"

	"github.com/google/uuid"
)

// Discount represents a promotional discount code.
// Value uses integer cents for fixed type (e.g. 500 = €5.00) or basis points for percentage type (e.g. 1000 = 10.00%).
type Discount struct {
	ID            uuid.UUID              `json:"id"`
	Code          string                 `json:"code"`
	Type          string                 `json:"type"` // percentage | fixed
	Value         int                    `json:"value"`
	MinOrderValue *int                   `json:"min_order_value,omitempty"`
	MaxUses       *int                   `json:"max_uses,omitempty"`
	UsedCount     int                    `json:"used_count"`
	ValidFrom     *time.Time             `json:"valid_from,omitempty"`
	ValidUntil    *time.Time             `json:"valid_until,omitempty"`
	Active        bool                   `json:"active"`
	Conditions    map[string]interface{} `json:"conditions,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}
