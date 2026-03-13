package sdk

import (
	"fmt"
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
	Fields    []UISchemaField `json:"fields"`
	SubmitURL string          `json:"submit_url,omitempty"`
	LoadURL   string          `json:"load_url,omitempty"`
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

// validateURL rejects path traversal and absolute URLs outside the expected scope.
func validateURL(u string) error {
	if u == "" {
		return nil
	}
	if strings.Contains(u, "..") {
		return fmt.Errorf("path traversal not allowed: %q", u)
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return fmt.Errorf("absolute URLs not allowed: %q", u)
	}
	return nil
}
