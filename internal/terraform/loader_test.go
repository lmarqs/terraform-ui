package terraform

import (
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
	t.Run("valid plan JSON", func(t *testing.T) {
		summary, err := LoadPlan([]byte(validPlanJSON))
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
		summary, err := LoadPlan([]byte(validPlanJSON))
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
		summary, err := LoadPlan([]byte(`{"format_version":"1.2","resource_changes":[]}`))
		if err != nil {
			t.Fatal(err)
		}
		if len(summary.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(summary.Changes))
		}
	})

	t.Run("null resource_changes", func(t *testing.T) {
		summary, err := LoadPlan([]byte(`{"format_version":"1.2","resource_changes":null}`))
		if err != nil {
			t.Fatal(err)
		}
		if len(summary.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(summary.Changes))
		}
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := LoadPlan([]byte(`not json`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("error: binary plan file", func(t *testing.T) {
		_, err := LoadPlan([]byte{0x00, 0x01, 0x02, 0x03})
		if err == nil {
			t.Error("expected error for binary plan")
		}
	})
}

func TestLoadState(t *testing.T) {
	t.Run("valid show-json state", func(t *testing.T) {
		resources, state, err := LoadState([]byte(validStateJSON))
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
		resources, state, err := LoadState([]byte(`{"format_version":"1.0","values":null}`))
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
		resources, _, err := LoadState([]byte(`{"format_version":"1.0"}`))
		if err != nil {
			t.Fatal(err)
		}
		if len(resources) != 0 {
			t.Errorf("len(resources) = %d, want 0", len(resources))
		}
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, _, err := LoadState([]byte(`{broken`))
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestParseRawState(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantResources  int
		wantAddresses  []string
		wantProviders  []string
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "flat state with single resource",
			input: `{
				"version": 4,
				"resources": [{
					"mode": "managed",
					"type": "aws_instance",
					"name": "web",
					"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
					"instances": [{"index_key": null, "attributes": {"id": "i-123"}}]
				}]
			}`,
			wantResources: 1,
			wantAddresses: []string{"aws_instance.web"},
			wantProviders: []string{"registry.terraform.io/hashicorp/aws"},
		},
		{
			name: "count resources with numeric index",
			input: `{
				"version": 4,
				"resources": [{
					"mode": "managed",
					"type": "aws_subnet",
					"name": "public",
					"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
					"instances": [
						{"index_key": 0, "attributes": {"id": "subnet-0"}},
						{"index_key": 1, "attributes": {"id": "subnet-1"}}
					]
				}]
			}`,
			wantResources: 2,
			wantAddresses: []string{"aws_subnet.public[0]", "aws_subnet.public[1]"},
		},
		{
			name: "for_each with string keys",
			input: `{
				"version": 4,
				"resources": [{
					"mode": "managed",
					"type": "aws_subnet",
					"name": "region",
					"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
					"instances": [
						{"index_key": "us-east-1", "attributes": {"id": "subnet-a"}},
						{"index_key": "eu-west-1", "attributes": {"id": "subnet-b"}}
					]
				}]
			}`,
			wantResources: 2,
			wantAddresses: []string{`aws_subnet.region["us-east-1"]`, `aws_subnet.region["eu-west-1"]`},
		},
		{
			name: "nested module",
			input: `{
				"version": 4,
				"resources": [{
					"module": "module.vpc",
					"mode": "managed",
					"type": "aws_vpc",
					"name": "main",
					"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
					"instances": [{"index_key": null, "attributes": {"id": "vpc-123"}}]
				}]
			}`,
			wantResources: 1,
			wantAddresses: []string{"module.vpc.aws_vpc.main"},
		},
		{
			name: "data sources excluded",
			input: `{
				"version": 4,
				"resources": [
					{
						"mode": "data",
						"type": "aws_ami",
						"name": "latest",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": [{"index_key": null, "attributes": {"id": "ami-123"}}]
					},
					{
						"mode": "managed",
						"type": "aws_instance",
						"name": "web",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": [{"index_key": null, "attributes": {"id": "i-123"}}]
					}
				]
			}`,
			wantResources: 1,
			wantAddresses: []string{"aws_instance.web"},
		},
		{
			name:          "empty state with no resources",
			input:         `{"version": 4, "resources": []}`,
			wantResources: 0,
			wantAddresses: []string{},
		},
		{
			name:           "invalid JSON",
			input:          `{not valid`,
			wantErr:        true,
			wantErrContain: "parsing state",
		},
		{
			name:           "missing version field",
			input:          `{"resources": []}`,
			wantErr:        true,
			wantErrContain: "no version field",
		},
		{
			name: "provider cleaning",
			input: `{
				"version": 4,
				"resources": [{
					"mode": "managed",
					"type": "google_compute_instance",
					"name": "vm",
					"provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
					"instances": [{"index_key": null, "attributes": {"id": "vm-1"}}]
				}]
			}`,
			wantResources: 1,
			wantProviders: []string{"registry.terraform.io/hashicorp/google"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, state, err := parseRawState([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.wantErrContain != "" && !contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if len(resources) != tt.wantResources {
				t.Fatalf("len(resources) = %d, want %d", len(resources), tt.wantResources)
			}

			if state == nil && tt.wantResources > 0 {
				t.Fatal("state should not be nil")
			}

			for i, want := range tt.wantAddresses {
				if i >= len(resources) {
					break
				}
				if resources[i].Address != want {
					t.Errorf("resources[%d].Address = %q, want %q", i, resources[i].Address, want)
				}
			}

			for i, want := range tt.wantProviders {
				if i >= len(resources) {
					break
				}
				if resources[i].ProviderName != want {
					t.Errorf("resources[%d].ProviderName = %q, want %q", i, resources[i].ProviderName, want)
				}
			}
		})
	}
}

func TestBuildAddress(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		resType  string
		resName  string
		indexKey interface{}
		want     string
	}{
		{"simple", "", "aws_instance", "web", nil, "aws_instance.web"},
		{"with module", "module.vpc", "aws_subnet", "a", nil, "module.vpc.aws_subnet.a"},
		{"count index", "", "aws_instance", "web", float64(0), "aws_instance.web[0]"},
		{"for_each key", "", "aws_instance", "web", "us-east-1", `aws_instance.web["us-east-1"]`},
		{"key with special chars", "", "aws_instance", "web", "a/b", `aws_instance.web["a/b"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAddress(tt.module, tt.resType, tt.resName, tt.indexKey)
			if got != tt.want {
				t.Errorf("buildAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseShowState_WhenStateHasInvalidJSON_ShouldReturnError(t *testing.T) {
	_, _, err := parseShowState([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseShowState_WhenStateHasNilValues_ShouldReturnEmptyResources(t *testing.T) {
	data := []byte(`{"format_version":"1.0","values":null}`)
	resources, state, err := parseShowState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("len(resources) = %d, want 0", len(resources))
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestParseShowState_WhenStateHasResources_ShouldReturnParsedResources(t *testing.T) {
	data := []byte(`{
		"format_version":"1.0",
		"values": {
			"root_module": {
				"resources": [{
					"address": "aws_instance.web",
					"type": "aws_instance",
					"name": "web",
					"provider_name": "registry.terraform.io/hashicorp/aws",
					"values": {"id": "i-123"},
					"sensitive_values": {}
				}]
			}
		}
	}`)
	resources, state, err := parseShowState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	if resources[0].Address != "aws_instance.web" {
		t.Errorf("address = %q, want aws_instance.web", resources[0].Address)
	}
	if state == nil || state.Values == nil {
		t.Fatal("state.Values should not be nil")
	}
}
