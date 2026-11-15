package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

const sessionPlanJSON = `{"format_version":"1.2","resource_changes":[{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["update"],"before":{"memory_size":128},"after":{"memory_size":256}}}]}` + "\n"

func TestRunPlugin_GivenSeededPlan_ShouldAnalyzeWithoutTerraformInInitialContext(t *testing.T) {
	for _, chdir := range []string{"", "member"} {
		for _, id := range []string{"plan", "risk"} {
			t.Run(id+"/"+chdir, func(t *testing.T) {
				dir := t.TempDir()
				workDir := filepath.Join(dir, chdir)
				if err := os.MkdirAll(workDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte("# seeded analysis\n"), 0600); err != nil {
					t.Fatal(err)
				}
				source := filepath.Join(dir, "plan.json")
				if err := os.WriteFile(source, []byte(sessionPlanJSON), 0600); err != nil {
					t.Fatal(err)
				}
				var stdout, stderr bytes.Buffer
				session := &Session{
					cfg:     config.Config{Dir: dir, Chdir: chdir, PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
					sources: PreseedSources{Plan: source}, silentStderr: true,
					effects: Effects{Stdout: &stdout, Stderr: &stderr},
				}
				var plan *tfuiplan.Plugin
				var risk *tfuirisk.Plugin
				var acquisitionErr error
				var evidence *sdk.PolicyEvidence
				err := session.RunPlugin(context.Background(), id, func(p sdk.Plugin) tea.Cmd {
					if id == "plan" {
						plan = p.(*tfuiplan.Plugin)
						return plan.Activate(tfuiplan.Input{})
					}
					risk = p.(*tfuirisk.Plugin)
					if got := risk.GetCtx().WorkingDir; got != workDir {
						t.Errorf("initial working directory = %q, want %q", got, workDir)
					}
					summary, err := risk.GetCtx().Service.Plan(context.Background(), sdk.PlanOptions{})
					acquisitionErr = err
					if err == nil {
						evidence = summary.PolicyEvidence
					}
					risk.Analyze(summary)
					return nil
				})
				var exitErr *exitError
				if err != nil && (id != "plan" || !errors.As(err, &exitErr) || exitErr.code != 2) {
					t.Fatalf("seeded %s RunPlugin error = %v; stderr = %s", id, err, stderr.String())
				}
				if acquisitionErr != nil {
					t.Fatalf("seeded Risk acquisition invoked nonexistent Terraform: %v", acquisitionErr)
				}
				if id == "plan" {
					if !strings.Contains(stdout.String(), "aws_lambda_function.worker") {
						t.Fatalf("seeded Plan output = %q, want reviewed resource", stdout.String())
					}
					summary, err := plan.GetCtx().Service.Plan(context.Background(), sdk.PlanOptions{})
					if err != nil {
						t.Fatal(err)
					}
					evidence = summary.PolicyEvidence
				} else if !strings.Contains(risk.View(100, 30), "aws_lambda_function.worker") {
					t.Fatalf("seeded Risk view = %q, want reviewed resource", risk.View(100, 30))
				}
				if evidence == nil || evidence.PlanID != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(sessionPlanJSON))) {
					t.Fatalf("seeded analysis evidence = %#v, want original reviewed JSON identity", evidence)
				}
				if len(evidence.Resources) != 1 || evidence.Resources[0].Input == nil || evidence.Resources[0].Input.Resource.Address != "aws_lambda_function.worker" {
					t.Fatalf("seeded policy resource evidence = %#v, want reviewed Lambda input", evidence.Resources)
				}
				if _, err := os.Stat(filepath.Join(workDir, "tfplan.out")); !os.IsNotExist(err) {
					t.Fatalf("seeded analysis created default plan artifact: %v", err)
				}
			})
		}
	}
}

func TestRunPlugin_GivenPlanJSON_ShouldPassThroughExactTerraformShowBytes(t *testing.T) {
	dir := t.TempDir()
	source := " \n" + sessionPlanJSON + "\t\n"
	if err := os.WriteFile(filepath.Join(dir, "show.json"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
case "$1" in
version) printf '%s\n' '{"terraform_version":"1.14.0","platform":"linux_amd64","provider_selections":{}}' ;;
plan) exit 2 ;;
show) cat show.json ;;
*) exit 1 ;;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	session := &Session{cfg: config.Config{Dir: dir, Terraform: config.TerraformConfig{Bin: binary}}, silentStderr: true, effects: Effects{Stdout: &stdout, Stderr: &stderr}}
	err := session.RunPlugin(context.Background(), "plan", func(p sdk.Plugin) tea.Cmd { return p.(*tfuiplan.Plugin).Activate(tfuiplan.Input{JSON: true}) })
	var exitErr *exitError
	if err != nil && (!errors.As(err, &exitErr) || exitErr.code != 2) {
		t.Fatalf("RunPlugin error = %v; stderr = %s", err, stderr.String())
	}
	if got := stdout.String(); got != source {
		t.Fatalf("Plan JSON output = %q, want exact Terraform bytes %q", got, source)
	}
}
