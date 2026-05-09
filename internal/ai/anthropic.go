// Package ai provides AI-powered features for terraform-ui using Claude.
// It implements the sdk.AIProvider interface with streaming support for real-time
// token display in the TUI.
//
// Provider detection order:
//  1. Explicit config (ai.provider in tfui.yaml)
//  2. ANTHROPIC_API_KEY env var → direct Anthropic API
//  3. AWS credentials available → Bedrock
//  4. Disabled (no provider found)
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Provider represents the detected AI backend.
type Provider string

const (
	ProviderBedrock   Provider = "bedrock"
	ProviderAnthropic Provider = "anthropic"
	ProviderNone      Provider = "none"
)

// AnthropicProvider implements sdk.AIProvider using Claude.
// Supports both direct Anthropic API and AWS Bedrock.
type AnthropicProvider struct {
	client   *anthropic.Client
	model    string
	provider Provider
}

// DetectProvider determines the best available AI provider from the environment.
// Priority: explicit config > ANTHROPIC_API_KEY > AWS credentials > none.
func DetectProvider(cfg sdk.AIConfig) Provider {
	if cfg.Provider != "" {
		return Provider(cfg.Provider)
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return ProviderAnthropic
	}
	if hasAWSCredentials() {
		return ProviderBedrock
	}
	return ProviderNone
}

// NewAnthropicProvider creates a new AI provider using the best available backend.
// Auto-detects whether to use Bedrock or direct API based on available credentials.
func NewAnthropicProvider(cfg sdk.AIConfig) (*AnthropicProvider, error) {
	provider := DetectProvider(cfg)

	switch provider {
	case ProviderBedrock:
		return newBedrockProvider(cfg)
	case ProviderAnthropic:
		return newDirectProvider(cfg)
	default:
		return nil, fmt.Errorf("no AI credentials found: set ANTHROPIC_API_KEY or configure AWS credentials")
	}
}

func newBedrockProvider(cfg sdk.AIConfig) (*AnthropicProvider, error) {
	opts := []option.RequestOption{
		bedrock.WithLoadDefaultConfig(context.Background()),
	}

	if cfg.Region != "" {
		opts = append(opts, option.WithBaseURL(
			fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", cfg.Region),
		))
	}

	client := anthropic.NewClient(opts...)

	model := cfg.Model
	if model == "" {
		model = "us.anthropic.claude-sonnet-4-6-v1"
	}

	return &AnthropicProvider{
		client:   &client,
		model:    model,
		provider: ProviderBedrock,
	}, nil
}

func newDirectProvider(cfg sdk.AIConfig) (*AnthropicProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6-20250514"
	}

	return &AnthropicProvider{
		client:   &client,
		model:    model,
		provider: ProviderAnthropic,
	}, nil
}

// hasAWSCredentials checks if AWS credentials are available in the environment.
func hasAWSCredentials() bool {
	// Check standard AWS env vars
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		return true
	}
	if os.Getenv("AWS_PROFILE") != "" {
		return true
	}
	if os.Getenv("AWS_ROLE_ARN") != "" {
		return true
	}
	// Check if aws CLI is configured (shared credentials file exists)
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(home + "/.aws/credentials"); err == nil {
			return true
		}
		if _, err := os.Stat(home + "/.aws/config"); err == nil {
			return true
		}
	}
	// Check if running on EC2/ECS with instance role (IMDSv2 quick check)
	if _, err := exec.LookPath("aws"); err == nil {
		return true
	}
	return false
}

// DetectedProvider returns which provider backend is in use.
func (p *AnthropicProvider) DetectedProvider() Provider {
	return p.provider
}

const systemPrompt = `You are a Terraform infrastructure expert embedded in a TUI tool called tfui.
Your role is to explain terraform changes, assess risks, and help users understand their infrastructure.

Guidelines:
- Be concise and actionable. Users are looking at a terminal, not a document.
- Focus on the "why" and "what could go wrong", not just restating the diff.
- When assessing risk, consider: data loss, downtime, cost impact, security implications.
- When explaining resources, describe what they do in the context of the infrastructure.
- Use specific AWS/GCP/Azure terminology when relevant.
- Never suggest running commands outside of tfui — the user is inside the TUI.`

