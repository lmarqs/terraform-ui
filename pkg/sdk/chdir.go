package sdk

// Chdir is the relative member path within a project (e.g., "modules/vpc").
// This is the user-facing concept: which member directory is selected.
type Chdir string

func (c Chdir) String() string { return string(c) }
func (c Chdir) IsZero() bool   { return c == "" }
