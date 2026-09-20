package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfineRemapsMistypedPrefix(t *testing.T) {
	dir := t.TempDir()
	sb, err := New(dir, "4884e194-7369-4414-a223-6050a9697e89")
	if err != nil {
		t.Fatal(err)
	}
	typoRoot := filepath.Join("/Users/yq/go/src/github.com/klvins-io/eino-multi-agent/workspace", filepath.Base(sb.Root))
	got, err := Confine(sb.Root, typoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != sb.Root {
		t.Fatalf("root remap: got %s want %s", got, sb.Root)
	}
	got, err = Confine(sb.Root, filepath.Join(typoRoot, "input", "sales.csv"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sb.InputDir(), "sales.csv")
	if got != want {
		t.Fatalf("file remap: got %s want %s", got, want)
	}
}

func TestConfineRejectsUnrelatedPath(t *testing.T) {
	dir := t.TempDir()
	sb, err := New(dir, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Confine(sb.Root, "/etc/passwd"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestResolveRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	sb, err := New(dir, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sb.Resolve("../secret"); err == nil {
		t.Fatal("expected escape error")
	}
	p, err := sb.Resolve("output/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p) != sb.OutputDir() {
		t.Fatalf("got %s", p)
	}
}

func TestSaveUploadAndScan(t *testing.T) {
	dir := t.TempDir()
	sb, err := New(dir, "task-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sb.SaveUpload("hello.txt", strings.NewReader("hi")); err != nil {
		t.Fatal(err)
	}
	inputs, err := sb.ListInputs()
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 1 {
		t.Fatalf("inputs=%d", len(inputs))
	}
	if err := os.WriteFile(filepath.Join(sb.OutputDir(), "report.md"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	outs, err := sb.ScanOutput()
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || outs[0].Name != "report.md" {
		t.Fatalf("unexpected artifacts: %+v", outs)
	}
}
