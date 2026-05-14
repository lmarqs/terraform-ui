package terraform

import (
	"context"
	"fmt"
	"sync"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const defaultBinary = "terraform"

type commandStore struct {
	mu       sync.Mutex
	commands []sdk.Command
}

func (s *commandStore) append(cmd sdk.Command) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, cmd)
}

func (s *commandStore) all() []sdk.Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commands
}

// RecordingService wraps any sdk.Service and records every operation as a
// sdk.Command. The recorded commands can be retrieved via Commands().
type RecordingService struct {
	inner  sdk.Service
	binary string
	store  *commandStore
}

// NewRecordingService creates a recording decorator around inner.
// Binary sets the terraform binary name in recorded commands.
func NewRecordingService(inner sdk.Service, binary string) *RecordingService {
	if binary == "" {
		binary = defaultBinary
	}
	return &RecordingService{
		inner:  inner,
		binary: binary,
		store:  &commandStore{},
	}
}

// Commands returns all recorded commands in order.
// Returns nil when nothing was recorded.
func (r *RecordingService) Commands() []sdk.Command {
	return r.store.all()
}

func (r *RecordingService) record(verb string, args, flags []string) {
	r.store.append(sdk.Command{
		Binary: r.binary,
		Verb:   verb,
		Args:   args,
		Flags:  flags,
	})
}

func (r *RecordingService) Plan(ctx context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	r.record("plan", nil, buildPlanFlags(opts))
	return r.inner.Plan(ctx, opts)
}

func (r *RecordingService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	r.record("apply", nil, buildApplyFlags(opts))
	return r.inner.Apply(ctx, opts)
}

func (r *RecordingService) StateList(ctx context.Context) ([]sdk.Resource, error) {
	r.record("state list", nil, nil)
	return r.inner.StateList(ctx)
}

func (r *RecordingService) Show(ctx context.Context, address string) (string, error) {
	r.record("state show", []string{address}, nil)
	return r.inner.Show(ctx, address)
}

func (r *RecordingService) Workspace(ctx context.Context) (string, error) {
	r.record("workspace show", nil, nil)
	return r.inner.Workspace(ctx)
}

func (r *RecordingService) WorkspaceList(ctx context.Context) ([]string, error) {
	r.record("workspace list", nil, nil)
	return r.inner.WorkspaceList(ctx)
}

func (r *RecordingService) WorkspaceSelect(ctx context.Context, name string) error {
	r.record("workspace select", []string{name}, nil)
	return r.inner.WorkspaceSelect(ctx, name)
}

func (r *RecordingService) WorkspaceNew(ctx context.Context, name string) error {
	r.record("workspace new", []string{name}, nil)
	return r.inner.WorkspaceNew(ctx, name)
}

func (r *RecordingService) WorkspaceDelete(ctx context.Context, name string) error {
	r.record("workspace delete", []string{name}, nil)
	return r.inner.WorkspaceDelete(ctx, name)
}

func (r *RecordingService) StateRm(ctx context.Context, address string) error {
	r.record("state rm", []string{address}, nil)
	return r.inner.StateRm(ctx, address)
}

func (r *RecordingService) StateMove(ctx context.Context, src, dst string) error {
	r.record("state mv", []string{src, dst}, nil)
	return r.inner.StateMove(ctx, src, dst)
}

func (r *RecordingService) Import(ctx context.Context, address, id string) error {
	r.record("import", []string{address, id}, nil)
	return r.inner.Import(ctx, address, id)
}

func (r *RecordingService) Taint(ctx context.Context, address string) error {
	r.record("taint", []string{address}, nil)
	return r.inner.Taint(ctx, address)
}

func (r *RecordingService) Untaint(ctx context.Context, address string) error {
	r.record("untaint", []string{address}, nil)
	return r.inner.Untaint(ctx, address)
}

func (r *RecordingService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	r.record("validate", nil, nil)
	return r.inner.Validate(ctx)
}

func (r *RecordingService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	r.record("output", nil, nil)
	return r.inner.Output(ctx)
}

func (r *RecordingService) Refresh(ctx context.Context) error {
	r.record("refresh", nil, nil)
	return r.inner.Refresh(ctx)
}

func (r *RecordingService) Init(ctx context.Context) error {
	r.record("init", nil, nil)
	return r.inner.Init(ctx)
}

func (r *RecordingService) ForceUnlock(ctx context.Context, lockID string) error {
	r.record("force-unlock", nil, []string{"-force", lockID})
	return r.inner.ForceUnlock(ctx, lockID)
}

func (r *RecordingService) WithDir(dir string) sdk.Service {
	return &RecordingService{
		inner:  r.inner.WithDir(dir),
		binary: r.binary,
		store:  r.store,
	}
}

func buildPlanFlags(opts sdk.PlanOptions) []string {
	var flags []string
	for _, t := range opts.Targets {
		flags = append(flags, "-target="+t)
	}
	for _, f := range opts.VarFiles {
		flags = append(flags, "-var-file="+f)
	}
	for k, v := range opts.Vars {
		flags = append(flags, "-var", k+"="+v)
	}
	for _, r := range opts.Replace {
		flags = append(flags, "-replace="+r)
	}
	if opts.Destroy {
		flags = append(flags, "-destroy")
	}
	if opts.RefreshOnly {
		flags = append(flags, "-refresh-only")
	}
	if opts.Refresh != nil && !*opts.Refresh {
		flags = append(flags, "-refresh=false")
	}
	if opts.Parallelism > 0 {
		flags = append(flags, fmt.Sprintf("-parallelism=%d", opts.Parallelism))
	}
	if opts.Lock != nil && !*opts.Lock {
		flags = append(flags, "-lock=false")
	}
	if opts.LockTimeout != "" {
		flags = append(flags, "-lock-timeout="+opts.LockTimeout)
	}
	flags = append(flags, opts.ExtraArgs...)
	return flags
}

func buildApplyFlags(opts sdk.ApplyOptions) []string {
	var flags []string
	for _, t := range opts.Targets {
		flags = append(flags, "-target="+t)
	}
	for _, f := range opts.VarFiles {
		flags = append(flags, "-var-file="+f)
	}
	for k, v := range opts.Vars {
		flags = append(flags, "-var", k+"="+v)
	}
	if opts.Parallelism > 0 {
		flags = append(flags, fmt.Sprintf("-parallelism=%d", opts.Parallelism))
	}
	if opts.Lock != nil && !*opts.Lock {
		flags = append(flags, "-lock=false")
	}
	if opts.LockTimeout != "" {
		flags = append(flags, "-lock-timeout="+opts.LockTimeout)
	}
	flags = append(flags, opts.ExtraArgs...)
	return flags
}
