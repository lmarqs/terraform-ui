package sdk

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
