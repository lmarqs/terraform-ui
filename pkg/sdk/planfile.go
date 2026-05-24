package sdk

import "os"

// PlanFile represents a reference to a terraform plan file on disk.
// It distinguishes between temporary plan files (owned by tfui, cleaned up
// automatically) and user-provided plan files (never deleted).
type PlanFile struct {
	path string
	temp bool
}

// NewTempPlanFile creates a PlanFile that will be removed on Cleanup.
func NewTempPlanFile(path string) PlanFile {
	return PlanFile{path: path, temp: true}
}

// NewUserPlanFile creates a PlanFile that is never removed on Cleanup.
func NewUserPlanFile(path string) PlanFile {
	return PlanFile{path: path, temp: false}
}

// Path returns the filesystem path to the plan file.
func (f PlanFile) Path() string {
	return f.path
}

// Cleanup removes the plan file if it is a temporary file.
// For user-provided files this is a no-op. Safe to call on a zero-value PlanFile.
func (f PlanFile) Cleanup() {
	if f.temp && f.path != "" {
		_ = os.Remove(f.path)
	}
}

// IsZero reports whether this is an uninitialized PlanFile.
func (f PlanFile) IsZero() bool {
	return f.path == ""
}
