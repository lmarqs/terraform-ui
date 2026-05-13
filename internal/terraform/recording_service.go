package terraform

import (
	"context"
	"sync"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// CommandKind classifies a recorded command as read-only or mutating.
type CommandKind int

const (
	CommandRead CommandKind = iota
	CommandMutate
)

// CommandFilter controls which commands are returned by Commands().
// Return true to include the command.
type CommandFilter func(sdk.Command, CommandKind) bool

// MutateOnly includes only mutating commands (apply, state rm, taint, etc.).
func MutateOnly(_ sdk.Command, kind CommandKind) bool {
	return kind == CommandMutate
}

type recordedEntry struct {
	cmd  sdk.Command
	kind CommandKind
}

type commandStore struct {
	mu      sync.Mutex
	entries []recordedEntry
}

func (s *commandStore) append(entry recordedEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
}

func (s *commandStore) commands(filter CommandFilter) []sdk.Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		return nil
	}
	var result []sdk.Command
	for _, e := range s.entries {
		if filter == nil || filter(e.cmd, e.kind) {
			result = append(result, e.cmd)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
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

// Commands returns recorded commands, optionally filtered.
// Pass nil to get all commands. Returns nil when nothing was recorded
// or when the filter matches nothing.
func (r *RecordingService) Commands(filter CommandFilter) []sdk.Command {
	return r.store.commands(filter)
}

func (r *RecordingService) record(kind CommandKind, verb string, args, flags []string) {
	r.store.append(recordedEntry{
		cmd: sdk.Command{
			Binary: r.binary,
			Verb:   verb,
			Args:   args,
			Flags:  flags,
		},
		kind: kind,
	})
}

func (r *RecordingService) Plan(ctx context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	r.record(CommandRead, "plan", nil, buildPlanFlags(opts))
	return r.inner.Plan(ctx, opts)
}

func (r *RecordingService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	r.record(CommandMutate, "apply", nil, buildApplyFlags(opts))
	return r.inner.Apply(ctx, opts)
}

func (r *RecordingService) StateList(ctx context.Context) ([]sdk.Resource, error) {
	r.record(CommandRead, "state list", nil, nil)
	return r.inner.StateList(ctx)
}

func (r *RecordingService) Show(ctx context.Context, address string) (string, error) {
	r.record(CommandRead, "state show", []string{address}, nil)
	return r.inner.Show(ctx, address)
}

func (r *RecordingService) Workspace(ctx context.Context) (string, error) {
	r.record(CommandRead, "workspace show", nil, nil)
	return r.inner.Workspace(ctx)
}

func (r *RecordingService) WorkspaceList(ctx context.Context) ([]string, error) {
	r.record(CommandRead, "workspace list", nil, nil)
	return r.inner.WorkspaceList(ctx)
}

func (r *RecordingService) WorkspaceSelect(ctx context.Context, name string) error {
	r.record(CommandMutate, "workspace select", []string{name}, nil)
	return r.inner.WorkspaceSelect(ctx, name)
}

func (r *RecordingService) WorkspaceNew(ctx context.Context, name string) error {
	r.record(CommandMutate, "workspace new", []string{name}, nil)
	return r.inner.WorkspaceNew(ctx, name)
}

func (r *RecordingService) WorkspaceDelete(ctx context.Context, name string) error {
	r.record(CommandMutate, "workspace delete", []string{name}, nil)
	return r.inner.WorkspaceDelete(ctx, name)
}

func (r *RecordingService) StateRm(ctx context.Context, address string) error {
	r.record(CommandMutate, "state rm", []string{address}, nil)
	return r.inner.StateRm(ctx, address)
}

func (r *RecordingService) StateMove(ctx context.Context, src, dst string) error {
	r.record(CommandMutate, "state mv", []string{src, dst}, nil)
	return r.inner.StateMove(ctx, src, dst)
}

func (r *RecordingService) Import(ctx context.Context, address, id string) error {
	r.record(CommandMutate, "import", []string{address, id}, nil)
	return r.inner.Import(ctx, address, id)
}

func (r *RecordingService) Taint(ctx context.Context, address string) error {
	r.record(CommandMutate, "taint", []string{address}, nil)
	return r.inner.Taint(ctx, address)
}

func (r *RecordingService) Untaint(ctx context.Context, address string) error {
	r.record(CommandMutate, "untaint", []string{address}, nil)
	return r.inner.Untaint(ctx, address)
}

func (r *RecordingService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	r.record(CommandRead, "validate", nil, nil)
	return r.inner.Validate(ctx)
}

func (r *RecordingService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	r.record(CommandRead, "output", nil, nil)
	return r.inner.Output(ctx)
}

func (r *RecordingService) Refresh(ctx context.Context) error {
	r.record(CommandMutate, "refresh", nil, nil)
	return r.inner.Refresh(ctx)
}

func (r *RecordingService) Init(ctx context.Context) error {
	r.record(CommandMutate, "init", nil, nil)
	return r.inner.Init(ctx)
}

func (r *RecordingService) ForceUnlock(ctx context.Context, lockID string) error {
	r.record(CommandMutate, "force-unlock", nil, []string{"-force", lockID})
	return r.inner.ForceUnlock(ctx, lockID)
}

func (r *RecordingService) WithDir(dir string) sdk.Service {
	return &RecordingService{
		inner:  r.inner.WithDir(dir),
		binary: r.binary,
		store:  r.store,
	}
}
