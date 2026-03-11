package plugin

import "github.com/stoa-hq/stoa/pkg/sdk"

// Plugin re-exports the SDK plugin interface.
type Plugin = sdk.Plugin

// AppContext re-exports the SDK app context.
type AppContext = sdk.AppContext

// HookRegistry re-exports the SDK hook registry.
type HookRegistry = sdk.HookRegistry

// HookHandler re-exports the SDK hook handler type.
type HookHandler = sdk.HookHandler

// HookEvent re-exports the SDK hook event type.
type HookEvent = sdk.HookEvent
