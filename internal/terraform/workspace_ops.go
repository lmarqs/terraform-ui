package terraform

import (
	"context"
	"fmt"
)

// Workspace returns the current workspace name.
func (s *ExecService) Workspace(ctx context.Context) (string, error) {
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
func (s *ExecService) WorkspaceNew(ctx context.Context, name string) error {
	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	if err := tf.WorkspaceNew(ctx, name); err != nil {
		return fmt.Errorf("creating workspace %q: %w", name, err)
	}
	return nil
}

// WorkspaceDelete deletes the specified workspace.
func (s *ExecService) WorkspaceDelete(ctx context.Context, name string) error {
	tf, err := s.newTerraform()
	if err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	if err := tf.WorkspaceDelete(ctx, name); err != nil {
		return fmt.Errorf("deleting workspace %q: %w", name, err)
	}
	return nil
}
