package plugin

import "github.com/lmarqs/terraform-ui/internal/terraform"

// Context provides shared state that plugins can read.
type Context struct {
	Dir       string
	Workspace string
	Service   terraform.Service
}
