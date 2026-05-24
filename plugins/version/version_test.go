package version

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "version" {
		t.Errorf("ID() = %q, want %q", p.ID(), "version")
	}
	if p.Name() != "Version" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Version")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{"tfui_version": "1.0.0", "unknown": true}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(&sdk.PluginDeps{Service: svc}); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestActivate_WhenServiceSucceeds_ShouldReturnVersionResult(t *testing.T) {
	svc := &sdktest.MockService{
		VersionFn: func(_ context.Context) (*sdk.VersionInfo, error) {
			return &sdk.VersionInfo{TerraformVersion: "1.5.0"}, nil
		},
	}
	p := New(svc).(*Plugin)
	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("Activate() returned nil, want cmd")
	}
	msg := cmd()
	result, ok := msg.(VersionResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want VersionResultMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}
	if result.Info.TerraformVersion != "1.5.0" {
		t.Errorf("TerraformVersion = %q, want %q", result.Info.TerraformVersion, "1.5.0")
	}
}

func TestActivate_WhenServiceFails_ShouldReturnError(t *testing.T) {
	svc := &sdktest.MockService{
		VersionFn: func(_ context.Context) (*sdk.VersionInfo, error) {
			return nil, errors.New("binary not found")
		},
	}
	p := New(svc).(*Plugin)
	cmd := p.Activate()
	msg := cmd()
	result := msg.(VersionResultMsg)
	if result.Err == nil {
		t.Error("Err = nil, want error")
	}
}

func TestUpdate_WhenVersionResultSuccess_ShouldSetDoneStatus(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.version = "0.1.0"

	info := &sdk.VersionInfo{
		TerraformVersion: "1.14.9",
		Providers: map[string]string{
			"registry.terraform.io/hashicorp/aws": "5.0.0",
		},
	}

	updated, _ := p.Update(VersionResultMsg{Info: info, Err: nil})
	pp := updated.(*Plugin)
	if pp.status != sdk.StatusDone {
		t.Errorf("status = %v, want StatusDone", pp.status)
	}
	if !pp.Ready() {
		t.Error("Ready() = false, want true")
	}
}

func TestUpdate_WhenVersionResultError_ShouldSetErrorStatus(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	updated, _ := p.Update(VersionResultMsg{Err: errors.New("failed")})
	pp := updated.(*Plugin)
	if pp.status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", pp.status)
	}
	if pp.errMsg != "failed" {
		t.Errorf("errMsg = %q, want %q", pp.errMsg, "failed")
	}
}

func TestUpdate_WhenEscPressed_ShouldDeactivate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd produced %T, want DeactivateMsg", msg)
	}
}

func TestUpdate_WhenQPressed_ShouldDeactivate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd produced %T, want DeactivateMsg", msg)
	}
}

func TestView_WhenLoading_ShouldReturnNonEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when loading")
	}
}

func TestView_WhenDone_ShouldShowVersionAndProviders(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.version = "0.1.0"
	p.info = &sdk.VersionInfo{
		TerraformVersion: "1.14.9",
		Providers: map[string]string{
			"registry.terraform.io/hashicorp/aws":  "5.0.0",
			"registry.terraform.io/hashicorp/null": "3.2.1",
		},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Fatal("View() = empty when done")
	}
	for _, want := range []string{"tfui v0.1.0", "terraform v1.14.9", "hashicorp/aws", "hashicorp/null"} {
		if !contains(view, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestView_WhenError_ShouldShowTfuiVersionAndError(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError
	p.version = "0.1.0"
	p.errMsg = "binary not found"

	view := p.View(80, 24)
	if !contains(view, "tfui v0.1.0") {
		t.Error("View() should still show tfui version on error")
	}
	if !contains(view, "binary not found") {
		t.Error("View() should show error message")
	}
}

func TestView_WhenDoneWithoutProviders_ShouldShowVersionOnly(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.version = "0.1.0"
	p.info = &sdk.VersionInfo{TerraformVersion: "1.14.9"}

	view := p.View(80, 24)
	if !contains(view, "terraform v1.14.9") {
		t.Error("View() missing terraform version")
	}
}

func TestHints_WhenCalled_ShouldReturnBackHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() = %d items, want 1", len(hints))
	}
	if hints[0].Key != "Esc" {
		t.Errorf("Hints()[0].Key = %q, want %q", hints[0].Key, "Esc")
	}
}

func TestView_WhenIdle_ShouldReturnNonEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when idle")
	}
}

func TestView_WhenNoVersionConfigured_ShouldShowUnknown(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.info = &sdk.VersionInfo{TerraformVersion: "1.0.0"}
	view := p.View(80, 24)
	if !contains(view, "tfui vunknown") {
		t.Errorf("View() without configured version should show 'unknown', got: %s", view)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfAndNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	updated, cmd := p.Update(tea.MouseMsg{})
	if updated != p {
		t.Error("unknown msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unknown msg should return nil cmd")
	}
}

func TestPlugin_WhenCancelCalledWithNilCancelFn_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelCalledWithActiveCancelFn_ShouldCallItAndClear(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should set cancelFn to nil")
	}
}

