package sdk

import (
	"fmt"
	"time"
)

// StalenessGuard checks if cached data is too old before destructive operations.
type StalenessGuard struct {
	Threshold time.Duration
}

// NewStalenessGuard creates a guard with the given threshold.
// If threshold is 0, defaults to 5 minutes.
func NewStalenessGuard(threshold time.Duration) *StalenessGuard {
	if threshold == 0 {
		threshold = 5 * time.Minute
	}
	return &StalenessGuard{Threshold: threshold}
}

// CheckState returns an InputRequest if the state is stale, or nil if fresh enough.
func (g *StalenessGuard) CheckState(state *TerraformState) *InputRequest {
	if state == nil {
		return &InputRequest{
			Mode:   InputRequestBool,
			Prompt: "No state loaded. Load state first? (y/n)",
		}
	}
	age := time.Since(state.LastRefreshed)
	if age > g.Threshold {
		return &InputRequest{
			Mode:   InputRequestBool,
			Prompt: fmt.Sprintf("State is %s old (threshold: %s). Refresh before proceeding? (y/n)", formatDuration(age), formatDuration(g.Threshold)),
		}
	}
	return nil
}

// CheckPlan returns an InputRequest if the plan is stale, or nil if fresh enough.
func (g *StalenessGuard) CheckPlan(plan *TerraformPlan) *InputRequest {
	if plan == nil {
		return &InputRequest{
			Mode:   InputRequestBool,
			Prompt: "No plan cached. Run plan first? (y/n)",
		}
	}
	age := time.Since(plan.LastRefreshed)
	if age > g.Threshold {
		return &InputRequest{
			Mode:   InputRequestBool,
			Prompt: fmt.Sprintf("Plan is %s old (threshold: %s). Re-run plan before proceeding? (y/n)", formatDuration(age), formatDuration(g.Threshold)),
		}
	}
	return nil
}

// IsStale checks if a timestamp is older than the threshold.
func (g *StalenessGuard) IsStale(lastRefreshed time.Time) bool {
	if lastRefreshed.IsZero() {
		return true
	}
	return time.Since(lastRefreshed) > g.Threshold
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
