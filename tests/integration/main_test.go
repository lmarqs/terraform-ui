//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("terraform"); err != nil {
		fmt.Println("skipping integration tests: terraform not found")
		os.Exit(0)
	}

	projectRoot := findProjectRoot()
	binaryPath = filepath.Join(projectRoot, "dist", "tfui-integration-test")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-ldflags", "-X main.version=0.0.0-test", "-o", binaryPath, "./cmd/tfui")
	buildCmd.Dir = projectRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Remove(binaryPath)
	os.Exit(code)
}

func findProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func fixtureDir(name string) string {
	return filepath.Join(findProjectRoot(), "tests", "fixtures", name)
}

func runTfui(args ...string) (string, string, error) {
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr []byte
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return "", "", err
	}
	stdout, _ = readAll(stdoutPipe)
	stderr, _ = readAll(stderrPipe)
	err := cmd.Wait()
	return string(stdout), string(stderr), err
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return result, nil
}

func initFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	dir := fixtureDir(fixtureName)

	cmd := exec.Command("terraform", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("terraform init failed for fixture %q: %v\n%s", fixtureName, err, out)
	}

	return dir
}
