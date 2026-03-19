package sdk

import (
	"fmt"
	"html"
	"strings"
)

// UIPlugin is an optional interface that plugins can implement to declare
// frontend UI extensions (schema-based forms or web components).
type UIPlugin interface {
	Plugin
	UIExtensions() []UIExtension
}

// UIExtension describes a single UI slot extension provided by a plugin.
type UIExtension struct {
	ID        string       `json:"id"`
	Slot      string       `json:"slot"`
	Type      string       `json:"type"` // "schema" or "component"
	Schema    *UISchema    `json:"schema,omitempty"`
	Component *UIComponent `json:"component,omitempty"`
}

// UISchema defines a form rendered from field descriptors.
type UISchema struct {
	Title          map[string]string `json:"title,omitempty"`
	Description    map[string]string `json:"description,omitempty"`
	SubmitLabel    map[string]string `json:"submit_label,omitempty"`
	SuccessMessage map[string]string `json:"success_message,omitempty"`
	Fields         []UISchemaField   `json:"fields"`
	SubmitURL      string            `json:"submit_url,omitempty"`
	LoadURL        string            `json:"load_url,omitempty"`
}

// UISchemaField describes a single form field.
type UISchemaField struct {
	Key         string            `json:"key"`
	Type        string            `json:"type"` // "text","password","toggle","select","number","textarea"
	Label       map[string]string `json:"label"`
	Placeholder map[string]string `json:"placeholder,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Options     []UISelectOption  `json:"options,omitempty"`
	HelpText    map[string]string `json:"help_text,omitempty"`
}

// UISelectOption is a value/label pair for select fields.
type UISelectOption struct {
	Value string            `json:"value"`
	Label map[string]string `json:"label"`
}

// UIComponent describes a web component loaded from a plugin's embedded assets.
type UIComponent struct {
	TagName         string   `json:"tag_name"`
	ScriptURL       string   `json:"script_url"`
	Integrity       string   `json:"integrity"`
	ExternalScripts []string `json:"external_scripts,omitempty"`
	StyleURL        string   `json:"style_url,omitempty"`
}

// validSlotPrefixes lists the allowed slot prefixes.
var validSlotPrefixes = []string{
	"storefront:",
	"admin:",
}

// validFieldTypes lists allowed UISchemaField.Type values.
var validFieldTypes = map[string]bool{
	"text":     true,
	"password": true,
	"toggle":   true,
	"select":   true,
	"number":   true,
	"textarea": true,
}

// ValidateUIExtension checks that an extension is well-formed and follows
// naming conventions. pluginName is the plugin's Name() value.
func ValidateUIExtension(pluginName string, ext UIExtension) error {
	if ext.ID == "" {
		return fmt.Errorf("ui extension: id must not be empty")
	}

	// Slot must start with a valid prefix.
	validSlot := false
	for _, prefix := range validSlotPrefixes {
		if strings.HasPrefix(ext.Slot, prefix) {
			validSlot = true
			break
		}
	}
	if !validSlot {
		return fmt.Errorf("ui extension %q: slot %q must start with storefront: or admin:", ext.ID, ext.Slot)
	}

	switch ext.Type {
	case "schema":
		if ext.Schema == nil {
			return fmt.Errorf("ui extension %q: type is schema but schema is nil", ext.ID)
		}
		for i, f := range ext.Schema.Fields {
			if !validFieldTypes[f.Type] {
				return fmt.Errorf("ui extension %q: field[%d] type %q is not valid", ext.ID, i, f.Type)
			}
			if f.Key == "" {
				return fmt.Errorf("ui extension %q: field[%d] key must not be empty", ext.ID, i)
			}
		}
		if err := validateURL(ext.Schema.SubmitURL); err != nil {
			return fmt.Errorf("ui extension %q: submit_url: %w", ext.ID, err)
		}
		if err := validateURL(ext.Schema.LoadURL); err != nil {
			return fmt.Errorf("ui extension %q: load_url: %w", ext.ID, err)
		}

	case "component":
		if ext.Component == nil {
			return fmt.Errorf("ui extension %q: type is component but component is nil", ext.ID)
		}
		expectedPrefix := "stoa-" + pluginName + "-"
		if !strings.HasPrefix(ext.Component.TagName, expectedPrefix) {
			return fmt.Errorf("ui extension %q: tag_name %q must start with %q", ext.ID, ext.Component.TagName, expectedPrefix)
		}
		if err := validateURL(ext.Component.ScriptURL); err != nil {
			return fmt.Errorf("ui extension %q: script_url: %w", ext.ID, err)
		}
		if ext.Component.StyleURL != "" {
			if err := validateURL(ext.Component.StyleURL); err != nil {
				return fmt.Errorf("ui extension %q: style_url: %w", ext.ID, err)
			}
		}

	default:
		return fmt.Errorf("ui extension %q: type must be schema or component, got %q", ext.ID, ext.Type)
	}

	return nil
}

// sanitizeI18n escapes HTML entities in all values of an i18n string map.
// This provides defense-in-depth against stored XSS via plugin-provided strings.
func sanitizeI18n(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = html.EscapeString(v)
	}
	return out
}

// SanitizeUIExtension escapes HTML in all plugin-provided i18n string values.
// Call after validation to ensure safe rendering regardless of the frontend.
func SanitizeUIExtension(ext *UIExtension) {
	if ext.Schema != nil {
		ext.Schema.Title = sanitizeI18n(ext.Schema.Title)
		ext.Schema.Description = sanitizeI18n(ext.Schema.Description)
		ext.Schema.SubmitLabel = sanitizeI18n(ext.Schema.SubmitLabel)
		ext.Schema.SuccessMessage = sanitizeI18n(ext.Schema.SuccessMessage)
		for i := range ext.Schema.Fields {
			ext.Schema.Fields[i].Label = sanitizeI18n(ext.Schema.Fields[i].Label)
			ext.Schema.Fields[i].Placeholder = sanitizeI18n(ext.Schema.Fields[i].Placeholder)
			ext.Schema.Fields[i].HelpText = sanitizeI18n(ext.Schema.Fields[i].HelpText)
			for j := range ext.Schema.Fields[i].Options {
				ext.Schema.Fields[i].Options[j].Label = sanitizeI18n(ext.Schema.Fields[i].Options[j].Label)
			}
		}
	}
}

// validateURL allows only relative paths starting with / (but not //).
// This blocks dangerous schemes (javascript:, data:, vbscript:), absolute URLs,
// protocol-relative URLs, and path traversal.
func validateURL(u string) error {
	if u == "" {
		return nil
	}
	if strings.Contains(u, "..") {
		return fmt.Errorf("path traversal not allowed: %q", u)
	}
	// Only allow relative paths starting with /
	// Block protocol-relative URLs (//), dangerous schemes, and anything not starting with /
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") {
		return fmt.Errorf("only relative paths starting with / are allowed: %q", u)
	}
	return nil
}
