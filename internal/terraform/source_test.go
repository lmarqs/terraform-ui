package terraform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSourceIndex(t *testing.T) {
	t.Run("indexes resource blocks", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "main" {
  bucket = "my-bucket"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("aws_s3_bucket.main")
		if !ok {
			t.Fatal("Lookup(aws_s3_bucket.main) not found")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
		if loc.File != filepath.Join(dir, "main.tf") {
			t.Errorf("File = %q, want %q", loc.File, filepath.Join(dir, "main.tf"))
		}
	})

	t.Run("indexes data blocks", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "data.tf"), `
data "aws_ami" "latest" {
  most_recent = true
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("data.aws_ami.latest")
		if !ok {
			t.Fatal("Lookup(data.aws_ami.latest) not found")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})

	t.Run("indexes module blocks", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "modules.tf"), `
module "vpc" {
  source = "./modules/vpc"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("module.vpc")
		if !ok {
			t.Fatal("Lookup(module.vpc) not found")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})

	t.Run("indexes multiple resources in one file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "first" {
  bucket = "first"
}

resource "aws_s3_bucket" "second" {
  bucket = "second"
}

data "aws_caller_identity" "current" {}

module "networking" {
  source = "./modules/networking"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		if idx.Count() != 4 {
			t.Errorf("Count() = %d, want 4", idx.Count())
		}

		if _, ok := idx.Lookup("aws_s3_bucket.first"); !ok {
			t.Error("aws_s3_bucket.first not found")
		}
		if _, ok := idx.Lookup("aws_s3_bucket.second"); !ok {
			t.Error("aws_s3_bucket.second not found")
		}
		if _, ok := idx.Lookup("data.aws_caller_identity.current"); !ok {
			t.Error("data.aws_caller_identity.current not found")
		}
		if _, ok := idx.Lookup("module.networking"); !ok {
			t.Error("module.networking not found")
		}
	})

	t.Run("skips .terraform directory", func(t *testing.T) {
		dir := t.TempDir()
		terraformDir := filepath.Join(dir, ".terraform", "providers")
		if err := os.MkdirAll(terraformDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(terraformDir, "provider.tf"), `
resource "aws_s3_bucket" "hidden" {
  bucket = "hidden"
}
`)
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "visible" {
  bucket = "visible"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		if _, ok := idx.Lookup("aws_s3_bucket.hidden"); ok {
			t.Error("aws_s3_bucket.hidden should not be indexed (inside .terraform)")
		}
		if _, ok := idx.Lookup("aws_s3_bucket.visible"); !ok {
			t.Error("aws_s3_bucket.visible should be indexed")
		}
	})

	t.Run("skips .git directory", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git", "hooks")
		if err := os.MkdirAll(gitDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(gitDir, "something.tf"), `
resource "aws_s3_bucket" "git_resource" {
  bucket = "git"
}
`)
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "real" {
  bucket = "real"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		if _, ok := idx.Lookup("aws_s3_bucket.git_resource"); ok {
			t.Error("aws_s3_bucket.git_resource should not be indexed (inside .git)")
		}
		if _, ok := idx.Lookup("aws_s3_bucket.real"); !ok {
			t.Error("aws_s3_bucket.real should be indexed")
		}
	})

	t.Run("empty directory returns empty index", func(t *testing.T) {
		dir := t.TempDir()
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}
		if idx.Count() != 0 {
			t.Errorf("Count() = %d, want 0", idx.Count())
		}
	})
}

func TestSourceIndexLookup(t *testing.T) {
	t.Run("existing address returns location", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {
  ami = "ami-123"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("aws_instance.web")
		if !ok {
			t.Fatal("Lookup() returned false")
		}
		if loc.Col != 1 {
			t.Errorf("Col = %d, want 1", loc.Col)
		}
	})

	t.Run("non-existing address returns false", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {
  ami = "ami-123"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		_, ok := idx.Lookup("aws_instance.nonexistent")
		if ok {
			t.Error("Lookup() returned true for non-existing address")
		}
	})

	t.Run("module-prefixed address falls back to leaf", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {
  ami = "ami-123"
}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("module.foo.aws_instance.web")
		if !ok {
			t.Fatal("Lookup() should find module-prefixed address via leaf fallback")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})

	t.Run("nested module prefix falls back to leaf", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `
data "aws_ssoadmin_instances" "this" {}
`)
		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.Lookup("module.identity_center.data.aws_ssoadmin_instances.this")
		if !ok {
			t.Fatal("Lookup() should find deeply nested module address via leaf fallback")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})
}

func TestStripModulePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aws_instance.web", "aws_instance.web"},
		{"module.foo.aws_instance.web", "aws_instance.web"},
		{"module.foo.module.bar.aws_instance.web", "aws_instance.web"},
		{"module.foo.data.aws_ami.latest", "data.aws_ami.latest"},
		{"data.aws_ami.latest", "data.aws_ami.latest"},
		{"module.x", "module.x"},
		{`module.user["github.com"].aws_iam_user.this`, "aws_iam_user.this"},
		{`module.user["devin.ai"].aws_iam_access_key.this_no_pgp`, "aws_iam_access_key.this_no_pgp"},
		{`module.a["x.y"].module.b["z"].aws_instance.web`, "aws_instance.web"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripModulePrefix(tt.input)
			if got != tt.want {
				t.Errorf("stripModulePrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLookup_IndexedResources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_iam_user" "this" {
  name = "test"
}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	t.Run("indexed resource falls back to base", func(t *testing.T) {
		loc, ok := idx.Lookup("aws_iam_user.this[0]")
		if !ok {
			t.Fatal("Lookup() should find indexed resource via base fallback")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})

	t.Run("module plus index falls back", func(t *testing.T) {
		loc, ok := idx.Lookup(`module.user["github.com"].aws_iam_user.this[0]`)
		if !ok {
			t.Fatal("Lookup() should find module+indexed address")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2", loc.Line)
		}
	})
}

func TestLookup_ModuleCallFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
module "user" {
  source   = "../modules/user"
  for_each = var.users
}

module "vpc" {
  source = "../modules/vpc"
}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	t.Run("external resource resolves to module call", func(t *testing.T) {
		loc, ok := idx.Lookup(`module.user["github.com"].aws_iam_user.this[0]`)
		if !ok {
			t.Fatal("Lookup() should fall back to module.user declaration")
		}
		if loc.Line != 2 {
			t.Errorf("Line = %d, want 2 (module block line)", loc.Line)
		}
	})

	t.Run("nested module resolves to outermost known call", func(t *testing.T) {
		loc, ok := idx.Lookup("module.vpc.module.subnets.aws_subnet.private[0]")
		if !ok {
			t.Fatal("Lookup() should fall back to module.vpc declaration")
		}
		if loc.Line != 7 {
			t.Errorf("Line = %d, want 7 (module vpc block line)", loc.Line)
		}
	})

	t.Run("completely unknown address returns false", func(t *testing.T) {
		_, ok := idx.Lookup("aws_nonexistent.thing")
		if ok {
			t.Error("Lookup() should return false for completely unknown address")
		}
	})
}

func TestSourceIndexLookupFile(t *testing.T) {
	t.Run("directory with main.tf returns main.tf", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.tf"), `resource "null_resource" "x" {}`)
		writeFile(t, filepath.Join(dir, "other.tf"), `resource "null_resource" "y" {}`)

		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.LookupFile(dir)
		if !ok {
			t.Fatal("LookupFile() returned false")
		}
		if loc.File != filepath.Join(dir, "main.tf") {
			t.Errorf("File = %q, want %q", loc.File, filepath.Join(dir, "main.tf"))
		}
		if loc.Line != 1 {
			t.Errorf("Line = %d, want 1", loc.Line)
		}
	})

	t.Run("directory without main.tf returns first .tf file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "alpha.tf"), `resource "null_resource" "a" {}`)

		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		loc, ok := idx.LookupFile(dir)
		if !ok {
			t.Fatal("LookupFile() returned false")
		}
		if loc.File != filepath.Join(dir, "alpha.tf") {
			t.Errorf("File = %q, want %q", loc.File, filepath.Join(dir, "alpha.tf"))
		}
	})

	t.Run("empty directory returns false", func(t *testing.T) {
		dir := t.TempDir()

		idx, err := NewSourceIndex(dir)
		if err != nil {
			t.Fatalf("NewSourceIndex() error = %v", err)
		}

		_, ok := idx.LookupFile(dir)
		if ok {
			t.Error("LookupFile() returned true for empty directory")
		}
	})
}

func TestSourceIndexCount(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "one" {}
resource "aws_s3_bucket" "two" {}
data "aws_ami" "three" {}
module "four" { source = "./m" }
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	if idx.Count() != 4 {
		t.Errorf("Count() = %d, want 4", idx.Count())
	}
}

func TestSourceIndexLineNumbers(t *testing.T) {
	dir := t.TempDir()
	content := `# Some comment
# Another comment

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id = aws_vpc.main.id
}
`
	writeFile(t, filepath.Join(dir, "network.tf"), content)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	loc, ok := idx.Lookup("aws_vpc.main")
	if !ok {
		t.Fatal("aws_vpc.main not found")
	}
	if loc.Line != 4 {
		t.Errorf("aws_vpc.main Line = %d, want 4", loc.Line)
	}

	loc, ok = idx.Lookup("aws_subnet.public")
	if !ok {
		t.Fatal("aws_subnet.public not found")
	}
	if loc.Line != 8 {
		t.Errorf("aws_subnet.public Line = %d, want 8", loc.Line)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
