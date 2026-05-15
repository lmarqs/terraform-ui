package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const planFileName = "tfplan.out"

// Service is a type alias for the SDK Service interface. Internal packages and
// existing code can continue to reference terraform.Service. New code should
// prefer importing pkg/sdk directly.
type Service = sdk.Service

// ExecService implements the Service interface using hashicorp/terraform-exec
// to shell out to the terraform (or tofu) binary.
type ExecService struct {
	workingDir string
	binaryPath string
	statePath  string
	cache      *ServiceCache
}

// NewExecService creates an ExecService with the given cache.
func NewExecService(workingDir, binaryPath string, cache *ServiceCache) *ExecService {
	if cache == nil {
		cache = NewServiceCache()
	}
	return &ExecService{
		workingDir: workingDir,
		binaryPath: binaryPath,
		cache:      cache,
	}
}

// WithDir returns a new ExecService scoped to the given working directory.
func (s *ExecService) WithDir(dir string) Service {
	return &ExecService{
		workingDir: dir,
		binaryPath: s.binaryPath,
		statePath:  s.statePath,
		cache:      NewServiceCache(),
	}
}

func (s *ExecService) loadState(ctx context.Context) (*tfjson.State, error) {
	if state, ok := s.cache.GetState(); ok {
		return state, nil
	}
	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}
	state, err := tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading terraform state: %w", err)
	}
	var resources []Resource
	if state != nil && state.Values != nil {
		resources = ParseStateResources(state.Values.RootModule)
	}
	s.cache.SetState(resources, state)
	return state, nil
}

func (s *ExecService) newTerraform() (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(s.workingDir, s.binaryPath)
	if err != nil {
		return nil, fmt.Errorf("creating terraform instance: %w", err)
	}
	return tf, nil
}

// Plan runs terraform plan and returns the parsed changes.
func (s *ExecService) Plan(ctx context.Context, opts sdk.PlanOptions) (*PlanSummary, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "plan", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("planning: %w", err)
	}

	planFilePath := filepath.Join(s.workingDir, planFileName)

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planFilePath),
	}
	for _, t := range opts.Targets {
		planOpts = append(planOpts, tfexec.Target(t))
	}
	for _, f := range opts.VarFiles {
		planOpts = append(planOpts, tfexec.VarFile(f))
	}
	for k, v := range opts.Vars {
		planOpts = append(planOpts, tfexec.Var(k+"="+v))
	}
	for _, r := range opts.Replace {
		planOpts = append(planOpts, tfexec.Replace(r))
	}
	if opts.Destroy {
		planOpts = append(planOpts, tfexec.Destroy(true))
	}
	if opts.RefreshOnly {
		planOpts = append(planOpts, tfexec.RefreshOnly(true))
	}
	if opts.Refresh != nil {
		planOpts = append(planOpts, tfexec.Refresh(*opts.Refresh))
	}
	if opts.Parallelism > 0 {
		planOpts = append(planOpts, tfexec.Parallelism(opts.Parallelism))
	}
	if opts.Lock != nil {
		planOpts = append(planOpts, tfexec.Lock(*opts.Lock))
	}
	if opts.LockTimeout != "" {
		planOpts = append(planOpts, tfexec.LockTimeout(opts.LockTimeout))
	}

	_, err = tf.Plan(ctx, planOpts...)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("running terraform plan: %w", err)
	}

	plan, err := tf.ShowPlanFile(ctx, planFilePath)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("reading plan file: %w", err)
	}

	summary := ParsePlan(plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = ClassifyRisk(&summary.Changes[i])
	}

	DetectPhantomChanges(summary.Changes)

	logging.Logger().Debug("terraform.result", "cmd", "plan", "changes", len(summary.Changes), "duration", time.Since(start).String())
	return summary, nil
}

