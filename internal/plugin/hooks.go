package plugin

import "github.com/stoa-hq/stoa/pkg/sdk"

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return sdk.NewHookRegistry()
}
