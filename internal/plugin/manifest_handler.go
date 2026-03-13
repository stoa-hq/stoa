package plugin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ManifestHandler serves the plugin UI extension manifest.
type ManifestHandler struct {
	registry *Registry
}

// NewManifestHandler creates a handler that reads extensions from the registry.
func NewManifestHandler(registry *Registry) *ManifestHandler {
	return &ManifestHandler{registry: registry}
}

type manifestResponse struct {
	Data manifestData `json:"data"`
}

type manifestData struct {
	Extensions []sdk.UIExtension `json:"extensions"`
}

// StoreManifest returns only extensions with storefront:* slots.
func (h *ManifestHandler) StoreManifest(w http.ResponseWriter, r *http.Request) {
	h.writeManifest(w, "storefront:")
}

// AdminManifest returns only extensions with admin:* slots.
func (h *ManifestHandler) AdminManifest(w http.ResponseWriter, r *http.Request) {
	h.writeManifest(w, "admin:")
}

func (h *ManifestHandler) writeManifest(w http.ResponseWriter, prefix string) {
	all := h.registry.UIExtensions()
	filtered := make([]sdk.UIExtension, 0, len(all))
	for _, ext := range all {
		if strings.HasPrefix(ext.Slot, prefix) {
			filtered = append(filtered, ext)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifestResponse{
		Data: manifestData{Extensions: filtered},
	})
}