func TestPlugin_WhenOutputJSON_ShouldReturnTfuiVersionAndPlatformOnly(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.0.0"
	p.info = nil

	p.SetJSONOutput(true)

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !contains(s, `"tfui_version": "1.0.0"`) {
		t.Errorf("Output(true) missing tfui_version, got: %s", s)
	}
	if !contains(s, `"platform"`) {
		t.Errorf("Output(true) missing platform, got: %s", s)
	}
	if contains(s, `"terraform_version"`) {
		t.Errorf("Output(true) should not contain terraform_version when info is nil, got: %s", s)
	}
	if contains(s, `"provider_selections"`) {
		t.Errorf("Output(true) should not contain provider_selections when info is nil, got: %s", s)
	}
}

func TestPlugin_WhenOutputJSON_ShouldReturnFullJSONWithProviders(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.0.0"
	p.info = &sdk.VersionInfo{
		TerraformVersion: "1.5.0",
		Providers: map[string]string{
			"registry.terraform.io/hashicorp/aws": "5.0.0",
		},
	}

	p.SetJSONOutput(true)

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !contains(s, `"terraform_version": "1.5.0"`) {
		t.Errorf("Output(true) missing terraform_version, got: %s", s)
	}
	if !contains(s, `"provider_selections"`) {
		t.Errorf("Output(true) missing provider_selections, got: %s", s)
	}
	if !contains(s, `"registry.terraform.io/hashicorp/aws": "5.0.0"`) {
		t.Errorf("Output(true) missing provider entry, got: %s", s)
	}
}

func TestPlugin_WhenOutputText_ShouldReturnTfuiLineOnly(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.0.0"
	p.info = nil

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !contains(s, "tfui v1.0.0") {
		t.Errorf("Output(false) missing tfui version line, got: %s", s)
	}
	if contains(s, "terraform v") {
		t.Errorf("Output(false) should not contain terraform version when info is nil, got: %s", s)
	}
}

func TestPlugin_WhenOutputText_ShouldReturnFullTextWithProviders(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.0.0"
	p.info = &sdk.VersionInfo{
		TerraformVersion: "1.5.0",
		Providers: map[string]string{
			"registry.terraform.io/hashicorp/aws":  "5.0.0",
			"registry.terraform.io/hashicorp/null": "3.2.1",
		},
	}

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !contains(s, "tfui v1.0.0") {
		t.Errorf("Output(false) missing tfui version, got: %s", s)
	}
	if !contains(s, "terraform v1.5.0") {
		t.Errorf("Output(false) missing terraform version, got: %s", s)
	}
	if !contains(s, "provider registry.terraform.io/hashicorp/aws v5.0.0") {
		t.Errorf("Output(false) missing aws provider, got: %s", s)
	}
	if !contains(s, "provider registry.terraform.io/hashicorp/null v3.2.1") {
		t.Errorf("Output(false) missing null provider, got: %s", s)
	}
}

func TestPlugin_WhenOutputWithEmptyVersion_ShouldShowUnknown(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = ""
	p.info = nil

	tests := []struct {
		name       string
		jsonOutput bool
		want       string
	}{
		{"ShouldShowUnknownInJSON", true, "unknown"},
		{"ShouldShowUnknownInText", false, "tfui vunknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.SetJSONOutput(tt.jsonOutput)
			data, err := p.Stdout()
			if err != nil {
				t.Fatalf("Stdout(%v) error = %v", tt.jsonOutput, err)
			}
			if !contains(string(data), tt.want) {
				t.Errorf("Stdout(%v) = %s, want to contain %q", tt.jsonOutput, string(data), tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestOutput_WhenTextWithNilInfo_ShouldShowOnlyTfuiVersion(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.2.3"
	p.info = nil

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !contains(s, "tfui v1.2.3") {
		t.Errorf("expected 'tfui v1.2.3', got %q", s)
	}
	if contains(s, "terraform v") {
		t.Error("should not show terraform version when info is nil")
	}
}

func TestOutput_WhenJsonWithNilInfo_ShouldOmitTerraformFields(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.2.3"
	p.info = nil

	p.SetJSONOutput(true)

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !contains(s, `"tfui_version": "1.2.3"`) {
		t.Errorf("expected tfui_version in JSON, got %q", s)
	}
	if contains(s, `"terraform_version"`) {
		t.Error("should not include terraform_version when info is nil")
	}
}

func TestOutput_WhenInfoHasEmptyTerraformVersion_ShouldOmitTerraformFields(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.version = "1.2.3"
	p.info = &sdk.VersionInfo{TerraformVersion: ""}

	p.SetJSONOutput(true)

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if contains(s, `"terraform_version"`) {
		t.Errorf("should not include terraform_version when it's empty, got %q", s)
	}

	p.SetJSONOutput(false)
	data, err = p.Stdout()
	if err != nil {
		t.Fatalf("Stdout(false) error = %v", err)
	}
	s = string(data)
	if contains(s, "terraform v") {
		t.Errorf("text should not include terraform version when empty, got %q", s)
	}
}
