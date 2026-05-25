package version

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newTestPlugin(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

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
	if cmd := p.Init(sdktest.NewDeps(svc).Deps); cmd != nil {
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
	p, _ := newTestPlugin(svc)
	cmd := p.Activate(Input{})
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
	p, _ := newTestPlugin(svc)
	cmd := p.Activate(Input{})
	msg := cmd()
	result := msg.(VersionResultMsg)
	if result.Err == nil {
		t.Error("Err = nil, want error")
	}
}

func TestActivate_WhenInputJSON_ShouldCallVersionJSONNotVersion(t *testing.T) {
	want := []byte(`{"terraform_version":"1.5.0","platform":"linux_amd64"}`)
	svc := &sdktest.MockService{
		VersionJSONFn: func(_ context.Context) ([]byte, error) {
			return want, nil
		},
	}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{JSON: true})
	if cmd == nil {
		t.Fatal("Activate(Input{JSON:true}) returned nil cmd")
	}
	msg := cmd()
	p.Update(msg)

	if svc.VersionCalls != 0 {
		t.Errorf("VersionCalls = %d, want 0 (JSON path must not call typed Version)", svc.VersionCalls)
	}
	if svc.VersionJSONCalls != 1 {
		t.Errorf("VersionJSONCalls = %d, want 1", svc.VersionJSONCalls)
	}
	if string(p.jsonBytes) != string(want) {
		t.Errorf("jsonBytes = %q, want %q", p.jsonBytes, want)
	}

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Stdout() error = %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("Stdout() = %q, want %q (verbatim passthrough)", data, want)
	}
}

func TestActivate_WhenInputJSONAndServiceFails_ShouldSetError(t *testing.T) {
	svc := &sdktest.MockService{
		VersionJSONFn: func(_ context.Context) ([]byte, error) {
			return nil, errors.New("version-json failed")
		},
	}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{JSON: true})
	msg := cmd()
	p.Update(msg)

	if p.status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", p.status)
	}
}

func TestUpdate_WhenVersionResultSuccess_ShouldSetDoneStatus(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when loading")
	}
}

func TestView_WhenDone_ShouldShowVersionAndProviders(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.version = "0.1.0"
	p.info = &sdk.VersionInfo{TerraformVersion: "1.14.9"}

	view := p.View(80, 24)
	if !contains(view, "terraform v1.14.9") {
		t.Error("View() missing terraform version")
	}
}

func TestHints_WhenCalled_ShouldReturnBackHint(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() = %d items, want 1", len(hints))
	}
	if hints[0].Key != "Esc" {
		t.Errorf("Hints()[0].Key = %q, want %q", hints[0].Key, "Esc")
	}
}

func TestView_WhenIdle_ShouldReturnNonEmpty(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when idle")
	}
}

func TestView_WhenNoVersionConfigured_ShouldShowUnknown(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.info = &sdk.VersionInfo{TerraformVersion: "1.0.0"}
	view := p.View(80, 24)
	if !contains(view, "tfui vunknown") {
		t.Errorf("View() without configured version should show 'unknown', got: %s", view)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfAndNil(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	updated, cmd := p.Update(tea.MouseMsg{})
	if updated != p {
		t.Error("unknown msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unknown msg should return nil cmd")
	}
}

func TestPlugin_WhenCancelCalledWithNilCancelFn_ShouldNotPanic(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelCalledWithActiveCancelFn_ShouldCallItAndClear(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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

func TestPlugin_WhenOutputJSON_ShouldPassthroughVerbatim(t *testing.T) {
	want := []byte(`{"terraform_version":"1.5.0","platform":"linux_amd64"}`)
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.input = Input{JSON: true}
	p.jsonBytes = want

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Stdout() error = %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("Stdout() = %q, want %q (verbatim passthrough)", data, want)
	}
}

func TestPlugin_WhenOutputJSONAndBytesNil_ShouldReturnNil(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.input = Input{JSON: true}
	p.jsonBytes = nil

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Stdout() error = %v", err)
	}
	if data != nil {
		t.Errorf("Stdout() = %q, want nil", string(data))
	}
}

func TestPlugin_WhenOutputText_ShouldReturnTfuiLineOnly(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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

func TestPlugin_WhenOutputTextWithEmptyVersion_ShouldShowUnknown(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.version = ""
	p.info = nil
	p.input = Input{JSON: false}

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Stdout() error = %v", err)
	}
	if !contains(string(data), "tfui vunknown") {
		t.Errorf("Stdout() = %s, want to contain %q", string(data), "tfui vunknown")
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
	p, _ := newTestPlugin(&sdktest.MockService{})
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

func TestOutput_WhenTextWithEmptyTerraformVersion_ShouldOmitTerraformLine(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.version = "1.2.3"
	p.info = &sdk.VersionInfo{TerraformVersion: ""}
	p.input = Input{JSON: false}

	data, err := p.Stdout()
	if err != nil {
		t.Fatalf("Stdout() error = %v", err)
	}
	if contains(string(data), "terraform v") {
		t.Errorf("text should not include terraform version when empty, got %q", string(data))
	}
}
