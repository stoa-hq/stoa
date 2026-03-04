package plugin

import "github.com/epoxx-arch/stoa/pkg/sdk"

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return sdk.NewHookRegistry()
}
