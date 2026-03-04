package tax

import (
	"time"

	"github.com/google/uuid"
)

// TaxRule represents a tax rule applied to products or orders.
// The Rate field uses integer basis points where 1900 equals 19.00%.
type TaxRule struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Rate        int       `json:"rate"`
	CountryCode string    `json:"country_code"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RatePercent returns the rate as a float64 percentage (e.g. 19.0 for 1900).
func (t *TaxRule) RatePercent() float64 {
	return float64(t.Rate) / 100.0
}
