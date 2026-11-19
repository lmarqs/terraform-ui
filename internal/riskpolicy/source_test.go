package riskpolicy

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
)

func writePolicyModule(t *testing.T, path, source string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
}

func expectedSourceDigest(modules []Module) string {
	bytes := []byte("tfui-risk-source-v1\x00")
	bytes = binary.BigEndian.AppendUint64(bytes, uint64(len(modules)))
	for _, m := range modules {
		bytes = binary.BigEndian.AppendUint64(bytes, uint64(len(m.Name)))
		bytes = append(bytes, m.Name...)
		bytes = binary.BigEndian.AppendUint64(bytes, uint64(len(m.Source)))
		bytes = append(bytes, m.Source...)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(bytes))
}

func TestLoadSource_GivenBuiltinSelection_ShouldIdentifyAndFreezeExactModules(t *testing.T) {
	modules := []Module{{Name: "main.rego", Source: "package tfui.risk\nimport rego.v1\ndefault result := {\"level\": \"none\", \"reason\": \"Default classification\"}\n"}}
	want := append([]Module(nil), modules...)
	snapshot, err := LoadSource(config.RiskPolicySelection{}, modules)
	if err != nil {
		t.Fatal(err)
	}
	modules[0].Source = "overwritten"
	copyModules := snapshot.Modules()
	copyModules[0].Name = "overwritten"
	if snapshot.Kind() != "builtin" || snapshot.ID() != "builtin/v1" || snapshot.Digest() != expectedSourceDigest(want) {
		t.Fatalf("builtin provenance: kind=%q id=%q digest=%q", snapshot.Kind(), snapshot.ID(), snapshot.Digest())
	}
	if !reflect.DeepEqual(snapshot.Modules(), want) {
		t.Fatalf("snapshot was mutable: %+v", snapshot.Modules())
	}
}

