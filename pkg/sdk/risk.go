package sdk

// RiskLevel classifies the risk severity of a planned infrastructure change,
// ranging from RiskNone (no risk) to RiskCritical (potentially destructive).
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

// String returns the lowercase string representation of the risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "none"
	}
}

// OverallRisk returns the highest risk level found across all changes in the slice.
func OverallRisk(changes []PlanChange) RiskLevel {
	max := RiskNone
	for i := range changes {
		if changes[i].Risk > max {
			max = changes[i].Risk
		}
	}
	return max
}
