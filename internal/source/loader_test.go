package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const validPlanJSON = `{
  "format_version": "1.2",
  "terraform_version": "1.5.0",
  "resource_changes": [
    {
      "address": "aws_instance.web",
      "type": "aws_instance",
      "name": "web",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {"ami": "ami-abc123", "instance_type": "t3.micro"},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    },
    {
      "address": "aws_s3_bucket.data",
      "type": "aws_s3_bucket",
      "name": "data",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {"bucket": "old-name"},
        "after": {"bucket": "new-name"},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    },
    {
      "address": "aws_iam_role.noop",
      "type": "aws_iam_role",
      "name": "noop",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["no-op"],
        "before": {"name": "role"},
        "after": {"name": "role"},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    }
  ]
}`

const validStateJSON = `{
  "format_version": "1.0",
  "terraform_version": "1.5.0",
  "values": {
    "root_module": {
      "resources": [
        {
          "address": "aws_instance.web",
          "type": "aws_instance",
          "name": "web",
          "provider_name": "registry.terraform.io/hashicorp/aws",
          "values": {"id": "i-123456", "ami": "ami-abc123", "instance_type": "t3.micro"},
          "sensitive_values": {}
        },
        {
          "address": "aws_s3_bucket.data",
          "type": "aws_s3_bucket",
          "name": "data",
          "provider_name": "registry.terraform.io/hashicorp/aws",
          "values": {"bucket": "my-bucket", "arn": "arn:aws:s3:::my-bucket"},
          "sensitive_values": {}
        }
      ],
      "child_modules": [
        {
          "address": "module.vpc",
          "resources": [
            {
              "address": "module.vpc.aws_vpc.main",
              "type": "aws_vpc",
              "name": "main",
              "provider_name": "registry.terraform.io/hashicorp/aws",
              "values": {"id": "vpc-abc", "cidr_block": "10.0.0.0/16"},
              "sensitive_values": {}
            }
          ]
        }
      ]
    }
  }
}`

func TestLoadPlan(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	t.Run("valid plan JSON", func(t *testing.T) {
		path := filepath.Join(dir, "plan.json")
		os.WriteFile(path, []byte(validPlanJSON), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		summary, err := LoadPlan(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}

		if summary.ToCreate != 1 {
			t.Errorf("ToCreate = %d, want 1", summary.ToCreate)
		}
		if summary.ToUpdate != 1 {
			t.Errorf("ToUpdate = %d, want 1", summary.ToUpdate)
		}
		if len(summary.Changes) != 2 {
			t.Fatalf("len(Changes) = %d, want 2 (no-op filtered)", len(summary.Changes))
		}
		if summary.Changes[0].Resource.Address != "aws_instance.web" {
			t.Errorf("Changes[0].Address = %q", summary.Changes[0].Resource.Address)
		}
		if summary.Changes[0].Action != "create" {
			t.Errorf("Changes[0].Action = %q", summary.Changes[0].Action)
		}
	})

	t.Run("plan with risk classification", func(t *testing.T) {
		path := filepath.Join(dir, "plan.json")
		os.WriteFile(path, []byte(validPlanJSON), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		summary, err := LoadPlan(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}

		for _, change := range summary.Changes {
			if change.Risk < 0 {
				t.Errorf("risk should be >= 0 for %s", change.Resource.Address)
			}
		}
	})

	t.Run("empty plan", func(t *testing.T) {
		path := filepath.Join(dir, "empty.json")
		os.WriteFile(path, []byte(`{"format_version":"1.2","resource_changes":[]}`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		summary, err := LoadPlan(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}
		if len(summary.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(summary.Changes))
		}
	})

	t.Run("null resource_changes", func(t *testing.T) {
		path := filepath.Join(dir, "null.json")
		os.WriteFile(path, []byte(`{"format_version":"1.2","resource_changes":null}`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		summary, err := LoadPlan(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}
		if len(summary.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(summary.Changes))
		}
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		path := filepath.Join(dir, "invalid.json")
		os.WriteFile(path, []byte(`not json`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		_, err := LoadPlan(ctx, resolver, path)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("error: file not found", func(t *testing.T) {
		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		_, err := LoadPlan(ctx, resolver, "/nonexistent/plan.json")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("error: binary plan file", func(t *testing.T) {
		path := filepath.Join(dir, "binary.tfplan")
		os.WriteFile(path, []byte{0x00, 0x01, 0x02, 0x03}, 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		_, err := LoadPlan(ctx, resolver, path)
		if err == nil {
			t.Error("expected error for binary plan")
		}
	})
}

func TestLoadState(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	t.Run("valid state JSON", func(t *testing.T) {
		path := filepath.Join(dir, "state.json")
		os.WriteFile(path, []byte(validStateJSON), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		resources, state, err := LoadState(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}

		if len(resources) != 3 {
			t.Fatalf("len(resources) = %d, want 3 (2 root + 1 child module)", len(resources))
		}
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if state.Values == nil {
			t.Fatal("state.Values should not be nil")
		}
	})

	t.Run("state with no values", func(t *testing.T) {
		path := filepath.Join(dir, "empty-state.json")
		os.WriteFile(path, []byte(`{"format_version":"1.0","values":null}`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		resources, state, err := LoadState(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}
		if len(resources) != 0 {
			t.Errorf("len(resources) = %d, want 0", len(resources))
		}
		if state == nil {
			t.Fatal("state should not be nil even with empty values")
		}
	})

	t.Run("empty state (format version only)", func(t *testing.T) {
		path := filepath.Join(dir, "minimal.json")
		os.WriteFile(path, []byte(`{"format_version":"1.0"}`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		resources, _, err := LoadState(ctx, resolver, path)
		if err != nil {
			t.Fatal(err)
		}
		if len(resources) != 0 {
			t.Errorf("len(resources) = %d, want 0", len(resources))
		}
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		path := filepath.Join(dir, "invalid.json")
		os.WriteFile(path, []byte(`{broken`), 0644)

		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		_, _, err := LoadState(ctx, resolver, path)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("error: file not found", func(t *testing.T) {
		resolver := NewResolver(&LocalProvider{BaseDir: dir})
		_, _, err := LoadState(ctx, resolver, "/nonexistent/state.json")
		if err == nil {
			t.Error("expected error")
		}
	})
}
