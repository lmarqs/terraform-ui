package extension

import "github.com/lmarqs/terraform-ui/internal/terraform"

// Context provides shared state that extensions can read.
type Context struct {
	Dir       string
	Workspace string
	Service   terraform.Service
}
