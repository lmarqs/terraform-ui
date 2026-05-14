package ai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockMessages struct {
	newResult  *anthropic.Message
	newErr     error
	streamData []anthropic.MessageStreamEventUnion
	streamErr  error
	lastParams anthropic.MessageNewParams
}

func (m *mockMessages) New(_ context.Context, params anthropic.MessageNewParams, _ ...option.RequestOption) (*anthropic.Message, error) {
	m.lastParams = params
	return m.newResult, m.newErr
}

func (m *mockMessages) NewStreaming(_ context.Context, params anthropic.MessageNewParams, _ ...option.RequestOption) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	m.lastParams = params
	return newMockStream(m.streamData, m.streamErr)
}

type mockDecoder struct {
	events []ssestream.Event
	idx    int
	err    error
}

func (d *mockDecoder) Event() ssestream.Event {
	if d.idx > 0 && d.idx <= len(d.events) {
		return d.events[d.idx-1]
	}
	return ssestream.Event{}
}

func (d *mockDecoder) Next() bool {
	if d.idx < len(d.events) {
		d.idx++
		return true
	}
	return false
}

func (d *mockDecoder) Close() error { return nil }
func (d *mockDecoder) Err() error   { return d.err }

func newMockStream(events []anthropic.MessageStreamEventUnion, streamErr error) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	var sseEvents []ssestream.Event
	for _, e := range events {
		data, _ := json.Marshal(e)
		sseEvents = append(sseEvents, ssestream.Event{
			Type: e.Type,
			Data: data,
		})
	}
	decoder := &mockDecoder{events: sseEvents, err: streamErr}
	return ssestream.NewStream[anthropic.MessageStreamEventUnion](decoder, nil)
}

func newTestProvider(m *mockMessages) *AnthropicProvider {
	return &AnthropicProvider{
		messages: m,
		model:    "test-model",
		provider: ProviderAnthropic,
	}
}

func textMessage(text string) *anthropic.Message {
	return &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: "text", Text: text},
		},
	}
}

// --- Pure helper tests ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"long string truncated", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
		{"max zero", "hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
			}
		})
	}
}

func TestCountAction(t *testing.T) {
	changes := []sdk.PlanChange{
		{Action: sdk.ActionCreate},
		{Action: sdk.ActionCreate},
		{Action: sdk.ActionUpdate},
		{Action: sdk.ActionDelete},
		{Action: sdk.ActionCreate},
	}

	tests := []struct {
		name     string
		action   sdk.Action
		expected int
	}{
		{"counts creates", sdk.ActionCreate, 3},
		{"counts updates", sdk.ActionUpdate, 1},
		{"counts deletes", sdk.ActionDelete, 1},
		{"counts missing action as zero", sdk.ActionNoOp, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countAction(changes, tt.action)
			if got != tt.expected {
				t.Errorf("countAction(changes, %q) = %d, want %d", tt.action, got, tt.expected)
			}
		})
	}
}

func TestCountAction_EmptySlice(t *testing.T) {
	got := countAction(nil, sdk.ActionCreate)
	if got != 0 {
		t.Errorf("countAction(nil, create) = %d, want 0", got)
	}
}

