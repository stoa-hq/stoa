package media

// UploadMediaRequest describes the fields available in the multipart upload form.
// The actual file bytes are read from the "file" form field; these fields come
// from additional form values.
type UploadMediaRequest struct {
	AltText string `json:"alt_text"`
}

// ListMediaRequest holds query parameters for the List endpoint.
type ListMediaRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	MimeType string `json:"mime_type,omitempty"`
}
