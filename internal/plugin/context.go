package plugin

import "github.com/lmarqs/terraform-ui/pkg/sdk"

// Context is a type alias for the SDK Context type. Internal packages
// can continue to reference plugin.Context. New code should prefer pkg/sdk.
type Context = sdk.Context
