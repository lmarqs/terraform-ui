package main

import (
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/spf13/cobra"
)

// bindLockFlags registers terraform's two state-locking flags on a command.
// The bool lands in a caller-owned variable rather than the plugin Input,
// because the plugin models locking as a tri-state: only lockModeFrom, which
// consults Changed(), can tell "the user asked for -lock=true" apart from "the
// user said nothing, keep whatever tfui.hcl resolved".
func bindLockFlags(c *cobra.Command, lock *bool, lockTimeout *string) {
	c.Flags().BoolVar(lock, "lock", true, "Lock state during the operation (--lock=false to skip)")
	c.Flags().StringVar(lockTimeout, "lock-timeout", "", "Duration to retry a state lock (e.g. 30s)")
}

// lockModeFrom maps the parsed --lock bool onto sdk.LockMode, returning
// LockDefault when the flag was never passed so the resolved config value
// survives.
func lockModeFrom(c *cobra.Command, lock bool) sdk.LockMode {
	if !c.Flags().Changed("lock") {
		return sdk.LockDefault
	}
	return sdk.LockModeFromPtr(&lock)
}