func TestFormatChangePrompt(t *testing.T) {
	tests := []struct {
		name     string
		change   sdk.PlanChange
		contains []string
	}{
		{
			"basic change",
			sdk.PlanChange{
				Action:   sdk.ActionCreate,
				Resource: sdk.Resource{Address: "aws_s3_bucket.main", Type: "aws_s3_bucket"},
				Risk:     sdk.RiskLow,
			},
			[]string{"create", "aws_s3_bucket.main", "aws_s3_bucket", "low"},
		},
		{
			"with module",
			sdk.PlanChange{
				Action:   sdk.ActionUpdate,
				Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a", Type: "aws_subnet", Module: "module.vpc"},
				Risk:     sdk.RiskMedium,
			},
			[]string{"Module: module.vpc"},
		},
		{
			"phantom change",
			sdk.PlanChange{
				Action:    sdk.ActionUpdate,
				Resource:  sdk.Resource{Address: "aws_iam_role.main", Type: "aws_iam_role"},
				Risk:      sdk.RiskNone,
				IsPhantom: true,
			},
			[]string{"phantom", "cosmetic"},
		},
		{
			"with attribute diffs",
			sdk.PlanChange{
				Action:   sdk.ActionUpdate,
				Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance"},
				Risk:     sdk.RiskHigh,
				AttributeDiffs: []sdk.AttributeDiff{
					{Key: "instance_type", OldValue: "t3.micro", NewValue: "t3.large"},
				},
			},
			[]string{"instance_type", "t3.micro", "t3.large"},
		},
		{
			"sensitive attribute",
			sdk.PlanChange{
				Action:   sdk.ActionUpdate,
				Resource: sdk.Resource{Address: "aws_db_instance.main", Type: "aws_db_instance"},
				Risk:     sdk.RiskHigh,
				AttributeDiffs: []sdk.AttributeDiff{
					{Key: "password", Sensitive: true},
				},
			},
			[]string{"password", "(sensitive)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatChangePrompt(tt.change)
			for _, substr := range tt.contains {
				if !strings.Contains(got, substr) {
					t.Errorf("formatChangePrompt() missing %q in:\n%s", substr, got)
				}
			}
		})
	}
}

func TestFormatPlanPrompt(t *testing.T) {
	summary := &sdk.PlanSummary{
		ToCreate:  2,
		ToUpdate:  1,
		ToDelete:  0,
		ToReplace: 1,
		Changes: []sdk.PlanChange{
			{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "aws_s3_bucket.a"}, Risk: sdk.RiskLow},
			{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "aws_s3_bucket.b"}, Risk: sdk.RiskLow},
			{Action: sdk.ActionUpdate, Resource: sdk.Resource{Address: "aws_iam_role.x"}, Risk: sdk.RiskMedium},
			{Action: sdk.ActionDeleteThenCreate, Resource: sdk.Resource{Address: "aws_instance.w"}, Risk: sdk.RiskHigh, IsPhantom: true},
		},
	}

	got := formatPlanPrompt(summary)

	expected := []string{
		"2 to create",
		"1 to update",
		"0 to delete",
		"1 to replace",
		"aws_s3_bucket.a",
		"aws_s3_bucket.b",
		"aws_iam_role.x",
		"aws_instance.w",
		"(phantom)",
	}

	for _, substr := range expected {
		if !strings.Contains(got, substr) {
			t.Errorf("formatPlanPrompt() missing %q in:\n%s", substr, got)
		}
	}
}

func TestFormatPlanPrompt_Empty(t *testing.T) {
	summary := &sdk.PlanSummary{}
	got := formatPlanPrompt(summary)
	if !strings.Contains(got, "0 to create") {
		t.Errorf("formatPlanPrompt(empty) missing '0 to create' in:\n%s", got)
	}
}

func TestFormatRiskPrompt(t *testing.T) {
	changes := []sdk.PlanChange{
		{
			Action:   sdk.ActionDelete,
			Resource: sdk.Resource{Address: "aws_db_instance.prod", Type: "aws_db_instance"},
			Risk:     sdk.RiskCritical,
			AttributeDiffs: []sdk.AttributeDiff{
				{Key: "engine", OldValue: "postgres", NewValue: ""},
				{Key: "password", Sensitive: true},
			},
		},
		{
			Action:   sdk.ActionUpdate,
			Resource: sdk.Resource{Address: "aws_security_group.web", Type: "aws_security_group"},
			Risk:     sdk.RiskHigh,
		},
	}

	got := formatRiskPrompt(changes)

	expected := []string{
		"aws_db_instance.prod",
		"aws_security_group.web",
		"delete",
		"update",
		"critical",
		"high",
		"engine",
		"postgres",
		"(sensitive)",
		"Aggregate",
	}

	for _, substr := range expected {
		if !strings.Contains(got, substr) {
			t.Errorf("formatRiskPrompt() missing %q in:\n%s", substr, got)
		}
	}
}

