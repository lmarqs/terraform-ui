package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/lmarqs/terraform-ui/internal/macro"
)

var version string

func init() {
	if version == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
				return
			}
		}
		version = "0.0.0-SNAPSHOT"
	}
}

func main() {
	rootCmd, session := buildRoot()

	rootCmd.AddCommand(
		buildPlanCommand(session),
		buildApplyCommand(session),
		buildStateCommand(session),
		buildTaintCommand(session),
		buildUntaintCommand(session),
		buildImportCommand(session),
		buildInitCommand(session),
		buildValidateCommand(session),
		buildOutputCommand(session),
		buildVersionCommand(session),
		buildWorkspaceCommand(&session.cfg),
		buildForceUnlockCommand(&session.cfg),
		buildScaffoldCommand(session),
	)

	os.Args, session.cfg.ExtraArgs = splitPassthrough(os.Args)
	os.Args = normalizeArgs(os.Args)

	if err := rootCmd.Execute(); err != nil {
		if runErr, ok := err.(*macro.RunError); ok {
			fmt.Fprintf(os.Stderr, "macro: %s\n", runErr.Error())
			os.Exit(runErr.Code)
		}
		os.Exit(1)
	}
}
