package media

import (
	"time"

	"github.com/google/uuid"
)

// Media represents an uploaded file stored in the system.
// Size is in bytes.
type Media struct {
	ID           uuid.UUID              `json:"id"`
	Filename     string                 `json:"filename"`
	MimeType     string                 `json:"mime_type"`
	Size         int64                  `json:"size"`
	StoragePath  string                 `json:"storage_path"`
	URL          string                 `json:"url,omitempty"`
	AltText      string                 `json:"alt_text"`
	Thumbnails   map[string]interface{} `json:"thumbnails,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}
