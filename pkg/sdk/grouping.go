package sdk

import "sort"

// GroupByModule groups plan changes by their module path and returns the groups
// sorted alphabetically. Resources without a module are placed in the "root" group.
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
