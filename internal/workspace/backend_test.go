package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteCommandAllowsRelativeScript(t *testing.T) {
	sb := newTestSandbox(t)
	got, err := rewriteCommand(sb.Root, "python3 tmp/compute.py")
	if err != nil {
		t.Fatal(err)
	}
	if got != "python3 tmp/compute.py" {
		t.Fatalf("relative path should stay unchanged, got %q", got)
	}
	if _, err := rewriteCommand(sb.Root, "python3 ./tmp/compute.py"); err != nil {
		t.Fatal(err)
	}
	if _, err := rewriteCommand(sb.Root, `python3 -c "print(1/2)"`); err != nil {
		t.Fatal(err)
	}
}

func TestRewriteCommandMapsSandboxAbsolute(t *testing.T) {
	sb := newTestSandbox(t)
	got, err := rewriteCommand(sb.Root, "python3 /tmp/compute.py")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sb.TmpDir(), "compute.py")
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}

	got, err = rewriteCommand(sb.Root, "python3 /compute.py")
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(sb.Root, "compute.py")
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}

func TestRewriteCommandKeepsWorkspaceAndSystemPaths(t *testing.T) {
	sb := newTestSandbox(t)
	script := filepath.Join(sb.TmpDir(), "compute.py")
	got, err := rewriteCommand(sb.Root, "python3 "+script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, script) {
		t.Fatalf("got %q", got)
	}
	if _, err := rewriteCommand(sb.Root, "python3 /usr/bin/python3 tmp/compute.py"); err != nil {
		t.Fatal(err)
	}
}

func TestRewriteCommandRemapsMistypedPrefix(t *testing.T) {
	sb := newTestSandbox(t)
	typo := filepath.Join("/Users/yq/wrong-prefix", filepath.Base(sb.Root), "tmp", "compute.py")
	got, err := rewriteCommand(sb.Root, "python3 "+typo)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sb.TmpDir(), "compute.py")
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}

func TestValidateCommandRejectsEscape(t *testing.T) {
	sb := newTestSandbox(t)
	if err := validateCommand(sb.Root, "cat /etc/passwd"); err == nil {
		t.Fatal("expected deny")
	}
	if err := validateCommand(sb.Root, "python3 /Users/other/secret.py"); err == nil {
		t.Fatal("expected outside workspace")
	}
}

func newTestSandbox(t *testing.T) *Sandbox {
	t.Helper()
	sb, err := New(t.TempDir(), "task-cmd")
	if err != nil {
		t.Fatal(err)
	}
	return sb
}
