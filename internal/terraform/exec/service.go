package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const planFileName = "tfplan.out"

// ExecService implements the Service interface using hashicorp/terraform-exec
// to shell out to the terraform (or tofu) binary.
type ExecService struct {
	workingDir string
	binaryPath string
	statePath  string
	cache      *terraform.ServiceCache
	dirLock    *DirLock
}

// NewExecService creates an ExecService with the given cache.
func NewExecService(workingDir, binaryPath string, cache *terraform.ServiceCache) *ExecService {
	if cache == nil {
		cache = terraform.NewServiceCache()
	}
	return &ExecService{
		workingDir: workingDir,
		binaryPath: binaryPath,
		cache:      cache,
		dirLock:    NewDirLock(),
	}
}

// WithDir returns a new ExecService scoped to the given working directory.
func (s *ExecService) WithDir(dir string) sdk.Service {
	return &ExecService{
		workingDir: dir,
		binaryPath: s.binaryPath,
		statePath:  s.statePath,
		cache:      terraform.NewServiceCache(),
		dirLock:    s.dirLock,
	}
}

func (s *ExecService) loadState(ctx context.Context) (*tfjson.State, error) {
	if state, ok := s.cache.GetState(); ok {
		return state, nil
	}
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}
	state, err := tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading terraform state: %w", err)
	}
	var resources []sdk.Resource
	if state != nil && state.Values != nil {
		resources = terraform.ParseStateResources(state.Values.RootModule)
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
func (s *ExecService) Plan(ctx context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	logging.Logger().Debug("terraform.exec", "cmd", "plan", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("planning: %w", err)
	}

	planFilePath := opts.PlanFile
	if planFilePath == "" {
		planFilePath = filepath.Join(s.workingDir, planFileName)
	}

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planFilePath),
	}
	if opts.Writer != nil {
		tf.SetStdout(opts.Writer)
		tf.SetStderr(opts.Writer)
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
	switch opts.Refresh {
	case sdk.RefreshOnly:
		planOpts = append(planOpts, tfexec.RefreshOnly(true))
	case sdk.RefreshEnabled:
		planOpts = append(planOpts, tfexec.Refresh(true))
	case sdk.RefreshDisabled:
		planOpts = append(planOpts, tfexec.Refresh(false))
	case sdk.RefreshDefault:
	}
	if opts.Parallelism > 0 {
		planOpts = append(planOpts, tfexec.Parallelism(opts.Parallelism))
	}
	switch opts.Lock {
	case sdk.LockEnabled:
		planOpts = append(planOpts, tfexec.Lock(true))
	case sdk.LockDisabled:
		planOpts = append(planOpts, tfexec.Lock(false))
	case sdk.LockDefault:
	}
	if opts.LockTimeout != "" {
		planOpts = append(planOpts, tfexec.LockTimeout(string(opts.LockTimeout)))
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

	summary := terraform.ParsePlan(plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = terraform.ClassifyRisk(&summary.Changes[i])
	}

	terraform.DetectPhantomChanges(summary.Changes)

	logging.Logger().Debug("terraform.result", "cmd", "plan", "changes", len(summary.Changes), "duration", time.Since(start).String())
	return summary, nil
}

// Apply runs terraform apply. All options are passed through to terraform as-is.
func (s *ExecService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	logging.Logger().Debug("terraform.exec", "cmd", "apply", "dir", s.workingDir, "plan", opts.PlanFile, "targets", opts.Targets)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "apply", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("applying: %w", err)
	}

	if opts.Writer != nil {
		tf.SetStdout(opts.Writer)
		tf.SetStderr(opts.Writer)
	}

	var applyOpts []tfexec.ApplyOption
	if opts.PlanFile != "" {
		applyOpts = append(applyOpts, tfexec.DirOrPlan(opts.PlanFile))
	}
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

	logging.Logger().Debug("terraform.result", "cmd", "apply", "duration", time.Since(start).String())
	return nil
}

