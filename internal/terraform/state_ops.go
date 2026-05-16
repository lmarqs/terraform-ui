package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// StateList returns all resources in the current state.
func (s *ExecService) StateList(ctx context.Context, opts ...sdk.StateListOption) ([]Resource, error) {
	cfg := sdk.ApplyStateListOptions(opts)
	if cfg.ShouldSkipCache() {
		s.cache.InvalidateState()
	} else if resources, ok := s.cache.GetResources(); ok {
		return resources, nil
	}

	state, err := s.loadState(ctx)
	if err != nil {
		return nil, err
	}

	if state == nil || state.Values == nil {
		return []Resource{}, nil
	}

	resources := ParseStateResources(state.Values.RootModule)
	return resources, nil
}

// Show returns detailed information about a specific resource.
// Sensitive attribute values are redacted before returning.
func (s *ExecService) Show(ctx context.Context, address string) (string, error) {
	state, err := s.loadState(ctx)
	if err != nil {
		return "", err
	}

	if state == nil || state.Values == nil {
		return "", fmt.Errorf("no state available")
	}

	resource := FindResourceInState(state.Values.RootModule, address)
	if resource == nil {
		return "", fmt.Errorf("resource %q not found in state", address)
	}

	redacted := RedactSensitiveValues(resource.AttributeValues, resource.SensitiveValues)

	display := struct {
		Address      string                 `json:"address"`
		Type         string                 `json:"type"`
		Name         string                 `json:"name"`
		ProviderName string                 `json:"provider_name"`
		Tainted      bool                   `json:"tainted,omitempty"`
		Values       map[string]interface{} `json:"values"`
	}{
		Address:      resource.Address,
		Type:         resource.Type,
		Name:         resource.Name,
		ProviderName: resource.ProviderName,
		Tainted:      resource.Tainted,
		Values:       redacted,
	}

	output, err := json.MarshalIndent(display, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling resource: %w", err)
	}

	return string(output), nil
}

// StateRm removes a resource from state.
func (s *ExecService) StateRm(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "state rm", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("removing from state: %w", err)
	}

	var rmOpts []tfexec.StateRmCmdOption
	if s.statePath != "" {
		rmOpts = append(rmOpts, tfexec.State(s.statePath)) //nolint:staticcheck // required for integration tests with custom state paths
	}
	if err := tf.StateRm(ctx, address, rmOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "state rm", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("removing %q from state: %w", address, err)
	}

	s.cache.InvalidateState()
	logging.Logger().Debug("terraform.result", "cmd", "state rm", "duration", time.Since(start).String())
	return nil
}

// StateMove moves a resource in state.
func (s *ExecService) StateMove(ctx context.Context, source, dest string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "state mv", "dir", s.workingDir, "source", source, "dest", dest)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("moving state: %w", err)
	}

	var mvOpts []tfexec.StateMvCmdOption
	if s.statePath != "" {
		mvOpts = append(mvOpts, tfexec.State(s.statePath)) //nolint:staticcheck // required for integration tests with custom state paths
	}
	if err := tf.StateMv(ctx, source, dest, mvOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "state mv", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("moving %q to %q: %w", source, dest, err)
	}

	s.cache.InvalidateState()
	logging.Logger().Debug("terraform.result", "cmd", "state mv", "duration", time.Since(start).String())
	return nil
}

// Import imports an existing resource into state.
func (s *ExecService) Import(ctx context.Context, address, id string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "import", "dir", s.workingDir, "address", address, "id", id)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("importing resource: %w", err)
	}

	var importOpts []tfexec.ImportOption
	if s.statePath != "" {
		importOpts = append(importOpts, tfexec.State(s.statePath)) //nolint:staticcheck // required for integration tests with custom state paths
	}
	if err := tf.Import(ctx, address, id, importOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "import", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("importing %q with id %q: %w", address, id, err)
	}

	s.cache.InvalidateState()
	logging.Logger().Debug("terraform.result", "cmd", "import", "duration", time.Since(start).String())
	return nil
}

// Taint marks a resource for recreation.
// Note: terraform taint is deprecated in newer versions of Terraform in favor of
// using -replace with plan/apply. terraform-exec still supports the command.
func (s *ExecService) Taint(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "taint", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("tainting resource: %w", err)
	}

	var taintOpts []tfexec.TaintOption
	if s.statePath != "" {
		taintOpts = append(taintOpts, tfexec.State(s.statePath)) //nolint:staticcheck // required for integration tests with custom state paths
	}
	if err := tf.Taint(ctx, address, taintOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "taint", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("tainting %q: %w", address, err)
	}

	s.cache.InvalidateState()
	logging.Logger().Debug("terraform.result", "cmd", "taint", "duration", time.Since(start).String())
	return nil
}

// Untaint removes taint from a resource.
// Note: terraform untaint is deprecated in newer versions of Terraform.
func (s *ExecService) Untaint(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "untaint", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("untainting resource: %w", err)
	}

	var untaintOpts []tfexec.UntaintOption
	if s.statePath != "" {
		untaintOpts = append(untaintOpts, tfexec.State(s.statePath)) //nolint:staticcheck // required for integration tests with custom state paths
	}
	if err := tf.Untaint(ctx, address, untaintOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "untaint", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("untainting %q: %w", address, err)
	}

	s.cache.InvalidateState()
	logging.Logger().Debug("terraform.result", "cmd", "untaint", "duration", time.Since(start).String())
	return nil
}

// ParseStateResources recursively extracts resources from a state module.
func ParseStateResources(module *tfjson.StateModule) []Resource {
	if module == nil {
		return []Resource{}
	}

	resources := make([]Resource, 0)

	for _, r := range module.Resources {
		resources = append(resources, Resource{
			Address:      r.Address,
			Type:         r.Type,
			Name:         r.Name,
			Module:       ExtractModule(r.Address),
			ProviderName: r.ProviderName,
			Tainted:      r.Tainted,
		})
	}

	for _, child := range module.ChildModules {
		resources = append(resources, ParseStateResources(child)...)
	}

	return resources
}

// FindResourceInState searches for a resource by address in the state module tree.
func FindResourceInState(module *tfjson.StateModule, address string) *tfjson.StateResource {
	if module == nil {
		return nil
	}

	for _, r := range module.Resources {
		if r.Address == address {
			return r
		}
	}

	for _, child := range module.ChildModules {
		if r := FindResourceInState(child, address); r != nil {
			return r
		}
	}

	return nil
}

// RedactSensitiveValues replaces sensitive attribute values with "(sensitive)".
func RedactSensitiveValues(values map[string]interface{}, sensitive interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	result := make(map[string]interface{}, len(values))
	for k, v := range values {
		if isSensitiveKey(sensitive, k) {
			result[k] = "(sensitive)"
		} else {
			result[k] = v
		}
	}
	return result
}

func isSensitiveKey(sensitive interface{}, key string) bool {
	if sensitive == nil {
		return false
	}
	switch s := sensitive.(type) {
	case bool:
		return s
	case map[string]interface{}:
		if v, ok := s[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
			return v != nil
		}
	}
	return false
}
