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

// InvalidateAll clears exec-sourced data, re-reads file-sourced data, and
// leaves stdin-sourced data unchanged.
func (c *ServiceCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.invalidatePlan()
	c.invalidateState()
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
		raw, err := os.ReadFile(c.planSource.filePath)
		if err != nil {
			c.plan = nil
			c.planSource = cacheSource{kind: sourceNone}
			return
		}
		plan, err := LoadPlan(raw)
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
