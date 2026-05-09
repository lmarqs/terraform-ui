package sdk

// PluginAction represents a named operation exposed by a plugin.
// Actions are callable from CLI (tfui state mv) and REPL (:state mv).
type PluginAction struct {
	Name        string
	Description string
	Args        []ArgDef
	Flags       []FlagDef
	Run         func(ctx *AppContext, args ActionArgs) error
}

// ArgDef defines a positional argument for an action.
type ArgDef struct {
	Name     string
	Required bool
	Desc     string
}

// FlagDef defines a flag parameter for an action.
type FlagDef struct {
	Name     string
	Short    string
	Default  string
	Required bool
	Desc     string
}

// ActionArgs holds parsed arguments for an action invocation.
type ActionArgs struct {
	Positional []string
	Flags      map[string]string
}

// GetFlag returns the value of a flag, or its default if not set.
func (a ActionArgs) GetFlag(name, defaultValue string) string {
	if v, ok := a.Flags[name]; ok {
		return v
	}
	return defaultValue
}

// GetArg returns the positional argument at index, or empty string if out of bounds.
func (a ActionArgs) GetArg(index int) string {
	if index < len(a.Positional) {
		return a.Positional[index]
	}
	return ""
}

// HasFlag reports whether a flag was provided.
func (a ActionArgs) HasFlag(name string) bool {
	_, ok := a.Flags[name]
	return ok
}
