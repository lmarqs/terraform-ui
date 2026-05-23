package plugin

import "github.com/lmarqs/terraform-ui/pkg/sdk"

// PluginDeps is a type alias for the SDK PluginDeps type. Internal packages
// can continue to reference plugin.PluginDeps. New code should prefer pkg/sdk.
type PluginDeps = sdk.PluginDeps
