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

func (r *RecordingService) Apply(_ context.Context, opts sdk.ApplyOptions) error {
	r.record("apply", nil, buildApplyFlags(opts))
	return nil
}

func (r *RecordingService) StateList(ctx context.Context) ([]sdk.Resource, error) {
	return r.inner.StateList(ctx)
}

func (r *RecordingService) Show(ctx context.Context, address string) (string, error) {
	return r.inner.Show(ctx, address)
}

func (r *RecordingService) Workspace(ctx context.Context) (string, error) {
	return r.inner.Workspace(ctx)
}

func (r *RecordingService) WorkspaceList(ctx context.Context) ([]string, error) {
	return r.inner.WorkspaceList(ctx)
}

func (r *RecordingService) WorkspaceSelect(_ context.Context, name string) error {
	r.record("workspace select", []string{name}, nil)
	return nil
}

func (r *RecordingService) WorkspaceNew(_ context.Context, name string) error {
	r.record("workspace new", []string{name}, nil)
	return nil
}

func (r *RecordingService) WorkspaceDelete(_ context.Context, name string) error {
	r.record("workspace delete", []string{name}, nil)
	return nil
}

func (r *RecordingService) StateRm(_ context.Context, address string) error {
	r.record("state rm", []string{address}, nil)
	return nil
}

func (r *RecordingService) StateMove(_ context.Context, src, dst string) error {
	r.record("state mv", []string{src, dst}, nil)
	return nil
}

func (r *RecordingService) Import(_ context.Context, address, id string) error {
	r.record("import", []string{address, id}, nil)
	return nil
}

func (r *RecordingService) Taint(_ context.Context, address string) error {
	r.record("taint", []string{address}, nil)
	return nil
}

func (r *RecordingService) Untaint(_ context.Context, address string) error {
	r.record("untaint", []string{address}, nil)
	return nil
}

func (r *RecordingService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	return r.inner.Validate(ctx)
}

func (r *RecordingService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	return r.inner.Output(ctx)
}

func (r *RecordingService) Refresh(_ context.Context) error {
	r.record("refresh", nil, nil)
	return nil
}

func (r *RecordingService) Init(_ context.Context) error {
	r.record("init", nil, nil)
	return nil
}

func (r *RecordingService) ForceUnlock(_ context.Context, lockID string) error {
	r.record("force-unlock", nil, []string{"-force", lockID})
	return nil
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