func TestFormatRiskPrompt_TruncatesLongValues(t *testing.T) {
	longValue := strings.Repeat("x", 100)
	changes := []sdk.PlanChange{
		{
			Action:   sdk.ActionUpdate,
			Resource: sdk.Resource{Address: "res.a", Type: "t"},
			Risk:     sdk.RiskLow,
			AttributeDiffs: []sdk.AttributeDiff{
				{Key: "data", OldValue: longValue, NewValue: "short"},
			},
		},
	}

	got := formatRiskPrompt(changes)
	if strings.Contains(got, longValue) {
		t.Error("formatRiskPrompt should truncate long values")
	}
	if !strings.Contains(got, "...") {
		t.Error("formatRiskPrompt should include ellipsis for truncated values")
	}
}

func TestFormatRiskPrompt_LimitsAttributeDiffs(t *testing.T) {
	diffs := make([]sdk.AttributeDiff, 10)
	for i := range diffs {
		diffs[i] = sdk.AttributeDiff{Key: "attr_" + string(rune('a'+i)), OldValue: "old", NewValue: "new"}
	}

	changes := []sdk.PlanChange{
		{
			Action:         sdk.ActionUpdate,
			Resource:       sdk.Resource{Address: "res.a", Type: "t"},
			Risk:           sdk.RiskLow,
			AttributeDiffs: diffs,
		},
	}

	got := formatRiskPrompt(changes)
	if strings.Contains(got, "attr_f") {
		t.Error("formatRiskPrompt should limit to 5 attribute diffs")
	}
	if !strings.Contains(got, "attr_e") {
		t.Error("formatRiskPrompt should include up to 5th attribute diff")
	}
}

func TestFormatRiskPrompt_AggregateJSON(t *testing.T) {
	changes := []sdk.PlanChange{
		{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "a", Type: "t"}, Risk: sdk.RiskLow},
		{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "b", Type: "t"}, Risk: sdk.RiskLow},
		{Action: sdk.ActionDelete, Resource: sdk.Resource{Address: "c", Type: "t"}, Risk: sdk.RiskHigh},
	}

	got := formatRiskPrompt(changes)

	var aggregate struct {
		Total   int
		Creates int
		Updates int
		Deletes int
	}

	idx := strings.Index(got, "Aggregate: ")
	if idx == -1 {
		t.Fatal("missing Aggregate line")
	}
	jsonStr := got[idx+len("Aggregate: "):]
	jsonStr = strings.TrimRight(jsonStr, "\n")
	if err := json.Unmarshal([]byte(jsonStr), &aggregate); err != nil {
		t.Fatalf("failed to parse aggregate JSON: %v, raw: %q", err, jsonStr)
	}
	if aggregate.Total != 3 {
		t.Errorf("Total = %d, want 3", aggregate.Total)
	}
	if aggregate.Creates != 2 {
		t.Errorf("Creates = %d, want 2", aggregate.Creates)
	}
	if aggregate.Deletes != 1 {
		t.Errorf("Deletes = %d, want 1", aggregate.Deletes)
	}
	if aggregate.Updates != 0 {
		t.Errorf("Updates = %d, want 0", aggregate.Updates)
	}
}

// --- DetectProvider tests ---

func TestDetectProvider_WhenExplicitConfig(t *testing.T) {
	got := DetectProvider(sdk.AIConfig{Provider: "bedrock"})
	if got != ProviderBedrock {
		t.Errorf("DetectProvider(explicit bedrock) = %q, want %q", got, ProviderBedrock)
	}
}

func TestDetectProvider_WhenAnthropicAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")

	got := DetectProvider(sdk.AIConfig{})
	if got != ProviderAnthropic {
		t.Errorf("DetectProvider(ANTHROPIC_API_KEY set) = %q, want %q", got, ProviderAnthropic)
	}
}

