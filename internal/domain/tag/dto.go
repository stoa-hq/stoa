package tag

// CreateTagRequest is the request body for creating a tag.
type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
	Slug string `json:"slug" validate:"required,min=1,max=255"`
}

// UpdateTagRequest is the request body for updating a tag.
type UpdateTagRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
	Slug string `json:"slug" validate:"required,min=1,max=255"`
}

// ListTagsRequest holds query parameters for the List endpoint.
type ListTagsRequest struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Name  string `json:"name,omitempty"`
}
