package sdk

import (
	"log/slog"
	"sync"
	"time"
)

// AppContext is the root context for the entire application.
// It's the single source of truth, partitioned by domain.
type AppContext struct {
	Project   ProjectContext
	Config    *ConfigContext
	Terraform *TerraformContext
	UI        *UIContext
	Cache     *CacheContext
	AI        AIProvider // nil if disabled
	Logger    *slog.Logger
}

// ProjectContext holds immutable project-level info.
type ProjectContext struct {
	Dir            string   // absolute path to root working directory
	Scopes         []string // discovered terraform sub-projects (relative paths)
	ActiveScope    string   // currently selected sub-project (relative path)
	ActiveScopeAbs string   // absolute path of active scope
}

// TerraformContext holds all terraform operational state.
type TerraformContext struct {
	mu            sync.RWMutex
	WorkingDir    string
	Workspace     string
	PinnedTargets []string
	State         *TerraformState
	Plan          *TerraformPlan
	Service       Service
}

// TerraformState holds cached terraform state and metadata.
type TerraformState struct {
	Resources     []Resource
	LastRefreshed time.Time
	Loading       bool
	Error         error
}

// TerraformPlan holds cached plan results and metadata.
type TerraformPlan struct {
	Summary       *PlanSummary
	LastRefreshed time.Time
	Loading       bool
	Error         error
	FilePath      string
}

// Pin adds an address to pinned targets (no-op if already pinned).
func (t *TerraformContext) Pin(address string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, a := range t.PinnedTargets {
		if a == address {
			return
		}
	}
	t.PinnedTargets = append(t.PinnedTargets, address)
}

// Unpin removes an address from pinned targets.
func (t *TerraformContext) Unpin(address string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, a := range t.PinnedTargets {
		if a == address {
			t.PinnedTargets = append(t.PinnedTargets[:i], t.PinnedTargets[i+1:]...)
			return
		}
	}
}

// ClearPins removes all pinned targets.
func (t *TerraformContext) ClearPins() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.PinnedTargets = nil
}

// IsPinned checks if an address is in the pinned targets.
func (t *TerraformContext) IsPinned(address string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, a := range t.PinnedTargets {
		if a == address {
			return true
		}
	}
	return false
}

// PinnedCount returns the number of pinned targets.
func (t *TerraformContext) PinnedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.PinnedTargets)
}

// GetPinned returns a copy of the pinned targets slice.
func (t *TerraformContext) GetPinned() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]string, len(t.PinnedTargets))
	copy(result, t.PinnedTargets)
	return result
}

// SetState updates the cached terraform state.
func (t *TerraformContext) SetState(resources []Resource) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.State = &TerraformState{
		Resources:     resources,
		LastRefreshed: time.Now(),
	}
}

// SetPlan updates the cached plan summary.
func (t *TerraformContext) SetPlan(summary *PlanSummary, filePath string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Plan = &TerraformPlan{
		Summary:       summary,
		LastRefreshed: time.Now(),
		FilePath:      filePath,
	}
}

// InvalidatePlan clears the cached plan.
func (t *TerraformContext) InvalidatePlan() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Plan = nil
}

// InvalidateState clears the cached state.
func (t *TerraformContext) InvalidateState() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.State = nil
}

// UIContext holds current UI state visible to all plugins.
type UIContext struct {
	mu           sync.RWMutex
	Width        int
	Height       int
	ActivePlugin string
	InputMode    InputMode
}

// InputMode represents the current input mode of the application.
type InputMode int

const (
	InputModeNormal  InputMode = iota
	InputModeCommand           // ":"
	InputModeFilter            // "/"
	InputModePrompt            // waiting for user confirmation
	InputModeREPL              // terraform console
)

// SetSize updates the UI dimensions.
func (u *UIContext) SetSize(width, height int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Width = width
	u.Height = height
}

// GetSize returns the current UI dimensions.
func (u *UIContext) GetSize() (int, int) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Width, u.Height
}

// SetActivePlugin updates the currently active plugin ID.
func (u *UIContext) SetActivePlugin(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.ActivePlugin = id
}

// GetActivePlugin returns the currently active plugin ID.
func (u *UIContext) GetActivePlugin() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.ActivePlugin
}

// SetInputMode updates the current input mode.
func (u *UIContext) SetInputMode(mode InputMode) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.InputMode = mode
}

// GetInputMode returns the current input mode.
func (u *UIContext) GetInputMode() InputMode {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.InputMode
}


// NewAppContext creates a new AppContext with initialized sub-contexts.
func NewAppContext(dir, workspace string, svc Service, logger *slog.Logger) *AppContext {
	return &AppContext{
		Project: ProjectContext{
			Dir: dir,
		},
		Config: NewConfigContext(nil),
		Terraform: &TerraformContext{
			WorkingDir: dir,
			Workspace:  workspace,
			Service:    svc,
		},
		UI:     &UIContext{},
		Cache:  NewCacheContext(),
		Logger: logger,
	}
}
