package main

import (
	"errors"
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

	code, stderr := exitStatus(rootCmd.Execute())
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(code)
}

// exitError ends a verb run with a non-zero exit code. The run already wrote
// its stderr, so the error has nothing more to print.
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// exitStatus maps the root command's error to the process exit code and the
// text written to stderr before exiting.
func exitStatus(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	var exitErr *exitError
	if errors.As(err, &exitErr) {
		return exitErr.code, ""
	}
	stderr := "Error: " + err.Error() + "\n"
	var runErr *macro.RunError
	if errors.As(err, &runErr) {
		return runErr.Code, stderr + "macro: " + runErr.Error() + "\n"
	}
	return 1, stderr
}
