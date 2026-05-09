package sdk

import "context"

// AIProvider defines the interface for AI-powered features.
// Implementations provide natural language explanations, risk assessment,
// and suggestions powered by an LLM (Claude, etc.).
type AIProvider interface {
	// ExplainChange returns a human-readable explanation of a plan change.
	ExplainChange(ctx context.Context, change PlanChange) (string, error)

	// ExplainPlan summarizes the full plan in natural language.
	ExplainPlan(ctx context.Context, summary *PlanSummary) (string, error)

	// ExplainResource describes what a resource does based on its state.
	ExplainResource(ctx context.Context, resource Resource, detail string) (string, error)

	// SuggestFix takes a terraform error and suggests a resolution.
	SuggestFix(ctx context.Context, err error, tfContext string) (string, error)

	// AssessRisk provides AI-powered risk assessment beyond rule-based classification.
	AssessRisk(ctx context.Context, changes []PlanChange) (string, error)

	// GenerateImport suggests the import ID for a resource type.
	GenerateImport(ctx context.Context, resourceType, address string) (string, error)

	// Stream variants for real-time display:

	// StreamExplainChange streams the explanation token by token.
	StreamExplainChange(ctx context.Context, change PlanChange, onChunk func(string)) error

	// StreamExplainPlan streams the plan summary.
	StreamExplainPlan(ctx context.Context, summary *PlanSummary, onChunk func(string)) error

	// StreamExplainResource streams the resource explanation.
	StreamExplainResource(ctx context.Context, resource Resource, detail string, onChunk func(string)) error

	// StreamAssessRisk streams the risk assessment.
	StreamAssessRisk(ctx context.Context, changes []PlanChange, onChunk func(string)) error
}

// AIStreamChunkMsg is sent to the TUI as tokens arrive from the AI.
type AIStreamChunkMsg struct {
	PluginID string
	Chunk    string
	Done     bool
	Error    error
}

// AIConfig holds configuration for the AI service.
type AIConfig struct {
	Enabled  bool
	Provider string // "anthropic"
	Model    string // "claude-sonnet-4-6-20250514"
	APIKey   string // from env or config
}

// LoadAIConfig extracts AI configuration from the config context.
func LoadAIConfig(cfg *ConfigContext) AIConfig {
	return AIConfig{
		Enabled:  cfg.GetBool("ai.enabled", false),
		Provider: cfg.GetString("ai.provider", "anthropic"),
		Model:    cfg.GetString("ai.model", "claude-sonnet-4-6-20250514"),
		APIKey:   cfg.GetString("ai.api_key", ""),
	}
}
