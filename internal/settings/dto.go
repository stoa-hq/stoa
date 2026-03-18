package settings

// UpdateSettingsRequest is the request body for updating store settings.
type UpdateSettingsRequest struct {
	StoreName        string  `json:"store_name" validate:"required,min=1,max=255"`
	StoreDescription string  `json:"store_description" validate:"max=1000"`
	LogoURL          *string `json:"logo_url"`
	FaviconURL       *string `json:"favicon_url"`
	ContactEmail     *string `json:"contact_email" validate:"omitempty,email"`
	Currency         string  `json:"currency" validate:"required,len=3"`
	Country          *string `json:"country" validate:"omitempty,len=2"`
	Timezone         string  `json:"timezone" validate:"required,min=1,max=64"`
	CopyrightText    string  `json:"copyright_text" validate:"max=500"`
	MaintenanceMode  bool    `json:"maintenance_mode"`
}
