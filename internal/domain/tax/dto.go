package tax

// CreateTaxRuleRequest is the request body for creating a tax rule.
type CreateTaxRuleRequest struct {
	Name        string `json:"name"         validate:"required,min=1,max=255"`
	Rate        int    `json:"rate"         validate:"required,min=0,max=100000"`
	CountryCode string `json:"country_code" validate:"required,len=2"`
	Type        string `json:"type"         validate:"required,oneof=standard reduced zero custom"`
}

// UpdateTaxRuleRequest is the request body for updating a tax rule.
type UpdateTaxRuleRequest struct {
	Name        string `json:"name"         validate:"required,min=1,max=255"`
	Rate        int    `json:"rate"         validate:"required,min=0,max=100000"`
	CountryCode string `json:"country_code" validate:"required,len=2"`
	Type        string `json:"type"         validate:"required,oneof=standard reduced zero custom"`
}

// ListTaxRulesRequest holds query parameters for the List endpoint.
type ListTaxRulesRequest struct {
	Page        int    `json:"page"`
	Limit       int    `json:"limit"`
	CountryCode string `json:"country_code"`
	Type        string `json:"type"`
}
