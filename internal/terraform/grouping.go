package terraform

import (
	"strings"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// GroupByModule delegates to the SDK implementation.
var GroupByModule = sdk.GroupByModule

// ExtractModule extracts the module path prefix from a resource address.
// For example: "module.vpc.aws_subnet.main" returns "module.vpc",
// "module.vpc.module.subnets.aws_subnet.a" returns "module.vpc.module.subnets",
// and "aws_instance.web" returns "" (root module).
func ExtractModule(address string) string {
	parts := strings.Split(address, ".")
	lastModIdx := -1

	for i, part := range parts {
		if part == "module" {
			lastModIdx = i
		}
	}

	if lastModIdx == -1 {
		return ""
	}

	end := lastModIdx + 2
	if end > len(parts) {
		end = len(parts)
	}
	return strings.Join(parts[:end], ".")
}
