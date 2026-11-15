package ui

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
)

func (a App) riskPolicyDiagnostic() string {
	current := a.holder.current
	selection, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return err.Error()
	}
	// The evaluator owns embedded modules. Built-in selection needs no disk
	// acquisition; custom selection must never recover through that policy.
	if selection.Path == "" {
		return ""
	}
	if _, err := riskpolicy.LoadSource(selection, nil); err != nil {
		return err.Error()
	}
	return ""
}