func TestDetectProvider_WhenAWSCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA123")

	got := DetectProvider(sdk.AIConfig{})
	if got != ProviderBedrock {
		t.Errorf("DetectProvider(AWS_ACCESS_KEY_ID set) = %q, want %q", got, ProviderBedrock)
	}
}

func TestDetectProvider_WhenNoCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")

	got := DetectProvider(sdk.AIConfig{})
	if got != ProviderNone {
		t.Errorf("DetectProvider(no creds) = %q, want %q", got, ProviderNone)
	}
}

func TestDetectProvider_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")

	got := DetectProvider(sdk.AIConfig{Provider: "bedrock"})
	if got != ProviderBedrock {
		t.Errorf("explicit provider should override env, got %q", got)
	}
}

// --- hasAWSCredentials tests ---

func TestHasAWSCredentials_WhenAccessKeyID(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA123")
	if !hasAWSCredentials() {
		t.Error("expected true when AWS_ACCESS_KEY_ID is set")
	}
}

func TestHasAWSCredentials_WhenProfile(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "prod")
	if !hasAWSCredentials() {
		t.Error("expected true when AWS_PROFILE is set")
	}
}

func TestHasAWSCredentials_WhenRoleARN(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123:role/test")
	if !hasAWSCredentials() {
		t.Error("expected true when AWS_ROLE_ARN is set")
	}
}

func TestHasAWSCredentials_WhenCredentialsFile(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create .aws/credentials file
	awsDir := tmpHome + "/.aws"
	_ = makeDir(t, awsDir)
	_ = writeFile(t, awsDir+"/credentials", "")

	if !hasAWSCredentials() {
		t.Error("expected true when ~/.aws/credentials exists")
	}
}

func TestHasAWSCredentials_WhenConfigFile(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	awsDir := tmpHome + "/.aws"
	_ = makeDir(t, awsDir)
	_ = writeFile(t, awsDir+"/config", "")

	if !hasAWSCredentials() {
		t.Error("expected true when ~/.aws/config exists")
	}
}

func TestHasAWSCredentials_WhenNone(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")

	if hasAWSCredentials() {
		t.Error("expected false when no AWS credentials available")
	}
}

func TestHasAWSCredentials_WhenAWSCLIInPath(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	binDir := tmpHome + "/bin"
	_ = makeDir(t, binDir)
	_ = writeFile(t, binDir+"/aws", "#!/bin/sh\n")
	_ = os.Chmod(binDir+"/aws", 0o755)
	t.Setenv("PATH", binDir)

	if !hasAWSCredentials() {
		t.Error("expected true when aws CLI is in PATH")
	}
}

// --- NewAnthropicProvider tests ---

func TestNewAnthropicProvider_WhenNoCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ROLE_ARN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")

	_, err := NewAnthropicProvider(sdk.AIConfig{})
	if err == nil {
		t.Fatal("expected error when no credentials")
	}
	if !strings.Contains(err.Error(), "no AI credentials found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAnthropicProvider_WhenDirectProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")

	p, err := NewAnthropicProvider(sdk.AIConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.provider != ProviderAnthropic {
		t.Errorf("provider = %q, want %q", p.provider, ProviderAnthropic)
	}
	if p.model != "claude-sonnet-4-6-20250514" {
		t.Errorf("model = %q, want default direct model", p.model)
	}
}

