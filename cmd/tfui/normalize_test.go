package main

import (
	"slices"
	"testing"
)

func TestNormalizeArgs_WhenEmpty_ShouldReturnEmpty(t *testing.T) {
	got := normalizeArgs(nil)
	if len(got) != 0 {
		t.Errorf("normalizeArgs(nil) = %v, want empty", got)
	}

	got = normalizeArgs([]string{})
	if len(got) != 0 {
		t.Errorf("normalizeArgs([]) = %v, want empty", got)
	}
}

func TestNormalizeArgs_WhenBinaryOnly_ShouldPassThrough(t *testing.T) {
	args := []string{"tfui"}
	got := normalizeArgs(args)
	want := []string{"tfui"}
	if !slices.Equal(got, want) {
		t.Errorf("normalizeArgs(%v) = %v, want %v", args, got, want)
	}
}

func TestNormalizeArgs_WhenSubcommandOnly_ShouldPassThrough(t *testing.T) {
	args := []string{"tfui", "plan"}
	got := normalizeArgs(args)
	want := []string{"tfui", "plan"}
	if !slices.Equal(got, want) {
		t.Errorf("normalizeArgs(%v) = %v, want %v", args, got, want)
	}
}

func TestNormalizeArgs_WhenKnownFlagWithEquals_ShouldNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNormalizeTarget",
			[]string{"tfui", "plan", "-target=aws_instance.web"},
			[]string{"tfui", "plan", "--target=aws_instance.web"},
		},
		{
			"ShouldNormalizeVar",
			[]string{"tfui", "plan", "-var=region=us-east-1"},
			[]string{"tfui", "plan", "--var=region=us-east-1"},
		},
		{
			"ShouldNormalizeVarFile",
			[]string{"tfui", "plan", "-var-file=prod.tfvars"},
			[]string{"tfui", "plan", "--var-file=prod.tfvars"},
		},
		{
			"ShouldNormalizeReplace",
			[]string{"tfui", "plan", "-replace=aws_instance.web"},
			[]string{"tfui", "plan", "--replace=aws_instance.web"},
		},
		{
			"ShouldNormalizeParallelism",
			[]string{"tfui", "plan", "-parallelism=10"},
			[]string{"tfui", "plan", "--parallelism=10"},
		},
		{
			"ShouldNormalizeLock",
			[]string{"tfui", "plan", "-lock=false"},
			[]string{"tfui", "plan", "--lock=false"},
		},
		{
			"ShouldNormalizeLockTimeout",
			[]string{"tfui", "plan", "-lock-timeout=30s"},
			[]string{"tfui", "plan", "--lock-timeout=30s"},
		},
		{
			"ShouldNormalizeChdir",
			[]string{"tfui", "-chdir=/tmp/project"},
			[]string{"tfui", "--chdir=/tmp/project"},
		},
		{
			"ShouldNormalizeWorkspace",
			[]string{"tfui", "-workspace=staging"},
			[]string{"tfui", "--workspace=staging"},
		},
		{
			"ShouldNormalizeInput",
			[]string{"tfui", "apply", "-input=false"},
			[]string{"tfui", "apply", "--input=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenKnownFlagWithSpace_ShouldNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNormalizeTargetWithSpaceValue",
			[]string{"tfui", "plan", "-target", "aws_instance.web"},
			[]string{"tfui", "plan", "--target", "aws_instance.web"},
		},
		{
			"ShouldNormalizeVarWithSpaceValue",
			[]string{"tfui", "plan", "-var", "region=us-east-1"},
			[]string{"tfui", "plan", "--var", "region=us-east-1"},
		},
		{
			"ShouldNormalizeVarFileWithSpaceValue",
			[]string{"tfui", "plan", "-var-file", "prod.tfvars"},
			[]string{"tfui", "plan", "--var-file", "prod.tfvars"},
		},
		{
			"ShouldNormalizeReplaceWithSpaceValue",
			[]string{"tfui", "plan", "-replace", "aws_instance.web"},
			[]string{"tfui", "plan", "--replace", "aws_instance.web"},
		},
		{
			"ShouldNormalizeParallelismWithSpaceValue",
			[]string{"tfui", "plan", "-parallelism", "10"},
			[]string{"tfui", "plan", "--parallelism", "10"},
		},
		{
			"ShouldNormalizeLockWithSpaceValue",
			[]string{"tfui", "apply", "-lock", "false"},
			[]string{"tfui", "apply", "--lock", "false"},
		},
		{
			"ShouldNormalizeLockTimeoutWithSpaceValue",
			[]string{"tfui", "plan", "-lock-timeout", "30s"},
			[]string{"tfui", "plan", "--lock-timeout", "30s"},
		},
		{
			"ShouldNormalizeChdirWithSpaceValue",
			[]string{"tfui", "-chdir", "/tmp/project"},
			[]string{"tfui", "--chdir", "/tmp/project"},
		},
		{
			"ShouldNormalizeWorkspaceWithSpaceValue",
			[]string{"tfui", "-workspace", "staging"},
			[]string{"tfui", "--workspace", "staging"},
		},
		{
			"ShouldNormalizeInputWithSpaceValue",
			[]string{"tfui", "apply", "-input", "false"},
			[]string{"tfui", "apply", "--input", "false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenBooleanFlag_ShouldNotConsumeNextArg(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNotConsumeArgAfterDestroy",
			[]string{"tfui", "plan", "-destroy", "-target=aws_instance.web"},
			[]string{"tfui", "plan", "--destroy", "--target=aws_instance.web"},
		},
		{
			"ShouldNotConsumeArgAfterCompactWarnings",
			[]string{"tfui", "plan", "-compact-warnings", "-var=x=y"},
			[]string{"tfui", "plan", "--compact-warnings", "--var=x=y"},
		},
		{
			"ShouldNotConsumeArgAfterRefreshOnly",
			[]string{"tfui", "plan", "-refresh-only", "-target=aws_instance.web"},
			[]string{"tfui", "plan", "--refresh-only", "--target=aws_instance.web"},
		},
		{
			"ShouldNotConsumePositionalArgAfterDestroy",
			[]string{"tfui", "plan", "-destroy", "somedir"},
			[]string{"tfui", "plan", "--destroy", "somedir"},
		},
		{
			"ShouldHandleBooleanFlagAtEnd",
			[]string{"tfui", "plan", "-target=aws_instance.web", "-destroy"},
			[]string{"tfui", "plan", "--target=aws_instance.web", "--destroy"},
		},
		{
			"ShouldNotConsumeArgAfterBackend",
			[]string{"tfui", "init", "-backend", "-upgrade"},
			[]string{"tfui", "init", "--backend", "--upgrade"},
		},
		{
			"ShouldPreserveBackendWithEqualsFalse",
			[]string{"tfui", "init", "-backend=false"},
			[]string{"tfui", "init", "--backend=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenAlreadyDoubleDash_ShouldPassThrough(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNotModifyDoubleDashTarget",
			[]string{"tfui", "plan", "--target=aws_instance.web"},
			[]string{"tfui", "plan", "--target=aws_instance.web"},
		},
		{
			"ShouldNotModifyDoubleDashVar",
			[]string{"tfui", "plan", "--var", "region=us-east-1"},
			[]string{"tfui", "plan", "--var", "region=us-east-1"},
		},
		{
			"ShouldNotModifyDoubleDashDestroy",
			[]string{"tfui", "plan", "--destroy"},
			[]string{"tfui", "plan", "--destroy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenPassthroughSeparator_ShouldStopNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNotNormalizeAfterSeparator",
			[]string{"tfui", "plan", "-target=aws_instance.web", "--", "-var=x"},
			[]string{"tfui", "plan", "--target=aws_instance.web", "--", "-var=x"},
		},
		{
			"ShouldLeaveEverythingAfterSeparatorUnchanged",
			[]string{"tfui", "--", "-target=foo", "-destroy", "-var-file=bar"},
			[]string{"tfui", "--", "-target=foo", "-destroy", "-var-file=bar"},
		},
		{
			"ShouldHandleSeparatorWithNoFollowingArgs",
			[]string{"tfui", "plan", "-destroy", "--"},
			[]string{"tfui", "plan", "--destroy", "--"},
		},
		{
			"ShouldHandleUnknownFlagsAfterSeparator",
			[]string{"tfui", "plan", "--", "-unknown", "-target=x"},
			[]string{"tfui", "plan", "--", "-unknown", "-target=x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenStdinIndicator_ShouldNotTouch(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldPreserveStdinDashAlone",
			[]string{"tfui", "--plan", "-"},
			[]string{"tfui", "--plan", "-"},
		},
		{
			"ShouldPreserveStdinDashWithOtherFlags",
			[]string{"tfui", "--plan", "-", "-target=aws_instance.web"},
			[]string{"tfui", "--plan", "-", "--target=aws_instance.web"},
		},
		{
			"ShouldPreserveStdinDashAsValue",
			[]string{"tfui", "--state", "-", "-destroy"},
			[]string{"tfui", "--state", "-", "--destroy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenUnknownFlag_ShouldNormalizeMultiChar(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNormalizeUnknownMultiCharFlag",
			[]string{"tfui", "plan", "-unknown-flag"},
			[]string{"tfui", "plan", "--unknown-flag"},
		},
		{
			"ShouldNormalizeUnknownMultiCharFlagWithValue",
			[]string{"tfui", "plan", "-unknown=value"},
			[]string{"tfui", "plan", "--unknown=value"},
		},
		{
			"ShouldNotTouchShortFlag",
			[]string{"tfui", "-v"},
			[]string{"tfui", "-v"},
		},
		{
			"ShouldNormalizeMultipleUnknownMultiCharFlags",
			[]string{"tfui", "plan", "-foo", "-bar=baz"},
			[]string{"tfui", "plan", "--foo", "--bar=baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenRepeatedFlags_ShouldNormalizeAll(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldNormalizeMultipleTargets",
			[]string{"tfui", "plan", "-target=aws_instance.web", "-target=aws_s3_bucket.data"},
			[]string{"tfui", "plan", "--target=aws_instance.web", "--target=aws_s3_bucket.data"},
		},
		{
			"ShouldNormalizeMultipleVars",
			[]string{"tfui", "plan", "-var=a=1", "-var=b=2", "-var=c=3"},
			[]string{"tfui", "plan", "--var=a=1", "--var=b=2", "--var=c=3"},
		},
		{
			"ShouldNormalizeMultipleTargetsWithSpace",
			[]string{"tfui", "plan", "-target", "aws_instance.web", "-target", "aws_s3_bucket.data"},
			[]string{"tfui", "plan", "--target", "aws_instance.web", "--target", "aws_s3_bucket.data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenMixedFlags_ShouldNormalizeAll(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldMixKnownAndUnknown",
			[]string{"tfui", "plan", "-target=aws_instance.web", "-unknown", "--debug"},
			[]string{"tfui", "plan", "--target=aws_instance.web", "--unknown", "--debug"},
		},
		{
			"ShouldMixSingleAndDoubleDash",
			[]string{"tfui", "plan", "-target=foo", "--var=bar", "-destroy"},
			[]string{"tfui", "plan", "--target=foo", "--var=bar", "--destroy"},
		},
		{
			"ShouldHandleMixOfEqualsAndSpace",
			[]string{"tfui", "plan", "-target=foo", "-var", "x=1", "-destroy"},
			[]string{"tfui", "plan", "--target=foo", "--var", "x=1", "--destroy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenComplexRealWorldCommand_ShouldNormalizeCorrectly(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldHandleFullPlanCommand",
			[]string{"tfui", "plan", "-target=aws_instance.web", "-var-file=prod.tfvars", "-var=region=us-east-1", "-destroy"},
			[]string{"tfui", "plan", "--target=aws_instance.web", "--var-file=prod.tfvars", "--var=region=us-east-1", "--destroy"},
		},
		{
			"ShouldHandleApplyWithMultipleTargets",
			[]string{"tfui", "apply", "-target=aws_instance.web", "-target=aws_s3_bucket.data", "-parallelism=5", "-lock=true"},
			[]string{"tfui", "apply", "--target=aws_instance.web", "--target=aws_s3_bucket.data", "--parallelism=5", "--lock=true"},
		},
		{
			"ShouldHandleSpaceSeparatedMix",
			[]string{"tfui", "plan", "-target", "aws_instance.web", "-var-file", "prod.tfvars", "-destroy", "-lock-timeout", "10s"},
			[]string{"tfui", "plan", "--target", "aws_instance.web", "--var-file", "prod.tfvars", "--destroy", "--lock-timeout", "10s"},
		},
		{
			"ShouldHandleGlobalAndSubcommandFlags",
			[]string{"tfui", "--debug", "-chdir=/tmp/project", "plan", "-target=aws_instance.web", "-refresh-only"},
			[]string{"tfui", "--debug", "--chdir=/tmp/project", "plan", "--target=aws_instance.web", "--refresh-only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenValueContainsSpecialCharacters_ShouldPreserveValue(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldPreserveEqualsInValue",
			[]string{"tfui", "plan", "-var=name=hello=world"},
			[]string{"tfui", "plan", "--var=name=hello=world"},
		},
		{
			"ShouldPreserveQuotedSpacesInValue",
			[]string{"tfui", "plan", "-var=name=hello world"},
			[]string{"tfui", "plan", "--var=name=hello world"},
		},
		{
			"ShouldPreservePathCharactersInValue",
			[]string{"tfui", "plan", "-var-file=/path/to/my file.tfvars"},
			[]string{"tfui", "plan", "--var-file=/path/to/my file.tfvars"},
		},
		{
			"ShouldPreserveJsonInValue",
			[]string{"tfui", "plan", `-var=tags={"env":"prod","team":"infra"}`},
			[]string{"tfui", "plan", `--var=tags={"env":"prod","team":"infra"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenValueLooksLikeFlag_ShouldPreserveAsValue(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldPreserveFlagLikeValueWithEquals",
			[]string{"tfui", "plan", "-var=-something"},
			[]string{"tfui", "plan", "--var=-something"},
		},
		{
			"ShouldPreserveFlagLikeValueWithSpace",
			[]string{"tfui", "plan", "-var", "-something"},
			[]string{"tfui", "plan", "--var", "-something"},
		},
		{
			"ShouldPreserveDoubleDashLikeValueWithEquals",
			[]string{"tfui", "plan", "-var=--something"},
			[]string{"tfui", "plan", "--var=--something"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenNoFlags_ShouldPassThrough(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldPassThroughPositionalArgs",
			[]string{"tfui", "plan", "somedir"},
			[]string{"tfui", "plan", "somedir"},
		},
		{
			"ShouldPassThroughMultiplePositionalArgs",
			[]string{"tfui", "state", "mv", "source", "destination"},
			[]string{"tfui", "state", "mv", "source", "destination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenBooleanFlagWithExplicitValue_ShouldPreserveValue(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"ShouldPreserveDestroyWithEqualsTrue",
			[]string{"tfui", "plan", "-destroy=true"},
			[]string{"tfui", "plan", "--destroy=true"},
		},
		{
			"ShouldPreserveRefreshOnlyWithEqualsFalse",
			[]string{"tfui", "plan", "-refresh-only=false"},
			[]string{"tfui", "plan", "--refresh-only=false"},
		},
		{
			"ShouldPreserveCompactWarningsWithEqualsTrue",
			[]string{"tfui", "plan", "-compact-warnings=true"},
			[]string{"tfui", "plan", "--compact-warnings=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArgs_WhenInputNotMutated_ShouldNotModifyOriginal(t *testing.T) {
	original := []string{"tfui", "plan", "-target=aws_instance.web", "-destroy"}
	inputCopy := make([]string, len(original))
	copy(inputCopy, original)

	normalizeArgs(inputCopy)

	if !slices.Equal(inputCopy, original) {
		t.Errorf("normalizeArgs mutated input slice: got %v, original was %v", inputCopy, original)
	}
}
