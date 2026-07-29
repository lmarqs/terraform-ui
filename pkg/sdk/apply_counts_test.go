package sdk

import "testing"

func TestParseApplyCounts_GivenTerraformOutput_ShouldReadTheTally(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    ApplyCounts
		wantHit bool
	}{
		{
			name:    "apply tally",
			line:    "Apply complete! Resources: 1 added, 2 changed, 3 destroyed.",
			want:    ApplyCounts{Added: 1, Changed: 2, Destroyed: 3},
			wantHit: true,
		},
		{
			name:    "converged apply reports zeros",
			line:    "Apply complete! Resources: 0 added, 0 changed, 0 destroyed.",
			want:    ApplyCounts{},
			wantHit: true,
		},
		{
			name:    "destroy tally carries only one field",
			line:    "Destroy complete! Resources: 4 destroyed.",
			want:    ApplyCounts{Destroyed: 4},
			wantHit: true,
		},
		{
			name:    "styled output around the tally",
			line:    "\x1b[1m\x1b[32mApply complete! Resources: 7 added, 0 changed, 0 destroyed.\x1b[0m",
			want:    ApplyCounts{Added: 7},
			wantHit: true,
		},
		{
			// A count that does not fit an int is not a count we can report; the
			// field stays unknown rather than being guessed at.
			name:    "unreadable count leaves its field alone",
			line:    "Apply complete! Resources: 99999999999999999999 added, 2 changed, 0 destroyed.",
			want:    ApplyCounts{Changed: 2},
			wantHit: true,
		},
		{name: "plan summary is not an apply tally", line: "Plan: 3 to add, 0 to change, 0 to destroy."},
		{name: "progress line", line: "terraform_data.e: Creating..."},
		{name: "tally header without numbers", line: "Apply complete! Resources:"},
		{name: "empty line", line: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseApplyCounts(tt.line)
			if ok != tt.wantHit {
				t.Fatalf("ParseApplyCounts(%q) ok = %v, want %v", tt.line, ok, tt.wantHit)
			}
			if got != tt.want {
				t.Errorf("ParseApplyCounts(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestApplyCounts_String_WhenCalled_ShouldMatchTerraformWording(t *testing.T) {
	got := ApplyCounts{Added: 1, Changed: 0, Destroyed: 2}.String()
	want := "1 added, 0 changed, 2 destroyed"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParseApplyProgress_GivenTerraformOutput_ShouldReadCompletedActions(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    Action
		wantHit bool
	}{
		{
			name:    "creation notice",
			line:    "aws_instance.web: Creation complete after 3s [id=i-123]",
			want:    ActionCreate,
			wantHit: true,
		},
		{
			name:    "modification notice",
			line:    "aws_instance.web: Modifications complete after 1s [id=i-123]",
			want:    ActionUpdate,
			wantHit: true,
		},
		{
			name:    "destruction notice",
			line:    "aws_instance.old: Destruction complete after 8s",
			want:    ActionDelete,
			wantHit: true,
		},
		{name: "in-flight notice is not a completion", line: "aws_instance.web: Creating..."},
		{name: "tally line is not a per-resource notice", line: "Apply complete! Resources: 1 added, 0 changed, 0 destroyed."},
		{name: "empty line", line: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseApplyProgress(tt.line)
			if ok != tt.wantHit {
				t.Fatalf("ParseApplyProgress(%q) ok = %v, want %v", tt.line, ok, tt.wantHit)
			}
			if ok && got != tt.want {
				t.Errorf("ParseApplyProgress(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestApplyCounts_Add_WhenGivenActions_ShouldTallyPerAction(t *testing.T) {
	var counts ApplyCounts
	for _, a := range []Action{ActionCreate, ActionCreate, ActionUpdate, ActionDelete, ActionNoOp, ActionRead} {
		counts.Add(a)
	}
	want := ApplyCounts{Added: 2, Changed: 1, Destroyed: 1}
	if counts != want {
		t.Errorf("tally = %+v, want %+v", counts, want)
	}
}

func TestApplyCounts_Empty_WhenNothingTallied_ShouldReportTrue(t *testing.T) {
	if !(ApplyCounts{}).Empty() {
		t.Error("Empty() = false for a zero tally, want true")
	}
	if (ApplyCounts{Changed: 1}).Empty() {
		t.Error("Empty() = true for a tally with a change, want false")
	}
}

func TestApplyTally_GivenStreamedOutput_ShouldTallyBothWays(t *testing.T) {
	tally := &ApplyTally{}

	// Written in fragments that split lines mid-write, the way an exec pipe does.
	for _, chunk := range []string{
		"terraform_data.a: Creating...\nterraform_data.a: Creation compl",
		"ete after 0s\nterraform_data.b: Destruction complete after 1s\n",
		"\nApply complete! Resources: 1 added, 0 changed, 1 destroyed.\n",
	} {
		if n, err := tally.Write([]byte(chunk)); err != nil || n != len(chunk) {
			t.Fatalf("Write() = (%d, %v), want (%d, nil)", n, err, len(chunk))
		}
	}

	wantApplied := ApplyCounts{Added: 1, Destroyed: 1}
	if got := tally.Applied(); got != wantApplied {
		t.Errorf("Applied() = %+v, want %+v", got, wantApplied)
	}
	reported := tally.Reported()
	if reported == nil {
		t.Fatal("Reported() = nil, want terraform's tally")
	}
	if *reported != (ApplyCounts{Added: 1, Destroyed: 1}) {
		t.Errorf("Reported() = %+v, want %+v", *reported, ApplyCounts{Added: 1, Destroyed: 1})
	}
}

func TestApplyTally_WhenNoTallyLine_ShouldReportNil(t *testing.T) {
	tally := &ApplyTally{}
	if _, err := tally.Write([]byte("terraform_data.a: Creation complete after 0s\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if tally.Reported() != nil {
		t.Error("Reported() != nil for a run that printed no tally, want nil")
	}
	if got := tally.Applied(); got != (ApplyCounts{Added: 1}) {
		t.Errorf("Applied() = %+v, want 1 added", got)
	}
}
