package sdk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestNewTempPlanFile(t *testing.T) {
	t.Run("Path returns the provided path", func(t *testing.T) {
		f := sdk.NewTempPlanFile("/tmp/plan.out")
		if f.Path() != "/tmp/plan.out" {
			t.Errorf("Path() = %q, want %q", f.Path(), "/tmp/plan.out")
		}
	})

	t.Run("IsZero returns false", func(t *testing.T) {
		f := sdk.NewTempPlanFile("/tmp/plan.out")
		if f.IsZero() {
			t.Error("IsZero() = true, want false")
		}
	})

	t.Run("Cleanup removes the file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "plan.out")
		if err := os.WriteFile(path, []byte("plan data"), 0o600); err != nil {
			t.Fatal(err)
		}

		f := sdk.NewTempPlanFile(path)
		f.Cleanup()

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("file still exists after Cleanup, err = %v", err)
		}
	})

	t.Run("Cleanup on non-existent file does not panic", func(t *testing.T) {
		f := sdk.NewTempPlanFile("/tmp/does-not-exist-ever-12345.out")
		f.Cleanup() // must not panic
	})
}

func TestNewUserPlanFile(t *testing.T) {
	t.Run("Path returns the provided path", func(t *testing.T) {
		f := sdk.NewUserPlanFile("/home/user/saved.plan")
		if f.Path() != "/home/user/saved.plan" {
			t.Errorf("Path() = %q, want %q", f.Path(), "/home/user/saved.plan")
		}
	})

	t.Run("IsZero returns false", func(t *testing.T) {
		f := sdk.NewUserPlanFile("/home/user/saved.plan")
		if f.IsZero() {
			t.Error("IsZero() = true, want false")
		}
	})

	t.Run("Cleanup does not remove the file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "user.plan")
		if err := os.WriteFile(path, []byte("plan data"), 0o600); err != nil {
			t.Fatal(err)
		}

		f := sdk.NewUserPlanFile(path)
		f.Cleanup()

		if _, err := os.Stat(path); err != nil {
			t.Errorf("file was removed after Cleanup, err = %v", err)
		}
	})
}

func TestPlanFile_ZeroValue(t *testing.T) {
	var f sdk.PlanFile

	t.Run("Path returns empty string", func(t *testing.T) {
		if f.Path() != "" {
			t.Errorf("Path() = %q, want empty", f.Path())
		}
	})

	t.Run("IsZero returns true", func(t *testing.T) {
		if !f.IsZero() {
			t.Error("IsZero() = false, want true")
		}
	})

	t.Run("Cleanup does not panic", func(t *testing.T) {
		f.Cleanup() // must not panic
	})
}
