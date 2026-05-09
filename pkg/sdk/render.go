package sdk

// ActionSymbol returns a styled symbol representing the terraform action.
func ActionSymbol(action Action) string {
	switch action {
	case ActionCreate:
		return StyleCreate.Render("+")
	case ActionUpdate:
		return StyleUpdate.Render("~")
	case ActionDelete:
		return StyleDelete.Render("-")
	case ActionDeleteThenCreate, ActionCreateThenDelete:
		return StyleReplace.Render("-/+")
	case ActionRead:
		return StyleFaint.Render("<=")
	default:
		return " "
	}
}

// RiskBadge returns a styled badge representing the risk level.
func RiskBadge(risk RiskLevel) string {
	switch risk {
	case RiskLow:
		return StyleRiskLow.Render("[low]")
	case RiskMedium:
		return StyleRiskMedium.Render("[medium]")
	case RiskHigh:
		return StyleRiskHigh.Render("[HIGH]")
	case RiskCritical:
		return StyleRiskCritical.Render("[CRITICAL]")
	default:
		return ""
	}
}

// Truncate shortens a string to maxLen characters, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen < 10 {
		maxLen = 10
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// ScrollWindow calculates the visible range for a scrollable list.
// Returns the start and end indices for the visible portion.
func ScrollWindow(selected, total, availableHeight, minVisible int) (start, end int) {
	maxVisible := availableHeight
	if maxVisible < minVisible {
		maxVisible = minVisible
	}
	if selected >= maxVisible {
		start = selected - maxVisible + 1
	}
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
