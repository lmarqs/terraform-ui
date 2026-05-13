package sdk

// BuildPlanOptions constructs PlanOptions from the current session state + explicit targets.
// This ensures plan always uses the latest resolved config (var-files, vars, extra-args).
func BuildPlanOptions(session *Session, targets []string) PlanOptions {
	opts := PlanOptions{Targets: targets}
	if session == nil {
		return opts
	}
	if vf, ok := GetTyped[[]string](session, SessionKeyVarFiles); ok {
		opts.VarFiles = vf
	}
	if v, ok := GetTyped[map[string]string](session, SessionKeyVars); ok {
		opts.Vars = v
	}
	if ea, ok := GetTyped[[]string](session, SessionKeyExtraArgs); ok {
		opts.ExtraArgs = ea
	}
	return opts
}

// BuildApplyOptions constructs ApplyOptions from the current session state + explicit targets.
func BuildApplyOptions(session *Session, targets []string) ApplyOptions {
	opts := ApplyOptions{Targets: targets}
	if session == nil {
		return opts
	}
	if vf, ok := GetTyped[[]string](session, SessionKeyVarFiles); ok {
		opts.VarFiles = vf
	}
	if v, ok := GetTyped[map[string]string](session, SessionKeyVars); ok {
		opts.Vars = v
	}
	if ea, ok := GetTyped[[]string](session, SessionKeyExtraArgs); ok {
		opts.ExtraArgs = ea
	}
	return opts
}
