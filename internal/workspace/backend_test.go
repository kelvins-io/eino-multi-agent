package workspace

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/filesystem"
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

func TestWriteOverwritesConfirmOnRisk(t *testing.T) {
	sb := newTestSandbox(t)
	ctx := context.Background()
	b, err := NewBackend(ctx, sb, "on_risk")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sb.OutputDir(), "report.md")
	first := &filesystem.WriteRequest{FilePath: path, Content: "v1"}
	if err := b.Write(ctx, first); err != nil {
		t.Fatalf("first write: %v", err)
	}
	err = b.Write(ctx, &filesystem.WriteRequest{FilePath: path, Content: "v2"})
	if err == nil {
		t.Fatal("overwrite should interrupt")
	}
	if !strings.Contains(err.Error(), "请求确认") && !strings.Contains(err.Error(), "interrupt") {
		t.Fatalf("unexpected error: %v", err)
	}

	never, err := NewBackend(ctx, sb, "never")
	if err != nil {
		t.Fatal(err)
	}
	if err := never.Write(ctx, &filesystem.WriteRequest{FilePath: path, Content: "v3"}); err != nil {
		t.Fatalf("never should overwrite: %v", err)
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
