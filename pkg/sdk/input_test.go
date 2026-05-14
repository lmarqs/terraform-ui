package sdk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputConfirm_WhenCreated_ShouldHaveCorrectFields(t *testing.T) {
	req := InputConfirm("Apply?", func() tea.Cmd {
		return nil
	})

	if req.Mode != InputRequestBool {
		t.Errorf("Mode = %d, want %d (InputRequestBool)", req.Mode, InputRequestBool)
	}
	if req.Prompt != "Apply? (y/n)" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Apply? (y/n)")
	}
}

func TestInputConfirm_WhenAnswerIsY_ShouldCallOnYes(t *testing.T) {
	called := false
	req := InputConfirm("Delete?", func() tea.Cmd {
		called = true
		return nil
	})

	req.Callback("y")
	if !called {
		t.Error("onYes was not called for answer 'y'")
	}
}

func TestInputConfirm_WhenAnswerIsYesFull_ShouldCallOnYes(t *testing.T) {
	called := false
	req := InputConfirm("Delete?", func() tea.Cmd {
		called = true
		return nil
	})

	req.Callback("yes")
	if !called {
		t.Error("onYes was not called for answer 'yes'")
	}
}

func TestInputConfirm_WhenAnswerIsNo_ShouldReturnNil(t *testing.T) {
	called := false
	req := InputConfirm("Delete?", func() tea.Cmd {
		called = true
		return nil
	})

	cmd := req.Callback("n")
	if called {
		t.Error("onYes was called for answer 'n', want not called")
	}
	if cmd != nil {
		t.Error("Callback('n') returned non-nil cmd, want nil")
	}
}

func TestInputConfirm_WhenAnswerIsOther_ShouldReturnNil(t *testing.T) {
	called := false
	req := InputConfirm("Delete?", func() tea.Cmd {
		called = true
		return nil
	})

	cmd := req.Callback("maybe")
	if called {
		t.Error("onYes was called for answer 'maybe', want not called")
	}
	if cmd != nil {
		t.Error("Callback('maybe') returned non-nil cmd, want nil")
	}
}

func TestInputText_WhenCreated_ShouldHaveCorrectFields(t *testing.T) {
	req := InputText("Address:", "aws_instance.web", func(s string) tea.Cmd {
		return nil
	})

	if req.Mode != InputRequestText {
		t.Errorf("Mode = %d, want %d (InputRequestText)", req.Mode, InputRequestText)
	}
	if req.Prompt != "Address:" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Address:")
	}
	if req.Default != "aws_instance.web" {
		t.Errorf("Default = %q, want %q", req.Default, "aws_instance.web")
	}
}

func TestInputText_WhenSubmitted_ShouldPassAnswerToCallback(t *testing.T) {
	var received string
	req := InputText("Address:", "", func(s string) tea.Cmd {
		received = s
		return nil
	})

	req.Callback("aws_s3_bucket.data")
	if received != "aws_s3_bucket.data" {
		t.Errorf("callback received %q, want %q", received, "aws_s3_bucket.data")
	}
}

func TestInputSelect_WhenCreated_ShouldHaveCorrectFields(t *testing.T) {
	options := []string{"dev", "staging", "prod"}
	req := InputSelect("Workspace:", options, func(s string) tea.Cmd {
		return nil
	})

	if req.Mode != InputRequestSelect {
		t.Errorf("Mode = %d, want %d (InputRequestSelect)", req.Mode, InputRequestSelect)
	}
	if req.Prompt != "Workspace:" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Workspace:")
	}
	if len(req.Options) != 3 {
		t.Fatalf("Options length = %d, want 3", len(req.Options))
	}
	if req.Options[2] != "prod" {
		t.Errorf("Options[2] = %q, want %q", req.Options[2], "prod")
	}
}

func TestInputSelect_WhenSelected_ShouldPassChoiceToCallback(t *testing.T) {
	var received string
	req := InputSelect("Workspace:", []string{"dev", "prod"}, func(s string) tea.Cmd {
		received = s
		return nil
	})

	req.Callback("prod")
	if received != "prod" {
		t.Errorf("callback received %q, want %q", received, "prod")
	}
}
