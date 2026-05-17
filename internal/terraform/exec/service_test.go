package exec

import (
	"testing"
)

func TestNewExecService(t *testing.T) {
	svc := NewExecService("/work/dir", "/usr/bin/terraform", nil)
	if svc.workingDir != "/work/dir" {
		t.Errorf("NewExecService().workingDir = %q, want %q", svc.workingDir, "/work/dir")
	}
	if svc.binaryPath != "/usr/bin/terraform" {
		t.Errorf("NewExecService().binaryPath = %q, want %q", svc.binaryPath, "/usr/bin/terraform")
	}
	if svc.cache == nil {
		t.Error("NewExecService(nil cache).cache should be initialized")
	}
}
