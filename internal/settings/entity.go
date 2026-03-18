package settings

import "time"

// StoreSettings represents the singleton store configuration row.
type StoreSettings struct {
	StoreName        string  `json:"store_name"`
	StoreDescription string  `json:"store_description"`
	LogoURL          *string `json:"logo_url"`
	FaviconURL       *string `json:"favicon_url"`
	ContactEmail     *string `json:"contact_email"`
	Currency         string  `json:"currency"`
	Country          *string `json:"country"`
	Timezone         string  `json:"timezone"`
	CopyrightText    string  `json:"copyright_text"`
	MaintenanceMode  bool    `json:"maintenance_mode"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
