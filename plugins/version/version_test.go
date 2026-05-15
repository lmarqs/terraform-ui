package version

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	versionInfo *sdk.VersionInfo
	versionErr  error
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return nil, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error)  { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)       { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (m *mockService) StateRm(_ context.Context, _ string) error            { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error       { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error          { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error              { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error            { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (m *mockService) Refresh(_ context.Context) error                 { return nil }
func (m *mockService) Init(_ context.Context, _ sdk.InitOptions) error { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error   { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) {
	return m.versionInfo, m.versionErr
}
func (m *mockService) WithDir(_ string) sdk.Service { return m }

func TestPlugin_ID(t *testing.T) {
	p := New(&mockService{})
	if p.ID() != "version" {
		t.Errorf("ID() = %q, want %q", p.ID(), "version")
	}
}

func TestPlugin_Name(t *testing.T) {
	p := New(&mockService{})
	if p.Name() != "Version" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Version")
	}
}

func TestPlugin_Ready_WhenIdle(t *testing.T) {
	p := New(&mockService{})
	if p.Ready() {
		t.Error("Ready() = true before activation, want false")
	}
}

func TestPlugin_Configure_SetsTfuiVersion(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	_ = p.Configure(map[string]interface{}{"tfui_version": "1.2.3"})
	if p.version != "1.2.3" {
		t.Errorf("version = %q, want %q", p.version, "1.2.3")
	}
}

func TestPlugin_Configure_IgnoresUnknownKeys(t *testing.T) {
	p := New(&mockService{})
	err := p.Configure(map[string]interface{}{"unknown": true})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestPlugin_Activate_ReturnsCmd(t *testing.T) {
	svc := &mockService{versionInfo: &sdk.VersionInfo{TerraformVersion: "1.5.0"}}
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

func TestPlugin_Activate_WhenError(t *testing.T) {
	svc := &mockService{versionErr: errors.New("binary not found")}
	p := New(svc).(*Plugin)
	cmd := p.Activate()
	msg := cmd()
	result := msg.(VersionResultMsg)
	if result.Err == nil {
		t.Error("Err = nil, want error")
	}
}

func TestPlugin_Update_VersionResultSuccess(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
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

func TestPlugin_Update_VersionResultError(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
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

func TestPlugin_Update_EscDeactivates(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd produced %T, want DeactivateMsg", msg)
	}
}

func TestPlugin_Update_QDeactivates(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd produced %T, want DeactivateMsg", msg)
	}
}

func TestPlugin_View_Loading(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when loading")
	}
}

func TestPlugin_View_Done(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
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

func TestPlugin_View_Error(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
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

func TestPlugin_View_DoneWithoutProviders(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.version = "0.1.0"
	p.info = &sdk.VersionInfo{TerraformVersion: "1.14.9"}

	view := p.View(80, 24)
	if !contains(view, "terraform v1.14.9") {
		t.Error("View() missing terraform version")
	}
}

func TestPlugin_Hints(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() = %d items, want 1", len(hints))
	}
	if hints[0].Key != "q" {
		t.Errorf("Hints()[0].Key = %q, want %q", hints[0].Key, "q")
	}
}

func TestPlugin_Description(t *testing.T) {
	p := New(&mockService{})
	if p.Description() == "" {
		t.Error("Description() = empty")
	}
}

func TestPlugin_View_Idle(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusIdle
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() = empty when idle")
	}
}

func TestPlugin_View_NoVersion(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.info = &sdk.VersionInfo{TerraformVersion: "1.0.0"}
	view := p.View(80, 24)
	if !contains(view, "tfui vunknown") {
		t.Errorf("View() without configured version should show 'unknown', got: %s", view)
	}
}

func TestPlugin_Update_UnknownMsg(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	updated, cmd := p.Update(tea.MouseMsg{})
	if updated != p {
		t.Error("unknown msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unknown msg should return nil cmd")
	}
}

func TestPlugin_Init(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	cmd := p.Init(&sdk.Context{Service: svc})
	if cmd != nil {
		t.Error("Init() should return nil cmd")
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