func TestNewAnthropicProvider_WhenDirectWithCustomModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")

	p, err := NewAnthropicProvider(sdk.AIConfig{Model: "claude-3-haiku-20240307"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.model != "claude-3-haiku-20240307" {
		t.Errorf("model = %q, want custom model", p.model)
	}
}

func TestNewAnthropicProvider_WhenBedrockProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	p, err := NewAnthropicProvider(sdk.AIConfig{Provider: "bedrock"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.provider != ProviderBedrock {
		t.Errorf("provider = %q, want %q", p.provider, ProviderBedrock)
	}
	if p.model != "us.anthropic.claude-sonnet-4-6-v1" {
		t.Errorf("model = %q, want default bedrock model", p.model)
	}
}

func TestNewAnthropicProvider_WhenBedrockWithCustomModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	p, err := NewAnthropicProvider(sdk.AIConfig{Provider: "bedrock", Model: "custom-model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.model != "custom-model" {
		t.Errorf("model = %q, want custom-model", p.model)
	}
}

func TestNewAnthropicProvider_WhenBedrockWithRegion(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	p, err := NewAnthropicProvider(sdk.AIConfig{Provider: "bedrock", Region: "eu-west-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.provider != ProviderBedrock {
		t.Errorf("provider = %q, want bedrock", p.provider)
	}
}

func TestNewDirectProvider_WhenNoAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := newDirectProvider(sdk.AIConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY not set") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- DetectedProvider tests ---

func TestDetectedProvider(t *testing.T) {
	p := &AnthropicProvider{provider: ProviderBedrock}
	if p.DetectedProvider() != ProviderBedrock {
		t.Errorf("DetectedProvider() = %q, want %q", p.DetectedProvider(), ProviderBedrock)
	}
}

// --- Complete method tests (via public API methods) ---

func TestExplainChange_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("This change creates a new S3 bucket."),
	}
	p := newTestProvider(m)

	result, err := p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "aws_s3_bucket.main", Type: "aws_s3_bucket"},
		Risk:     sdk.RiskLow,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "This change creates a new S3 bucket." {
		t.Errorf("result = %q, want expected text", result)
	}
}

func TestExplainChange_WhenAPIError(t *testing.T) {
	m := &mockMessages{
		newErr: errors.New("rate limited"),
	}
	p := newTestProvider(m)

	_, err := p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "anthropic API error") {
		t.Errorf("error should wrap as anthropic API error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error should contain original message, got: %v", err)
	}
}

func TestExplainChange_WhenMultipleTextBlocks(t *testing.T) {
	m := &mockMessages{
		newResult: &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: "First part. "},
				{Type: "text", Text: "Second part."},
			},
		},
	}
	p := newTestProvider(m)

	result, err := p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "First part. Second part." {
		t.Errorf("result = %q, want concatenated text", result)
	}
}

func TestExplainChange_WhenNonTextBlocks(t *testing.T) {
	m := &mockMessages{
		newResult: &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "thinking", Text: "should be skipped"},
				{Type: "text", Text: "visible"},
			},
		},
	}
	p := newTestProvider(m)

	result, err := p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "visible" {
		t.Errorf("result = %q, want only text blocks", result)
	}
}

func TestExplainPlan_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("Plan creates 2 resources."),
	}
	p := newTestProvider(m)

	result, err := p.ExplainPlan(context.Background(), &sdk.PlanSummary{
		ToCreate: 2,
		Changes: []sdk.PlanChange{
			{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "a"}, Risk: sdk.RiskLow},
			{Action: sdk.ActionCreate, Resource: sdk.Resource{Address: "b"}, Risk: sdk.RiskLow},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Plan creates 2 resources." {
		t.Errorf("result = %q", result)
	}
}