// ExplainChange returns a human-readable explanation of a plan change.
func (p *AnthropicProvider) ExplainChange(ctx context.Context, change sdk.PlanChange) (string, error) {
	prompt := formatChangePrompt(change)
	return p.complete(ctx, prompt)
}

// ExplainPlan summarizes the full plan in natural language.
func (p *AnthropicProvider) ExplainPlan(ctx context.Context, summary *sdk.PlanSummary) (string, error) {
	prompt := formatPlanPrompt(summary)
	return p.complete(ctx, prompt)
}

// ExplainResource describes what a resource does based on its state.
func (p *AnthropicProvider) ExplainResource(ctx context.Context, resource sdk.Resource, detail string) (string, error) {
	prompt := fmt.Sprintf(`Explain this terraform resource in 2-3 sentences. What does it do? What depends on it?

Resource: %s (type: %s, provider: %s)

State detail:
%s`, resource.Address, resource.Type, resource.ProviderName, detail)
	return p.complete(ctx, prompt)
}

// SuggestFix takes a terraform error and suggests a resolution.
func (p *AnthropicProvider) SuggestFix(ctx context.Context, err error, tfContext string) (string, error) {
	prompt := fmt.Sprintf(`A terraform operation failed with this error:

%s

Context: %s

Suggest how to fix this in 2-3 bullet points. Be specific.`, err.Error(), tfContext)
	return p.complete(ctx, prompt)
}

// AssessRisk provides AI-powered risk assessment beyond rule-based classification.
func (p *AnthropicProvider) AssessRisk(ctx context.Context, changes []sdk.PlanChange) (string, error) {
	prompt := formatRiskPrompt(changes)
	return p.complete(ctx, prompt)
}

// GenerateImport suggests the import ID for a resource type.
func (p *AnthropicProvider) GenerateImport(ctx context.Context, resourceType, address string) (string, error) {
	prompt := fmt.Sprintf(`What is the correct import ID format for terraform resource type %q?

Resource address: %s

Respond with ONLY the import ID format/example (e.g., "arn:aws:s3:::bucket-name" for aws_s3_bucket).
If you can infer the likely ID from the address, provide it. Otherwise show the format with placeholders.`, resourceType, address)
	return p.complete(ctx, prompt)
}

// StreamExplainChange streams the explanation token by token.
func (p *AnthropicProvider) StreamExplainChange(ctx context.Context, change sdk.PlanChange, onChunk func(string)) error {
	prompt := formatChangePrompt(change)
	return p.stream(ctx, prompt, onChunk)
}

// StreamExplainPlan streams the plan summary.
func (p *AnthropicProvider) StreamExplainPlan(ctx context.Context, summary *sdk.PlanSummary, onChunk func(string)) error {
	prompt := formatPlanPrompt(summary)
	return p.stream(ctx, prompt, onChunk)
}

// StreamExplainResource streams the resource explanation.
func (p *AnthropicProvider) StreamExplainResource(ctx context.Context, resource sdk.Resource, detail string, onChunk func(string)) error {
	prompt := fmt.Sprintf(`Explain this terraform resource in 2-3 sentences. What does it do?

Resource: %s (type: %s, provider: %s)

State detail:
%s`, resource.Address, resource.Type, resource.ProviderName, detail)
	return p.stream(ctx, prompt, onChunk)
}

// StreamAssessRisk streams the risk assessment.
func (p *AnthropicProvider) StreamAssessRisk(ctx context.Context, changes []sdk.PlanChange, onChunk func(string)) error {
	prompt := formatRiskPrompt(changes)
	return p.stream(ctx, prompt, onChunk)
}

