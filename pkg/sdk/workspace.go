package sdk

// Workspace is a terraform workspace name within a chdir.
type Workspace string

const WorkspaceDefault Workspace = "default"

func NewWorkspace(name string) Workspace { return Workspace(name) }
func (w Workspace) String() string       { return string(w) }
func (w Workspace) IsDefault() bool      { return w == WorkspaceDefault }
func (w Workspace) IsZero() bool         { return w == "" }