// Apply runs terraform apply on the saved plan file.
func (s *ExecService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	logging.Logger().Debug("terraform.exec", "cmd", "apply", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "apply", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("applying: %w", err)
	}

	planFilePath := filepath.Join(s.workingDir, planFileName)

	applyOpts := []tfexec.ApplyOption{tfexec.DirOrPlan(planFilePath)}
	for _, t := range opts.Targets {
		applyOpts = append(applyOpts, tfexec.Target(t))
	}
	for _, f := range opts.VarFiles {
		applyOpts = append(applyOpts, tfexec.VarFile(f))
	}
	for k, v := range opts.Vars {
		applyOpts = append(applyOpts, tfexec.Var(k+"="+v))
	}

	err = tf.Apply(ctx, applyOpts...)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "apply", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("running terraform apply: %w", err)
	}

	_ = os.Remove(planFilePath)
	logging.Logger().Debug("terraform.result", "cmd", "apply", "duration", time.Since(start).String())
	return nil
}

// Validate runs terraform validate.
func (s *ExecService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "validate", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return nil, fmt.Errorf("validating: %w", err)
	}

	validateOutput, err := tf.Validate(ctx)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "validate", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("running terraform validate: %w", err)
	}

	result := make([]sdk.Diagnostic, 0)
	if validateOutput != nil {
		for _, d := range validateOutput.Diagnostics {
			diag := sdk.Diagnostic{
				Severity: string(d.Severity),
				Summary:  d.Summary,
				Detail:   d.Detail,
			}
			if d.Range != nil {
				diag.File = d.Range.Filename
				diag.Line = d.Range.Start.Line
			}
			result = append(result, diag)
		}
	}

	logging.Logger().Debug("terraform.result", "cmd", "validate", "diagnostics", len(result), "duration", time.Since(start).String())
	return result, nil
}

// Output returns all terraform outputs.
func (s *ExecService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "output", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return nil, fmt.Errorf("getting output: %w", err)
	}

	outputs, err := tf.Output(ctx)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "output", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("getting outputs: %w", err)
	}

	result := make(map[string]sdk.OutputValue)
	for name, meta := range outputs {
		var value interface{}
		if len(meta.Value) > 0 {
			_ = json.Unmarshal(meta.Value, &value)
		}
		result[name] = sdk.OutputValue{
			Name:      name,
			Value:     value,
			Type:      string(meta.Type),
			Sensitive: meta.Sensitive,
		}
	}

	logging.Logger().Debug("terraform.result", "cmd", "output", "count", len(result), "duration", time.Since(start).String())
	return result, nil
}

// Refresh refreshes terraform state.
func (s *ExecService) Refresh(ctx context.Context) error {
	logging.Logger().Debug("terraform.exec", "cmd", "refresh", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("refreshing state: %w", err)
	}

	if err := tf.Refresh(ctx); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "refresh", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("refreshing state: %w", err)
	}

	s.cache.InvalidateAll()
	logging.Logger().Debug("terraform.result", "cmd", "refresh", "duration", time.Since(start).String())
	return nil
}

// Init runs terraform init.
func (s *ExecService) Init(ctx context.Context) error {
	logging.Logger().Debug("terraform.exec", "cmd", "init", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	if err := tf.Init(ctx); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "init", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("running terraform init: %w", err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "init", "duration", time.Since(start).String())
	return nil
}

// Version returns the terraform binary version and provider selections.
func (s *ExecService) Version(ctx context.Context) (*sdk.VersionInfo, error) {
	tf, err := s.newTerraform()
	if err != nil {
		return nil, fmt.Errorf("getting version: %w", err)
	}

	ver, provVersions, err := tf.Version(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("getting version: %w", err)
	}

	info := &sdk.VersionInfo{
		TerraformVersion: ver.String(),
	}
	if len(provVersions) > 0 {
		info.Providers = make(map[string]string, len(provVersions))
		for k, v := range provVersions {
			info.Providers[k] = v.String()
		}
	}
	return info, nil
}

// ForceUnlock removes a state lock by ID.
func (s *ExecService) ForceUnlock(ctx context.Context, lockID string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "force-unlock", "dir", s.workingDir, "lockID", lockID)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("force-unlocking: %w", err)
	}

	if err := tf.ForceUnlock(ctx, lockID); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "force-unlock", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("force-unlocking state (ID %s): %w", lockID, err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "force-unlock", "duration", time.Since(start).String())
	return nil
}
