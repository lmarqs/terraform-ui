package terraform

import (
	"sort"
	"strings"
)

// ModuleGroup represents a group of changes belonging to the same module.
type ModuleGroup struct {
	Module  string
	Summary ActionSummary
	Changes []PlanChange
}

// ActionSummary holds counts of changes by action type.
type ActionSummary struct {
	Add     int
	Change  int
	Destroy int
	Replace int
}

// GroupByModule groups plan changes by their module path.
// Groups are sorted alphabetically by module path.
func GroupByModule(changes []PlanChange) []ModuleGroup {
	groups := make(map[string]*ModuleGroup)

	for i := range changes {
		mod := changes[i].Resource.Module
		if mod == "" {
			mod = "root"
		}

		g, exists := groups[mod]
		if !exists {
			g = &ModuleGroup{
				Module:  mod,
				Changes: make([]PlanChange, 0),
			}
			groups[mod] = g
		}
		g.Changes = append(g.Changes, changes[i])

		switch changes[i].Action {
		case ActionCreate:
			g.Summary.Add++
		case ActionUpdate:
			g.Summary.Change++
		case ActionDelete:
			g.Summary.Destroy++
		case ActionDeleteThenCreate, ActionCreateThenDelete:
			g.Summary.Replace++
		}
	}

	result := make([]ModuleGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Module < result[j].Module
	})

	return result
}

// ExtractModule extracts the module path from a resource address.
// "module.vpc.aws_subnet.main" -> "module.vpc"
// "module.vpc.module.subnets.aws_subnet.a" -> "module.vpc.module.subnets"
// "aws_instance.web" -> ""
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
