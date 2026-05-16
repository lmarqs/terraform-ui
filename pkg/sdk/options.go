package sdk

// ResolvedOptions holds CLI/config values resolved from tfui.hcl.
// Shared across plugins via Context pointer — re-resolved on workspace/chdir change.
type ResolvedOptions struct {
	VarFiles  []string
	Vars      map[string]string
	ExtraArgs []string
}

// BuildPlanOptions constructs PlanOptions from resolved options + explicit targets.
func BuildPlanOptions(resolved *ResolvedOptions, targets []string) PlanOptions {
	opts := PlanOptions{Targets: targets}
	if resolved == nil {
		return opts
	}
	opts.VarFiles = resolved.VarFiles
	opts.Vars = resolved.Vars
	opts.ExtraArgs = resolved.ExtraArgs
	return opts
}

// BuildApplyOptions constructs ApplyOptions from resolved options + explicit targets.
func BuildApplyOptions(resolved *ResolvedOptions, targets []string) ApplyOptions {
	opts := ApplyOptions{Targets: targets}
	if resolved == nil {
		return opts
	}
	opts.VarFiles = resolved.VarFiles
	opts.Vars = resolved.Vars
	opts.ExtraArgs = resolved.ExtraArgs
	return opts
}
