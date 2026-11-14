package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	tfuichdir "github.com/lmarqs/terraform-ui/plugins/chdir"
)

func TestBuildRegistry_WhenNoMembersDeclared_ShouldAutoSelectImplicitDotChdir(t *testing.T) {
	tests := []struct {
		name    string
		rootCfg *config.RootConfig
	}{
		{name: "zero-config without tfui.hcl", rootCfg: nil},
		{name: "tfui.hcl without member blocks", rootCfg: &config.RootConfig{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := buildRegistry(&sdktest.MockService{}, config.Config{Dir: t.TempDir()}, tt.rootCfg)
			p, ok := registry.ByID("chdir")
			if !ok {
				t.Fatal("chdir plugin not registered")
			}
			chdir, ok := p.(*tfuichdir.Plugin)
			if !ok {
				t.Fatalf("chdir plugin is %T, want *chdir.Plugin", p)
			}

			cmd := chdir.Activate()
			if cmd == nil {
				t.Fatal("chdir Activate() returned nil cmd, want ContextSwitchRequestMsg for the implicit \".\" member")
			}
			msg, ok := cmd().(sdk.ContextSwitchRequestMsg)
			if !ok {
				t.Fatalf("cmd() = %T, want sdk.ContextSwitchRequestMsg", cmd())
			}
			if msg.Chdir != sdk.Chdir(".") {
				t.Errorf("Chdir = %q, want %q", msg.Chdir, ".")
			}
			if msg.Workspace != sdk.WorkspaceDefault {
				t.Errorf("Workspace = %q, want %q", msg.Workspace, sdk.WorkspaceDefault)
			}
		})
	}
}