func (p *AnthropicProvider) complete(ctx context.Context, userPrompt string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API error: %w", err)
	}

	var result strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}
	return result.String(), nil
}

func (p *AnthropicProvider) stream(ctx context.Context, userPrompt string, onChunk func(string)) error {
	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})

	for stream.Next() {
		event := stream.Current()
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			onChunk(event.Delta.Text)
		}
	}

	return stream.Err()
}

func formatChangePrompt(change sdk.PlanChange) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Explain this terraform plan change in 2-3 sentences:\n\n"))
	b.WriteString(fmt.Sprintf("Action: %s\n", change.Action))
	b.WriteString(fmt.Sprintf("Resource: %s (type: %s)\n", change.Resource.Address, change.Resource.Type))
	if change.Resource.Module != "" {
		b.WriteString(fmt.Sprintf("Module: %s\n", change.Resource.Module))
	}
	b.WriteString(fmt.Sprintf("Risk: %s\n", change.Risk.String()))
	if change.IsPhantom {
		b.WriteString("This is a phantom/cosmetic change.\n")
	}
	if len(change.AttributeDiffs) > 0 {
		b.WriteString("\nAttribute changes:\n")
		for _, diff := range change.AttributeDiffs {
			if diff.Sensitive {
				b.WriteString(fmt.Sprintf("  %s: (sensitive)\n", diff.Key))
			} else {
				b.WriteString(fmt.Sprintf("  %s: %s → %s\n", diff.Key, diff.OldValue, diff.NewValue))
			}
		}
	}
	return b.String()
}

func formatPlanPrompt(summary *sdk.PlanSummary) string {
	var b strings.Builder
	b.WriteString("Summarize this terraform plan in 3-5 sentences. Focus on what's happening and potential impact:\n\n")
	b.WriteString(fmt.Sprintf("Summary: %d to create, %d to update, %d to delete, %d to replace\n\n",
		summary.ToCreate, summary.ToUpdate, summary.ToDelete, summary.ToReplace))
	b.WriteString("Changes:\n")
	for _, c := range summary.Changes {
		b.WriteString(fmt.Sprintf("  %s %s [risk: %s]", c.Action, c.Resource.Address, c.Risk.String()))
		if c.IsPhantom {
			b.WriteString(" (phantom)")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func formatRiskPrompt(changes []sdk.PlanChange) string {
	var b strings.Builder
	b.WriteString("Assess the risk of this terraform plan. Consider: data loss, downtime, cost, security.\n")
	b.WriteString("Respond with: overall risk level, top concerns (2-3 bullets), and any recommendations.\n\n")
	b.WriteString("Changes:\n")
	for _, c := range changes {
		b.WriteString(fmt.Sprintf("  %s %s (type: %s, risk: %s)\n", c.Action, c.Resource.Address, c.Resource.Type, c.Risk.String()))
		if len(c.AttributeDiffs) > 0 {
			diffs := c.AttributeDiffs
			if len(diffs) > 5 {
				diffs = diffs[:5]
			}
			for _, d := range diffs {
				if d.Sensitive {
					b.WriteString(fmt.Sprintf("    %s: (sensitive)\n", d.Key))
				} else {
					old := truncate(d.OldValue, 50)
					new := truncate(d.NewValue, 50)
					b.WriteString(fmt.Sprintf("    %s: %s → %s\n", d.Key, old, new))
				}
			}
		}
	}

	changeJSON, _ := json.Marshal(struct {
		Total   int
		Creates int
		Updates int
		Deletes int
	}{len(changes), countAction(changes, sdk.ActionCreate), countAction(changes, sdk.ActionUpdate), countAction(changes, sdk.ActionDelete)})
	b.WriteString(fmt.Sprintf("\nAggregate: %s\n", string(changeJSON)))

	return b.String()
}

func countAction(changes []sdk.PlanChange, action sdk.Action) int {
	count := 0
	for _, c := range changes {
		if c.Action == action {
			count++
		}
	}
	return count
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
