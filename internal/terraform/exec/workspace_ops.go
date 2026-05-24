package exec

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Workspace returns the current workspace name.
func (s *ExecService) Workspace(ctx context.Context) (string, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return "", fmt.Errorf("getting workspace: %w", err)
	}

	workspace, err := tf.WorkspaceShow(ctx)
	if err != nil {
		return "", fmt.Errorf("getting current workspace: %w", err)
	}

	return workspace, nil
}

// WorkspaceList returns all workspace names.
func (s *ExecService) WorkspaceList(ctx context.Context) ([]string, error) {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	workspaces, current, err := tf.WorkspaceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	// Ensure the current workspace is included in the list
	found := false
	for _, ws := range workspaces {
		if ws == current {
			found = true
			break
		}
	}
	if !found {
		workspaces = append(workspaces, current)
	}

	return workspaces, nil
}

// WorkspaceSelect switches to the specified workspace.
func (s *ExecService) WorkspaceSelect(ctx context.Context, name string) error {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("selecting workspace: %w", err)
	}

	if err := tf.WorkspaceSelect(ctx, name); err != nil {
		return fmt.Errorf("selecting workspace %q: %w", name, err)
	}
	s.cache.InvalidateAll()
	return nil
}

// WorkspaceNew creates a new workspace and switches to it.
func (s *ExecService) WorkspaceNew(ctx context.Context, name string, opts sdk.WorkspaceNewOptions) error {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	var tfOpts []tfexec.WorkspaceNewCmdOption
	switch opts.Lock {
	case sdk.LockEnabled:
		tfOpts = append(tfOpts, tfexec.Lock(true))
	case sdk.LockDisabled:
		tfOpts = append(tfOpts, tfexec.Lock(false))
	case sdk.LockDefault:
	}
	if opts.LockTimeout != "" {
		tfOpts = append(tfOpts, tfexec.LockTimeout(string(opts.LockTimeout)))
	}

	if err := tf.WorkspaceNew(ctx, name, tfOpts...); err != nil {
		return fmt.Errorf("creating workspace %q: %w", name, err)
	}
	return nil
}

// WorkspaceDelete deletes the specified workspace.
func (s *ExecService) WorkspaceDelete(ctx context.Context, name string, opts sdk.WorkspaceDeleteOptions) error {
	s.dirLock.Acquire(s.workingDir)
	defer s.dirLock.Release(s.workingDir)
	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	var tfOpts []tfexec.WorkspaceDeleteCmdOption
	if opts.Force {
		tfOpts = append(tfOpts, tfexec.Force(true))
	}
	switch opts.Lock {
	case sdk.LockEnabled:
		tfOpts = append(tfOpts, tfexec.Lock(true))
	case sdk.LockDisabled:
		tfOpts = append(tfOpts, tfexec.Lock(false))
	case sdk.LockDefault:
	}
	if opts.LockTimeout != "" {
		tfOpts = append(tfOpts, tfexec.LockTimeout(string(opts.LockTimeout)))
	}

	if err := tf.WorkspaceDelete(ctx, name, tfOpts...); err != nil {
		return fmt.Errorf("deleting workspace %q: %w", name, err)
	}
	return nil
}
