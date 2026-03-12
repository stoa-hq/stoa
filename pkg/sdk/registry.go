package sdk

import "sync"

var (
	globalMu      sync.Mutex
	globalPlugins []Plugin
)

// Register adds a plugin to the global registry.
// Call this from your plugin's init() function so that Stoa
// automatically initialises the plugin on startup.
//
//	func init() { sdk.Register(New()) }
func Register(p Plugin) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalPlugins = append(globalPlugins, p)
}

// RegisteredPlugins returns a snapshot of all globally registered plugins.
func RegisteredPlugins() []Plugin {
	globalMu.Lock()
	defer globalMu.Unlock()
	return append([]Plugin(nil), globalPlugins...)
}
