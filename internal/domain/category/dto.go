package category

import "github.com/google/uuid"

// -------------------------------------------------------------------------
// Request DTOs (inbound)
// -------------------------------------------------------------------------

// TranslationInput carries locale-specific data for create/update requests.
type TranslationInput struct {
	Locale      string `json:"locale"       validate:"required,bcp47_language_tag|min=2,max=10"`
	Name        string `json:"name"         validate:"required,min=1,max=255"`
	Description string `json:"description"  validate:"max=5000"`
	Slug        string `json:"slug"         validate:"required,min=1,max=255"`
}

// CreateCategoryRequest is the payload for POST /admin/categories.
type CreateCategoryRequest struct {
	ParentID     *uuid.UUID             `json:"parent_id"`
	Position     int                    `json:"position"      validate:"min=0"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Translations []TranslationInput     `json:"translations"  validate:"required,min=1,dive"`
}

// UpdateCategoryRequest is the payload for PUT /admin/categories/{id}.
type UpdateCategoryRequest struct {
	ParentID     *uuid.UUID             `json:"parent_id"`
	Position     int                    `json:"position"      validate:"min=0"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Translations []TranslationInput     `json:"translations"  validate:"required,min=1,dive"`
}

// ListCategoryRequest captures query parameters for GET /admin/categories.
type ListCategoryRequest struct {
	Page     int        `schema:"page"      validate:"min=1"`
	Limit    int        `schema:"limit"     validate:"min=1,max=200"`
	ParentID *uuid.UUID `schema:"parent_id"`
	Active   *bool      `schema:"active"`
}

// TreeRequest captures query parameters for GET /store/categories/tree.
type TreeRequest struct {
	Locale string `schema:"locale" validate:"omitempty,min=2,max=10"`
}

// -------------------------------------------------------------------------
// Response DTOs (outbound)
// -------------------------------------------------------------------------

// TranslationResponse is the public representation of a CategoryTranslation.
type TranslationResponse struct {
	Locale      string `json:"locale"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Slug        string `json:"slug"`
}

// CategoryResponse is the public representation of a Category.
type CategoryResponse struct {
	ID           uuid.UUID              `json:"id"`
	ParentID     *uuid.UUID             `json:"parent_id,omitempty"`
	Position     int                    `json:"position"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	Translations []TranslationResponse  `json:"translations,omitempty"`
	Children     []CategoryResponse     `json:"children,omitempty"`
}

// ToResponse converts a Category entity into a CategoryResponse.
func ToResponse(c *Category) CategoryResponse {
	resp := CategoryResponse{
		ID:           c.ID,
		ParentID:     c.ParentID,
		Position:     c.Position,
		Active:       c.Active,
		CustomFields: c.CustomFields,
		CreatedAt:    c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	for _, tr := range c.Translations {
		resp.Translations = append(resp.Translations, TranslationResponse{
			Locale:      tr.Locale,
			Name:        tr.Name,
			Description: tr.Description,
			Slug:        tr.Slug,
		})
	}
	for _, child := range c.Children {
		childCopy := child
		resp.Children = append(resp.Children, ToResponse(&childCopy))
	}
	return resp
}

// ToResponseList converts a slice of Category entities.
func ToResponseList(cats []Category) []CategoryResponse {
	out := make([]CategoryResponse, len(cats))
	for i := range cats {
		out[i] = ToResponse(&cats[i])
	}
	return out
}

// ToEntity maps a CreateCategoryRequest onto a fresh Category entity.
func (req *CreateCategoryRequest) ToEntity() *Category {
	cat := &Category{
		ParentID:     req.ParentID,
		Position:     req.Position,
		Active:       req.Active,
		CustomFields: req.CustomFields,
	}
	for _, tr := range req.Translations {
		cat.Translations = append(cat.Translations, CategoryTranslation{
			Locale:      tr.Locale,
			Name:        tr.Name,
			Description: tr.Description,
			Slug:        tr.Slug,
		})
	}
	return cat
}

// ApplyTo applies an UpdateCategoryRequest onto an existing Category entity
// (preserving the ID and timestamps which will be refreshed by the repository).
func (req *UpdateCategoryRequest) ApplyTo(cat *Category) {
	cat.ParentID = req.ParentID
	cat.Position = req.Position
	cat.Active = req.Active
	cat.CustomFields = req.CustomFields
	cat.Translations = nil
	for _, tr := range req.Translations {
		cat.Translations = append(cat.Translations, CategoryTranslation{
			CategoryID:  cat.ID,
			Locale:      tr.Locale,
			Name:        tr.Name,
			Description: tr.Description,
			Slug:        tr.Slug,
		})
	}
}