func TestExplainPlan_WhenAPIError(t *testing.T) {
	m := &mockMessages{newErr: errors.New("timeout")}
	p := newTestProvider(m)

	_, err := p.ExplainPlan(context.Background(), &sdk.PlanSummary{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExplainResource_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("This is an S3 bucket for static assets."),
	}
	p := newTestProvider(m)

	result, err := p.ExplainResource(context.Background(), sdk.Resource{
		Address:      "aws_s3_bucket.assets",
		Type:         "aws_s3_bucket",
		ProviderName: "aws",
	}, `{"bucket": "my-assets"}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "This is an S3 bucket for static assets." {
		t.Errorf("result = %q", result)
	}
}

func TestExplainResource_WhenAPIError(t *testing.T) {
	m := &mockMessages{newErr: errors.New("server error")}
	p := newTestProvider(m)

	_, err := p.ExplainResource(context.Background(), sdk.Resource{
		Address: "res.a", Type: "t", ProviderName: "p",
	}, "{}")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSuggestFix_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("Try refreshing state first."),
	}
	p := newTestProvider(m)

	result, err := p.SuggestFix(context.Background(), errors.New("resource not found"), "during apply")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Try refreshing state first." {
		t.Errorf("result = %q", result)
	}
}

func TestSuggestFix_WhenAPIError(t *testing.T) {
	m := &mockMessages{newErr: errors.New("api down")}
	p := newTestProvider(m)

	_, err := p.SuggestFix(context.Background(), errors.New("some err"), "ctx")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssessRisk_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("High risk: database deletion."),
	}
	p := newTestProvider(m)

	result, err := p.AssessRisk(context.Background(), []sdk.PlanChange{
		{Action: sdk.ActionDelete, Resource: sdk.Resource{Address: "aws_db.prod", Type: "aws_db_instance"}, Risk: sdk.RiskCritical},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "High risk: database deletion." {
		t.Errorf("result = %q", result)
	}
}

func TestAssessRisk_WhenAPIError(t *testing.T) {
	m := &mockMessages{newErr: errors.New("quota")}
	p := newTestProvider(m)

	_, err := p.AssessRisk(context.Background(), []sdk.PlanChange{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateImport_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("arn:aws:s3:::my-bucket"),
	}
	p := newTestProvider(m)

	result, err := p.GenerateImport(context.Background(), "aws_s3_bucket", "aws_s3_bucket.main")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "arn:aws:s3:::my-bucket" {
		t.Errorf("result = %q", result)
	}
}

func TestGenerateImport_WhenAPIError(t *testing.T) {
	m := &mockMessages{newErr: errors.New("no connection")}
	p := newTestProvider(m)

	_, err := p.GenerateImport(context.Background(), "type", "addr")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Stream method tests ---

func TestStreamExplainChange_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "Hello"}},
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: " world"}},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	if chunks[0] != "Hello" || chunks[1] != " world" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestStreamExplainChange_WhenStreamError(t *testing.T) {
	m := &mockMessages{
		streamErr: errors.New("connection reset"),
	}
	p := newTestProvider(m)

	err := p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	}, func(string) {})

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("error = %v, want connection reset", err)
	}
}

func TestStreamExplainChange_WhenNonDeltaEvents(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "message_start"},
			{Type: "content_block_start"},
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "only this"}},
			{Type: "content_block_stop"},
			{Type: "message_stop"},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "only this" {
		t.Errorf("chunks = %v, want [\"only this\"]", chunks)
	}
}

func TestStreamExplainChange_WhenEmptyDeltaText(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: ""}},
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "text"}},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "res.a", Type: "t"},
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "text" {
		t.Errorf("chunks = %v, want [\"text\"]", chunks)
	}
}

func TestStreamExplainPlan_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "Plan summary"}},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainPlan(context.Background(), &sdk.PlanSummary{ToCreate: 1}, func(s string) {
		chunks = append(chunks, s)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "Plan summary" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestStreamExplainPlan_WhenError(t *testing.T) {
	m := &mockMessages{streamErr: errors.New("timeout")}
	p := newTestProvider(m)

	err := p.StreamExplainPlan(context.Background(), &sdk.PlanSummary{}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamExplainResource_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "Resource info"}},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainResource(context.Background(), sdk.Resource{
		Address: "res.a", Type: "t", ProviderName: "p",
	}, "{}", func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "Resource info" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestStreamExplainResource_WhenError(t *testing.T) {
	m := &mockMessages{streamErr: errors.New("err")}
	p := newTestProvider(m)

	err := p.StreamExplainResource(context.Background(), sdk.Resource{
		Address: "a", Type: "t", ProviderName: "p",
	}, "{}", func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamAssessRisk_WhenSuccess(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "Risk: "}},
			{Type: "content_block_delta", Delta: anthropic.MessageStreamEventUnionDelta{Text: "high"}},
		},
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamAssessRisk(context.Background(), []sdk.PlanChange{
		{Action: sdk.ActionDelete, Resource: sdk.Resource{Address: "db.prod", Type: "aws_db"}, Risk: sdk.RiskCritical},
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	if chunks[0]+chunks[1] != "Risk: high" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestStreamAssessRisk_WhenError(t *testing.T) {
	m := &mockMessages{streamErr: errors.New("err")}
	p := newTestProvider(m)

	err := p.StreamAssessRisk(context.Background(), []sdk.PlanChange{}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Model parameter verification ---

func TestComplete_UsesCorrectModel(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("ok"),
	}
	p := &AnthropicProvider{
		messages: m,
		model:    "custom-model-v2",
		provider: ProviderAnthropic,
	}

	_, _ = p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "a", Type: "t"},
	})

	if m.lastParams.Model != "custom-model-v2" {
		t.Errorf("model = %q, want custom-model-v2", m.lastParams.Model)
	}
	if m.lastParams.MaxTokens != 1024 {
		t.Errorf("maxTokens = %d, want 1024", m.lastParams.MaxTokens)
	}
}

func TestComplete_UsesSystemPrompt(t *testing.T) {
	m := &mockMessages{
		newResult: textMessage("ok"),
	}
	p := newTestProvider(m)

	_, _ = p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "a", Type: "t"},
	})

	if len(m.lastParams.System) == 0 {
		t.Fatal("expected system prompt")
	}
	if m.lastParams.System[0].Text != systemPrompt {
		t.Errorf("system prompt mismatch")
	}
}

func TestStream_UsesCorrectModel(t *testing.T) {
	m := &mockMessages{
		streamData: []anthropic.MessageStreamEventUnion{},
	}
	p := &AnthropicProvider{
		messages: m,
		model:    "stream-model",
		provider: ProviderAnthropic,
	}

	_ = p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "a", Type: "t"},
	}, func(string) {})

	if m.lastParams.Model != "stream-model" {
		t.Errorf("model = %q, want stream-model", m.lastParams.Model)
	}
}

// --- Empty content tests ---

func TestExplainChange_WhenEmptyContent(t *testing.T) {
	m := &mockMessages{
		newResult: &anthropic.Message{Content: nil},
	}
	p := newTestProvider(m)

	result, err := p.ExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "a", Type: "t"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestStreamExplainChange_WhenNoEvents(t *testing.T) {
	m := &mockMessages{
		streamData: nil,
	}
	p := newTestProvider(m)

	var chunks []string
	err := p.StreamExplainChange(context.Background(), sdk.PlanChange{
		Action:   sdk.ActionCreate,
		Resource: sdk.Resource{Address: "a", Type: "t"},
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("chunks = %v, want empty", chunks)
	}
}

// --- Helpers ---

func makeDir(t *testing.T, path string) string {
	t.Helper()
	if err := makeDirectoryAll(path); err != nil {
		t.Fatalf("failed to create dir %s: %v", path, err)
	}
	return path
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := writeFileContent(path, content); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
	return path
}

func makeDirectoryAll(path string) error {
	return mkdirAll(path)
}

func mkdirAll(path string) error {
	return osWriteHelper(path, true, "")
}

func writeFileContent(path, content string) error {
	return osWriteHelper(path, false, content)
}

func osWriteHelper(path string, isDir bool, content string) error {
	if isDir {
		return os.MkdirAll(path, 0o755)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// Verify that AnthropicProvider satisfies the sdk.AIProvider interface.
var _ sdk.AIProvider = (*AnthropicProvider)(nil)

// Verify that MessageService satisfies the messageAPI interface.
var _ messageAPI = (*anthropic.MessageService)(nil)
