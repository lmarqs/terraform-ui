package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/macro"
)

var version string

func init() {
	info, ok := debug.ReadBuildInfo()
	version = resolveVersion(version, info, ok)
}

// resolveVersion returns the version without a leading "v": the ldflag one a
// release build sets, else the module version `go install` records, else a
// snapshot marker. The version plugin prints its own "v" prefix.
func resolveVersion(ldflagVersion string, info *debug.BuildInfo, ok bool) string {
	if ldflagVersion != "" {
		return ldflagVersion
	}
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "0.0.0-SNAPSHOT"
}

func main() {
	rootCmd, session, extraArgsDst := buildRoot()

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

	os.Args, *extraArgsDst = splitPassthrough(os.Args)
	os.Args = normalizeArgs(os.Args)

	if err := rootCmd.Execute(); err != nil {
		if runErr, ok := err.(*macro.RunError); ok {
			fmt.Fprintf(os.Stderr, "macro: %s\n", runErr.Error())
			os.Exit(runErr.Code)
		}
		os.Exit(1)
	}
}