func TestLoadSource_GivenDirectory_ShouldLoadOnlySortedImmediateProductionModules(t *testing.T) {
	dir := t.TempDir()
	writePolicyModule(t, filepath.Join(dir, "z.rego"), "package z\r\n")
	writePolicyModule(t, filepath.Join(dir, "a.rego"), "package a\n")
	writePolicyModule(t, filepath.Join(dir, "a_test.rego"), "invalid author test")
	writePolicyModule(t, filepath.Join(dir, "data.json"), "{}")
	writePolicyModule(t, filepath.Join(dir, "nested", "hidden.rego"), "package hidden")
	snapshot, err := LoadSource(config.RiskPolicySelection{Path: dir, Origin: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Module{{Name: "a.rego", Source: "package a\n"}, {Name: "z.rego", Source: "package z\r\n"}}
	if !reflect.DeepEqual(snapshot.Modules(), want) {
		t.Fatalf("selected modules=%+v want %+v", snapshot.Modules(), want)
	}
	digest := expectedSourceDigest(want)
	if snapshot.Kind() != "custom" || snapshot.Digest() != digest || snapshot.ID() != "custom/"+digest {
		t.Fatalf("custom provenance: kind=%q id=%q digest=%q", snapshot.Kind(), snapshot.ID(), snapshot.Digest())
	}
	copied := snapshot.Modules()
	copied[0].Source = "mutated"
	if !reflect.DeepEqual(snapshot.Modules(), want) {
		t.Fatal("snapshot exposed mutable module slice")
	}
}

func TestLoadSource_GivenMovedEditedOrRenamedModules_ShouldTrackSourceBytesAndNames(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a", "main.rego")
	writePolicyModule(t, file, "package x\n")
	first, err := LoadSource(config.RiskPolicySelection{Path: file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(dir, "b", "main.rego")
	writePolicyModule(t, moved, "package x\n")
	second, err := LoadSource(config.RiskPolicySelection{Path: moved}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() {
		t.Fatal("absolute source location changed source digest")
	}
	writePolicyModule(t, moved, "package x\r\n")
	edited, err := LoadSource(config.RiskPolicySelection{Path: moved}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Digest() == first.Digest() {
		t.Fatal("byte edit did not change digest")
	}
	if first.Modules()[0].Source != "package x\n" {
		t.Fatal("reloading edited file changed old snapshot")
	}
	renamed := filepath.Join(dir, "b", "renamed.rego")
	if err := os.Rename(moved, renamed); err != nil {
		t.Fatal(err)
	}
	third, err := LoadSource(config.RiskPolicySelection{Path: renamed}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if third.Digest() == edited.Digest() {
		t.Fatal("module rename did not change digest")
	}
}

func TestLoadSource_GivenInvalidSelection_ShouldFailWithoutBuiltinFallback(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string) string
		code  string
	}{
		{"missing", func(_ *testing.T, dir string) string { return filepath.Join(dir, "missing.rego") }, "source_unreadable"},
		{"non rego", func(t *testing.T, dir string) string {
			path := filepath.Join(dir, "policy.json")
			writePolicyModule(t, path, "{}")
			return path
		}, "source_invalid"},
		{"selected test", func(t *testing.T, dir string) string {
			path := filepath.Join(dir, "main_test.rego")
			writePolicyModule(t, path, "package x")
			return path
		}, "source_invalid"},
		{"empty directory", func(_ *testing.T, dir string) string { return dir }, "source_invalid"},
		{"only excluded files", func(t *testing.T, dir string) string {
			writePolicyModule(t, filepath.Join(dir, "main_test.rego"), "package x")
			writePolicyModule(t, filepath.Join(dir, "sub", "main.rego"), "package y")
			return dir
		}, "source_invalid"},
		{"dangling selected link", func(t *testing.T, dir string) string {
			path := filepath.Join(dir, "link.rego")
			if err := os.Symlink(filepath.Join(dir, "missing.rego"), path); err != nil {
				t.Fatal(err)
			}
			return path
		}, "source_unreadable"},
		{"dangling module link", func(t *testing.T, dir string) string {
			writePolicyModule(t, filepath.Join(dir, "main.rego"), "package x")
			if err := os.Symlink(filepath.Join(dir, "missing.rego"), filepath.Join(dir, "other.rego")); err != nil {
				t.Fatal(err)
			}
			return dir
		}, "source_unreadable"},
		{"link cycle", func(t *testing.T, dir string) string {
			first, second := filepath.Join(dir, "a.rego"), filepath.Join(dir, "b.rego")
			if err := os.Symlink(second, first); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(first, second); err != nil {
				t.Fatal(err)
			}
			return first
		}, "source_unreadable"},
		{"non UTF8 module", func(t *testing.T, dir string) string {
			writePolicyModule(t, filepath.Join(dir, string([]byte{0xff})+".rego"), "package x")
			return dir
		}, "source_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)
			_, err := LoadSource(config.RiskPolicySelection{Path: path}, []Module{{Name: "builtin.rego", Source: "package builtin"}})
			if err == nil || err.Error() != tt.code {
				t.Fatalf("error=%v want fixed %q without fallback", err, tt.code)
			}
		})
	}
}

func TestLoadSource_GivenSymlinkedSource_ShouldLoadTarget(t *testing.T) {
	link := func(t *testing.T, target, path string) {
		t.Helper()
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name  string
		setup func(*testing.T, string) string
		want  []Module
	}{
		{"linked selected file", func(t *testing.T, dir string) string {
			target := filepath.Join(dir, "real.rego")
			writePolicyModule(t, target, "package x")
			path := filepath.Join(dir, "link.rego")
			link(t, target, path)
			return path
		}, []Module{{Name: "link.rego", Source: "package x"}}},
		{"linked ancestor directory", func(t *testing.T, dir string) string {
			real := filepath.Join(dir, "real")
			writePolicyModule(t, filepath.Join(real, "main.rego"), "package x")
			linked := filepath.Join(dir, "link")
			link(t, real, linked)
			return filepath.Join(linked, "main.rego")
		}, []Module{{Name: "main.rego", Source: "package x"}}},
		{"linked selected directory", func(t *testing.T, dir string) string {
			real := filepath.Join(dir, "real")
			writePolicyModule(t, filepath.Join(real, "main.rego"), "package x")
			linked := filepath.Join(dir, "link")
			link(t, real, linked)
			return linked
		}, []Module{{Name: "main.rego", Source: "package x"}}},
		{"linked module entry", func(t *testing.T, dir string) string {
			writePolicyModule(t, filepath.Join(dir, "main.rego"), "package x")
			link(t, filepath.Join(dir, "main.rego"), filepath.Join(dir, "other.rego"))
			return dir
		}, []Module{{Name: "main.rego", Source: "package x"}, {Name: "other.rego", Source: "package x"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t, t.TempDir())
			snapshot, err := LoadSource(config.RiskPolicySelection{Path: path}, nil)
			if err != nil {
				t.Fatalf("LoadSource through a symlink returned %v, want the target loaded", err)
			}
			if snapshot.Kind() != "custom" || !reflect.DeepEqual(snapshot.Modules(), tt.want) {
				t.Fatalf("kind=%q modules=%v, want custom %v", snapshot.Kind(), snapshot.Modules(), tt.want)
			}
		})
	}
}

func TestLoadSource_GivenExcludedFileEdits_ShouldKeepSourceIdentity(t *testing.T) {
	dir := t.TempDir()
	writePolicyModule(t, filepath.Join(dir, "main.rego"), "package x\n")
	first, err := LoadSource(config.RiskPolicySelection{Path: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	writePolicyModule(t, filepath.Join(dir, "main_test.rego"), "package edited_test\n")
	writePolicyModule(t, filepath.Join(dir, "notes.txt"), "changed")
	second, err := LoadSource(config.RiskPolicySelection{Path: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() {
		t.Fatal("excluded author test or unrelated file changed source identity")
	}
}

func TestLoadSource_WhenSelectedFilesChangeDuringCapture_ShouldRejectMixedSnapshot(t *testing.T) {
	tests := []struct {
		name   string
		change func(*testing.T, string)
	}{
		{"edit", func(t *testing.T, dir string) {
			writePolicyModule(t, filepath.Join(dir, "main.rego"), "package changed\n")
		}},
		{"add", func(t *testing.T, dir string) {
			writePolicyModule(t, filepath.Join(dir, "added.rego"), "package added\n")
		}},
		{"remove", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "main.rego")); err != nil {
				t.Fatal(err)
			}
		}},
		{"delete directory", func(t *testing.T, dir string) {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writePolicyModule(t, filepath.Join(dir, "main.rego"), "package original\n")
			reads := 0
			reader := sourceReader{stat: os.Stat, readFile: os.ReadFile, readDir: func(path string) ([]os.DirEntry, error) {
				reads++
				if reads == 2 {
					tt.change(t, dir)
				}
				return os.ReadDir(path)
			}}
			_, err := reader.load(config.RiskPolicySelection{Path: dir}, nil)
			if err == nil || err.Error() != "source_changed" {
				t.Fatalf("concurrent %s returned %v, want source_changed", tt.name, err)
			}
		})
	}
}

func TestLoadSource_WhenSelectedFileChangesBetweenReads_ShouldRejectSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.rego")
	writePolicyModule(t, path, "package original\n")
	reads := 0
	reader := sourceReader{stat: os.Stat, readDir: os.ReadDir, readFile: func(path string) ([]byte, error) {
		reads++
		if reads == 2 {
			writePolicyModule(t, path, "package changed\n")
		}
		return os.ReadFile(path)
	}}
	_, err := reader.load(config.RiskPolicySelection{Path: path}, nil)
	if err == nil || err.Error() != "source_changed" {
		t.Fatalf("file edit returned %v, want source_changed", err)
	}
}

func TestLoadSource_GivenFilesystemFailure_ShouldReturnFixedUnavailableDiagnostic(t *testing.T) {
	tests := []struct {
		name      string
		directory bool
		configure func(*sourceReader, string)
	}{
		{"selected stat denied", false, func(r *sourceReader, _ string) {
			r.stat = func(string) (os.FileInfo, error) { return nil, os.ErrPermission }
		}},
		{"selected file stat disappears before module read", false, func(r *sourceReader, path string) {
			calls := 0
			r.stat = func(p string) (os.FileInfo, error) {
				if p == path {
					calls++
					if calls == 2 {
						return nil, os.ErrNotExist
					}
				}
				return os.Stat(p)
			}
		}},
		{"directory module stat disappears", true, func(r *sourceReader, path string) {
			module := filepath.Join(path, "main.rego")
			r.stat = func(p string) (os.FileInfo, error) {
				if p == module {
					return nil, os.ErrNotExist
				}
				return os.Stat(p)
			}
		}},
		{"directory listing denied", true, func(r *sourceReader, _ string) {
			r.readDir = func(string) ([]os.DirEntry, error) { return nil, os.ErrPermission }
		}},
		{"file read denied", false, func(r *sourceReader, _ string) {
			r.readFile = func(string) ([]byte, error) { return nil, os.ErrPermission }
		}},
		{"directory module read denied", true, func(r *sourceReader, _ string) {
			r.readFile = func(string) ([]byte, error) { return nil, os.ErrPermission }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "main.rego")
			writePolicyModule(t, path, "package x\n")
			if tt.directory {
				path = dir
			}
			reader := sourceReader{stat: os.Stat, readFile: os.ReadFile, readDir: os.ReadDir}
			tt.configure(&reader, path)
			_, err := reader.load(config.RiskPolicySelection{Path: path}, nil)
			if err == nil || err.Error() != "source_unreadable" {
				t.Fatalf("filesystem failure returned %v, want source_unreadable", err)
			}
		})
	}
}

type specialPolicyFileInfo struct{ os.FileInfo }

func (specialPolicyFileInfo) Mode() os.FileMode { return os.ModeNamedPipe }

func TestLoadSource_GivenNonregularSelectedFileOrModule_ShouldRejectWithoutReading(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "main.rego")
			writePolicyModule(t, file, "package x\n")
			selected := file
			if directory {
				selected = dir
			}
			reader := sourceReader{stat: func(path string) (os.FileInfo, error) {
				info, err := os.Stat(path)
				if err == nil && path == file {
					return specialPolicyFileInfo{info}, nil
				}
				return info, err
			}, readFile: func(string) ([]byte, error) { t.Fatal("must not read nonregular policy source"); return nil, nil }, readDir: os.ReadDir}
			_, err := reader.load(config.RiskPolicySelection{Path: selected}, nil)
			if err == nil || err.Error() != "source_invalid" {
				t.Fatalf("nonregular source returned %v, want source_invalid", err)
			}
		})
	}
}

func TestLoadSource_GivenDeletedInvocationDirectory_ShouldReportUnreadable(t *testing.T) {
	dir := t.TempDir()
	cwd := filepath.Join(dir, "cwd")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	if err := os.Remove(cwd); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSource(config.RiskPolicySelection{Path: "main.rego"}, nil)
	if err == nil || err.Error() != "source_unreadable" {
		t.Fatalf("error=%v", err)
	}
}
