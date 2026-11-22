package sdk

import (
	"errors"
	"testing"
)

func TestCommand_String_WhenVerbOnly(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldJoinBinaryAndVerb",
			Command{Binary: "terraform", Verb: "plan"},
			"terraform plan",
		},
		{
			"ShouldWorkWithTofu",
			Command{Binary: "tofu", Verb: "apply"},
			"tofu apply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.String()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_String_WhenFlagsProvided(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldPlaceFlagsAfterVerb",
			Command{Binary: "terraform", Verb: "plan", Flags: []string{"-out=plan.bin"}},
			"terraform plan -out=plan.bin",
		},
		{
			"ShouldPreserveMultipleFlagOrder",
			Command{Binary: "terraform", Verb: "plan", Flags: []string{"-target=aws_s3_bucket.a", "-target=aws_s3_bucket.b"}},
			"terraform plan -target=aws_s3_bucket.a -target=aws_s3_bucket.b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.String()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_String_WhenArgsProvided(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldPlaceArgsAfterVerb",
			Command{Binary: "terraform", Verb: "state rm", Args: []string{"aws_instance.web"}},
			"terraform state rm aws_instance.web",
		},
		{
			"ShouldPreserveMultipleArgOrder",
			Command{Binary: "terraform", Verb: "state rm", Args: []string{"aws_instance.a", "aws_instance.b"}},
			"terraform state rm aws_instance.a aws_instance.b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.String()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_String_WhenFlagsAndArgs(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldPlaceFlagsBeforeArgs",
			Command{
				Binary: "terraform",
				Verb:   "plan",
				Flags:  []string{"-target=aws_s3_bucket.main"},
				Args:   []string{"-destroy"},
			},
			"terraform plan -target=aws_s3_bucket.main -destroy",
		},
		{
			"ShouldHandleComplexInvocation",
			Command{
				Binary: "tofu",
				Verb:   "import",
				Flags:  []string{"-var-file=prod.tfvars"},
				Args:   []string{"aws_instance.web", "i-1234567890"},
			},
			"tofu import -var-file=prod.tfvars aws_instance.web i-1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.String()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_String_WhenDirSet(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldPlaceChdirRightAfterBinary",
			Command{Binary: "terraform", Verb: "plan", Dir: "/some/path"},
			"terraform -chdir=/some/path plan",
		},
		{
			"ShouldPlaceChdirBeforeFlagsAndArgs",
			Command{Binary: "tofu", Verb: "state rm", Flags: []string{"-lock=false"}, Args: []string{"aws_instance.web"}, Dir: "/work/modules/vpc"},
			"tofu -chdir=/work/modules/vpc state rm -lock=false aws_instance.web",
		},
		{
			"ShouldQuoteTheWholeChdirTokenWhenTheDirNeedsIt",
			Command{Binary: "terraform", Verb: "plan", Dir: "/my work/vpc"},
			"terraform '-chdir=/my work/vpc' plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.String(); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_String_WhenTokensNeedShellQuoting(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldQuoteAnAddressWithAnIndexKey",
			Command{Binary: "terraform", Verb: "state rm", Args: []string{`aws_instance.web["a"]`}},
			`terraform state rm 'aws_instance.web["a"]'`,
		},
		{
			"ShouldQuoteAWholeFlagWhenItsValueNeedsIt",
			Command{Binary: "terraform", Verb: "plan", Flags: []string{`-target=aws_instance.web["a"]`}},
			`terraform plan '-target=aws_instance.web["a"]'`,
		},
		{
			"ShouldQuoteNumericIndexBrackets",
			Command{Binary: "terraform", Verb: "taint", Args: []string{"aws_instance.web[0]"}},
			"terraform taint 'aws_instance.web[0]'",
		},
		{
			"ShouldEscapeEmbeddedSingleQuotes",
			Command{Binary: "terraform", Verb: "import", Args: []string{"aws_instance.web", "it's"}},
			`terraform import aws_instance.web 'it'\''s'`,
		},
		{
			"ShouldQuoteAnEmptyArgument",
			Command{Binary: "terraform", Verb: "import", Args: []string{"aws_instance.web", ""}},
			"terraform import aws_instance.web ''",
		},
		{
			"ShouldQuoteShellMetacharacters",
			Command{Binary: "terraform", Verb: "apply", Flags: []string{"-var=a b", "-var=$HOME", "-var=a;b", "-var=~"}},
			"terraform apply '-var=a b' '-var=$HOME' '-var=a;b' '-var=~'",
		},
		{
			"ShouldQuoteABinaryPathWithASpace",
			Command{Binary: "/opt/my tools/terraform", Verb: "plan"},
			"'/opt/my tools/terraform' plan",
		},
		{
			"ShouldLeaveSafeTokensBare",
			Command{Binary: "terraform", Verb: "plan", Flags: []string{"-out=plan.bin", "-var-file=prod.tfvars", "-var=region=us-east-1"}, Args: []string{"module.vpc.aws_subnet.a", "i-1234567890", "a@b:c,d+e%f/g"}},
			"terraform plan -out=plan.bin -var-file=prod.tfvars -var=region=us-east-1 module.vpc.aws_subnet.a i-1234567890 a@b:c,d+e%f/g",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.String(); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCommand_WithDir_ShouldReturnCopyWithDir(t *testing.T) {
	original := Command{
		Binary: "terraform",
		Verb:   "plan",
		Flags:  []string{"-out=plan.bin"},
	}

	copy := original.WithDir("/work/modules/vpc")

	if copy.Dir != "/work/modules/vpc" {
		t.Fatalf("expected copy Dir to be set, got %q", copy.Dir)
	}
	if copy.Binary != original.Binary {
		t.Fatalf("expected Binary preserved, got %q", copy.Binary)
	}
	if copy.Verb != original.Verb {
		t.Fatalf("expected Verb preserved, got %q", copy.Verb)
	}
}

func TestCommand_WithDir_ShouldNotMutateOriginal(t *testing.T) {
	original := Command{
		Binary: "terraform",
		Verb:   "apply",
		Dir:    "/original",
	}

	_ = original.WithDir("/new/path")

	if original.Dir != "/original" {
		t.Fatalf("expected original Dir unchanged, got %q", original.Dir)
	}
}

func TestCommandErr_Error_ShouldReturnCommandString(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{
			"ShouldSerializeSimpleCommand",
			Command{Binary: "terraform", Verb: "plan"},
			"terraform plan",
		},
		{
			"ShouldSerializeFullCommand",
			Command{
				Binary: "tofu",
				Verb:   "state rm",
				Flags:  []string{"-lock=false"},
				Args:   []string{"aws_instance.web"},
			},
			"tofu state rm -lock=false aws_instance.web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &CommandErr{Cmd: tt.cmd}
			got := err.Error()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestIsCommandErr_WhenNil(t *testing.T) {
	cmd, ok := IsCommandErr(nil)
	if ok {
		t.Fatal("expected false for nil error")
	}
	if cmd.Binary != "" || cmd.Verb != "" {
		t.Fatalf("expected zero Command, got %+v", cmd)
	}
}

func TestIsCommandErr_WhenNonCommandErr(t *testing.T) {
	err := errors.New("something went wrong")
	cmd, ok := IsCommandErr(err)
	if ok {
		t.Fatal("expected false for non-CommandErr")
	}
	if cmd.Binary != "" || cmd.Verb != "" {
		t.Fatalf("expected zero Command, got %+v", cmd)
	}
}

func TestIsCommandErr_WhenValidCommandErr(t *testing.T) {
	expected := Command{
		Binary: "terraform",
		Verb:   "plan",
		Flags:  []string{"-target=aws_vpc.main"},
		Args:   []string{},
		Dir:    "/work",
	}
	err := &CommandErr{Cmd: expected}

	cmd, ok := IsCommandErr(err)
	if !ok {
		t.Fatal("expected true for valid CommandErr")
	}
	if cmd.Binary != expected.Binary {
		t.Fatalf("expected Binary %q, got %q", expected.Binary, cmd.Binary)
	}
	if cmd.Verb != expected.Verb {
		t.Fatalf("expected Verb %q, got %q", expected.Verb, cmd.Verb)
	}
	if len(cmd.Flags) != 1 || cmd.Flags[0] != "-target=aws_vpc.main" {
		t.Fatalf("expected Flags [-target=aws_vpc.main], got %v", cmd.Flags)
	}
	if cmd.Dir != "/work" {
		t.Fatalf("expected Dir %q, got %q", "/work", cmd.Dir)
	}
}
