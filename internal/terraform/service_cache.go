package terraform

import (
	"fmt"
	"os"
	"sync"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type sourceKind int

const (
	sourceNone sourceKind = iota
	sourceFile
	sourceStdin
	sourceExec
)

type cacheSource struct {
	kind     sourceKind
	filePath string
	data     []byte
}

// ServiceCache is a typed, thread-safe cache for terraform service data.
type ServiceCache struct {
	mu                sync.RWMutex
	plan              *sdk.PlanSummary
	planSource        cacheSource
	resources         []sdk.Resource
	state             *tfjson.State
	stateSource       cacheSource
	outputs           map[string]sdk.OutputValue
	outputsSource     cacheSource
	diagnostics       []sdk.Diagnostic
	diagnosticsSource cacheSource
	workspaces        []string
	workspacesSource  cacheSource
}

// NewServiceCache returns an empty ServiceCache.
func NewServiceCache() *ServiceCache {
	return &ServiceCache{}
}

// SeedPlan seeds the cache from a file path OR raw bytes (stdin).
// If filePath != "", reads the file and sets sourceFile.
// If data != nil, uses data and sets sourceStdin.
// File takes precedence if both are provided.
func (c *ServiceCache) SeedPlan(filePath string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if filePath != "" {
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading plan file: %w", err)
		}
		plan, err := LoadPlan(raw)
		if err != nil {
			return err
		}
		c.plan = plan
		c.planSource = cacheSource{kind: sourceFile, filePath: filePath}
		return nil
	}

	if data != nil {
		plan, err := LoadPlan(data)
		if err != nil {
			return err
		}
		c.plan = plan
		c.planSource = cacheSource{kind: sourceStdin, data: data}
		return nil
	}

	return nil
}

// SeedState seeds the cache from a file path OR raw bytes (stdin).
// If filePath != "", reads the file and sets sourceFile.
// If data != nil, uses data and sets sourceStdin.
// File takes precedence if both are provided.
func (c *ServiceCache) SeedState(filePath string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if filePath != "" {
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading state file: %w", err)
		}
		resources, state, err := LoadState(raw)
		if err != nil {
			return err
		}
		c.resources = resources
		c.state = state
		c.stateSource = cacheSource{kind: sourceFile, filePath: filePath}
		return nil
	}

	if data != nil {
		resources, state, err := LoadState(data)
		if err != nil {
			return err
		}
		c.resources = resources
		c.state = state
		c.stateSource = cacheSource{kind: sourceStdin, data: data}
		return nil
	}

	return nil
}

// GetPlan returns the cached plan or (nil, false) if not available.
func (c *ServiceCache) GetPlan() (*sdk.PlanSummary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.planSource.kind == sourceNone {
		return nil, false
	}
	return c.plan, c.plan != nil
}

// SeededPlan returns the cached plan, read from its file again when reread is
// set; see ReloadPlan. A file source without a plan (a failed re-read) is read
// again even when reread is not set, so a -plan session never falls through to
// running terraform plan.
func (c *ServiceCache) SeededPlan(reread bool) (*sdk.PlanSummary, bool, error) {
	if reread || c.fileSourceWithoutPlan() {
		return c.ReloadPlan()
	}
	plan, ok := c.GetPlan()
	return plan, ok, nil
}

func (c *ServiceCache) fileSourceWithoutPlan() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.planSource.kind == sourceFile && c.plan == nil
}

// ReloadPlan is GetPlan for an explicit refresh: a file-sourced plan is read
// from its file again. A read or parse error comes back with ok true and no
// plan: the cached plan is dropped so nothing keeps serving it, and the file
// stays the source so a later call reads it again. Other sources keep their plan.
func (c *ServiceCache) ReloadPlan() (*sdk.PlanSummary, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case c.planSource.kind == sourceFile:
		plan, err := readPlanFile(c.planSource.filePath)
		c.plan = plan
		return plan, true, err
	case c.planSource.kind == sourceNone || c.plan == nil:
		return nil, false, nil
	default:
		return c.plan, true, nil
	}
}

func readPlanFile(path string) (*sdk.PlanSummary, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plan file: %w", err)
	}
	return LoadPlan(raw)
}

// PlanSnapshot reports whether the plan was seeded from stdin and so cannot be read again.
func (c *ServiceCache) PlanSnapshot() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.planSource.kind == sourceStdin
}

// SetPlan stores a plan from execution, setting sourceExec.
func (c *ServiceCache) SetPlan(summary *sdk.PlanSummary) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.plan = summary
	c.planSource = cacheSource{kind: sourceExec}
}

// GetResources returns the cached resources or (nil, false) if not available.
func (c *ServiceCache) GetResources() ([]sdk.Resource, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stateSource.kind == sourceNone {
		return nil, false
	}
	return c.resources, c.resources != nil
}

// GetState returns the cached tfjson.State or (nil, false) if not available.
func (c *ServiceCache) GetState() (*tfjson.State, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stateSource.kind == sourceNone {
		return nil, false
	}
	return c.state, c.state != nil
}