// Validate runs terraform validate.
func (s *ExecService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
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
				Severity: sdk.DiagnosticSeverity(d.Severity),
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
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
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
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
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
func (s *ExecService) Init(ctx context.Context, opts sdk.InitOptions) error {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	logging.Logger().Debug("terraform.exec", "cmd", "init", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	if opts.Writer != nil {
		tf.SetStdout(opts.Writer)
		tf.SetStderr(opts.Writer)
	}

	var initOpts []tfexec.InitOption
	if opts.Upgrade {
		initOpts = append(initOpts, tfexec.Upgrade(true))
	}
	if opts.Reconfigure {
		initOpts = append(initOpts, tfexec.Reconfigure(true))
	}
	switch opts.Backend {
	case sdk.BackendEnabled:
		initOpts = append(initOpts, tfexec.Backend(true))
	case sdk.BackendDisabled:
		initOpts = append(initOpts, tfexec.Backend(false))
	}
	for _, bc := range opts.BackendConfig {
		initOpts = append(initOpts, tfexec.BackendConfig(bc))
	}

	if err := tf.Init(ctx, initOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "init", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("running terraform init: %w", err)
	}

	s.cache.InvalidateAll()
	logging.Logger().Debug("terraform.result", "cmd", "init", "duration", time.Since(start).String())
	return nil
}

// Version returns the terraform binary version and provider selections.
func (s *ExecService) Version(ctx context.Context) (*sdk.VersionInfo, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
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
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
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

// runRawJSON invokes the terraform binary directly to capture raw `-json` byte
// output. tfexec's typed wrappers parse responses; we want the bytes verbatim.
func (s *ExecService) runRawJSON(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, args...)
	cmd.Dir = s.workingDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("terraform %s: %w: %s", args[0], err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// PlanJSON runs terraform plan, then `terraform show -json <planfile>`
// and returns the raw bytes terraform produced.
func (s *ExecService) PlanJSON(ctx context.Context, opts sdk.PlanOptions) ([]byte, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	logging.Logger().Debug("terraform.exec", "cmd", "plan-json", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return nil, fmt.Errorf("planning: %w", err)
	}

	planFilePath := opts.PlanFile
	if planFilePath == "" {
		planFilePath = filepath.Join(s.workingDir, planFileName)
	}

	planOpts := []tfexec.PlanOption{tfexec.Out(planFilePath)}
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
	switch opts.Refresh {
	case sdk.RefreshOnly:
		planOpts = append(planOpts, tfexec.RefreshOnly(true))
	case sdk.RefreshEnabled:
		planOpts = append(planOpts, tfexec.Refresh(true))
	case sdk.RefreshDisabled:
		planOpts = append(planOpts, tfexec.Refresh(false))
	case sdk.RefreshDefault:
	}
	if opts.Parallelism > 0 {
		planOpts = append(planOpts, tfexec.Parallelism(opts.Parallelism))
	}
	switch opts.Lock {
	case sdk.LockEnabled:
		planOpts = append(planOpts, tfexec.Lock(true))
	case sdk.LockDisabled:
		planOpts = append(planOpts, tfexec.Lock(false))
	case sdk.LockDefault:
	}
	if opts.LockTimeout != "" {
		planOpts = append(planOpts, tfexec.LockTimeout(string(opts.LockTimeout)))
	}

	if _, err := tf.Plan(ctx, planOpts...); err != nil {
		return nil, fmt.Errorf("running terraform plan: %w", err)
	}

	data, err := s.runRawJSON(ctx, "show", "-json", planFilePath)
	if err != nil {
		return nil, err
	}
	logging.Logger().Debug("terraform.result", "cmd", "plan-json", "bytes", len(data), "duration", time.Since(start).String())
	return data, nil
}

// ValidateJSON runs `terraform validate -json` and returns the raw bytes.
func (s *ExecService) ValidateJSON(ctx context.Context) ([]byte, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	return s.runRawJSON(ctx, "validate", "-json")
}

// OutputJSON runs `terraform output -json` and returns the raw bytes.
func (s *ExecService) OutputJSON(ctx context.Context) ([]byte, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	return s.runRawJSON(ctx, "output", "-json")
}

// VersionJSON runs `terraform version -json` and returns the raw bytes.
func (s *ExecService) VersionJSON(ctx context.Context) ([]byte, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	return s.runRawJSON(ctx, "version", "-json")
}
