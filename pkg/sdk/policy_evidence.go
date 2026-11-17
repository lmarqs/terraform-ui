package sdk

// PolicyEvidence retains resource-local policy inputs independently of display diffs.
// ErrorCode invalidates the entire evidence set, without preventing Plan review.
type PolicyEvidence struct {
	PlanID    string                   `json:"plan_id"`
	Resources []PolicyResourceEvidence `json:"resources"`
	ErrorCode ErrorCode                `json:"error_code,omitempty"`
}

// PolicyResourceEvidence distinguishes eligible inputs from explicit exclusions.
type PolicyResourceEvidence struct {
	Address   string       `json:"address"`
	Deposed   string       `json:"deposed,omitempty"`
	Actions   []string     `json:"actions"`
	Exclusion Exclusion    `json:"exclusion,omitempty"`
	Input     *PolicyInput `json:"input,omitempty"`
}

// PolicyInput is the independent tfui.risk.input/v1 envelope.
type PolicyInput struct {
	SchemaVersion string         `json:"schema_version"`
	Resource      PolicyResource `json:"resource"`
	Change        PolicyChange   `json:"change"`
}

// PolicyResource contains only the resource identity exposed to policy authors.
type PolicyResource struct {
	Address         string      `json:"address"`
	PreviousAddress *string     `json:"previous_address"`
	Deposed         *string     `json:"deposed"`
	Mode            string      `json:"mode"`
	Type            string      `json:"type"`
	Name            string      `json:"name"`
	ProviderName    string      `json:"provider_name"`
	Index           PolicyValue `json:"index"`
}

// PolicyChange preserves action order and tagged before/after evidence.
type PolicyChange struct {
	Actions      []string      `json:"actions"`
	Importing    *PolicyImport `json:"importing"`
	ReplacePaths [][]any       `json:"replace_paths"`
	Before       PolicyValue   `json:"before"`
	After        PolicyValue   `json:"after"`
}

// PolicyImport retains uncertainty without exposing an unmasked import identifier.
type PolicyImport struct {
	ID      PolicyValue `json:"id"`
	Unknown bool        `json:"unknown"`
}

// PolicyValue preserves absence, null, uncertainty and sensitivity as distinct kinds.
// Container pointers retain empty entries/items in JSON; opaque nodes have no children.
type PolicyValue struct {
	Kind    PolicyValueKind         `json:"kind"`
	Value   any                     `json:"value,omitempty"`
	Entries *map[string]PolicyValue `json:"entries,omitempty"`
	Items   *[]PolicyValue          `json:"items,omitempty"`
}