// SetState stores resources and state from execution, setting sourceExec.
func (c *ServiceCache) SetState(resources []sdk.Resource, state *tfjson.State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resources = resources
	c.state = state
	c.stateSource = cacheSource{kind: sourceExec}
}

// GetOutputs returns the cached outputs or (nil, false) if not available.
func (c *ServiceCache) GetOutputs() (map[string]sdk.OutputValue, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.outputsSource.kind == sourceNone {
		return nil, false
	}
	return c.outputs, c.outputs != nil
}

// SetOutputs stores outputs from execution, setting sourceExec.
func (c *ServiceCache) SetOutputs(outputs map[string]sdk.OutputValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outputs = outputs
	c.outputsSource = cacheSource{kind: sourceExec}
}

// GetDiagnostics returns the cached diagnostics or (nil, false) if not available.
func (c *ServiceCache) GetDiagnostics() ([]sdk.Diagnostic, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.diagnosticsSource.kind == sourceNone {
		return nil, false
	}
	return c.diagnostics, c.diagnostics != nil
}

// SetDiagnostics stores diagnostics from execution, setting sourceExec.
func (c *ServiceCache) SetDiagnostics(diagnostics []sdk.Diagnostic) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.diagnostics = diagnostics
	c.diagnosticsSource = cacheSource{kind: sourceExec}
}

// GetWorkspaces returns the cached workspaces or (nil, false) if not available.
func (c *ServiceCache) GetWorkspaces() ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.workspacesSource.kind == sourceNone {
		return nil, false
	}
	return c.workspaces, c.workspaces != nil
}

// SetWorkspaces stores workspaces from execution, setting sourceExec.
func (c *ServiceCache) SetWorkspaces(workspaces []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workspaces = workspaces
	c.workspacesSource = cacheSource{kind: sourceExec}
}

// InvalidateAll clears exec-sourced data, re-reads file-sourced data, and
// leaves stdin-sourced data unchanged.
func (c *ServiceCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.invalidatePlan()
	c.invalidateState()
	c.invalidateOutputs()
	c.invalidateDiagnostics()
	c.invalidateWorkspaces()
}

// InvalidateState clears/re-reads only state data, leaving plan unchanged.
func (c *ServiceCache) InvalidateState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.invalidateState()
}

// Clear wipes all cached data regardless of source.
func (c *ServiceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.plan = nil
	c.planSource = cacheSource{kind: sourceNone}
	c.resources = nil
	c.state = nil
	c.stateSource = cacheSource{kind: sourceNone}
	c.outputs = nil
	c.outputsSource = cacheSource{kind: sourceNone}
	c.diagnostics = nil
	c.diagnosticsSource = cacheSource{kind: sourceNone}
	c.workspaces = nil
	c.workspacesSource = cacheSource{kind: sourceNone}
}

func (c *ServiceCache) invalidatePlan() {
	switch c.planSource.kind {
	case sourceExec:
		c.plan = nil
		c.planSource = cacheSource{kind: sourceNone}
	case sourceFile:
		plan, err := readPlanFile(c.planSource.filePath)
		if err != nil {
			c.plan = nil
			c.planSource = cacheSource{kind: sourceNone}
			return
		}
		c.plan = plan
	case sourceStdin:
		// stdin data is immutable, cannot re-read
	case sourceNone:
		// nothing to do
	}
}

func (c *ServiceCache) invalidateState() {
	switch c.stateSource.kind {
	case sourceExec:
		c.resources = nil
		c.state = nil
		c.stateSource = cacheSource{kind: sourceNone}
	case sourceFile:
		raw, err := os.ReadFile(c.stateSource.filePath)
		if err != nil {
			c.resources = nil
			c.state = nil
			c.stateSource = cacheSource{kind: sourceNone}
			return
		}
		resources, state, err := LoadState(raw)
		if err != nil {
			c.resources = nil
			c.state = nil
			c.stateSource = cacheSource{kind: sourceNone}
			return
		}
		c.resources = resources
		c.state = state
	case sourceStdin:
		// stdin data is immutable, cannot re-read
	case sourceNone:
		// nothing to do
	}
}

func (c *ServiceCache) invalidateOutputs() {
	switch c.outputsSource.kind {
	case sourceExec:
		c.outputs = nil
		c.outputsSource = cacheSource{kind: sourceNone}
	case sourceStdin:
		// stdin data is immutable, cannot re-read
	case sourceNone:
		// nothing to do
	}
}

func (c *ServiceCache) invalidateDiagnostics() {
	switch c.diagnosticsSource.kind {
	case sourceExec:
		c.diagnostics = nil
		c.diagnosticsSource = cacheSource{kind: sourceNone}
	case sourceStdin:
		// stdin data is immutable, cannot re-read
	case sourceNone:
		// nothing to do
	}
}

func (c *ServiceCache) invalidateWorkspaces() {
	switch c.workspacesSource.kind {
	case sourceExec:
		c.workspaces = nil
		c.workspacesSource = cacheSource{kind: sourceNone}
	case sourceStdin:
		// stdin data is immutable, cannot re-read
	case sourceNone:
		// nothing to do
	}
}
