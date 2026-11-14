//go:build !linux

package exec

// TerraformHasOwnProcessGroup reports whether terraform runs in its own
// process group, so a terminal's Ctrl+C reaches tfui alone. terraform-exec
// sets Setpgid only on Linux.
const TerraformHasOwnProcessGroup = false
