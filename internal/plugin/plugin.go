package plugin

import "github.com/lmarqs/terraform-ui/pkg/sdk"

// Plugin is a type alias for the SDK Plugin interface. Internal packages
// can continue to reference plugin.Plugin. New code should prefer pkg/sdk.
type Plugin = sdk.Plugin

// PluginFactory is a type alias for the SDK PluginFactory type.
type PluginFactory = sdk.PluginFactory
