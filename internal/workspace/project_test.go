package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyTreeMissingSrc(t *testing.T) {
	dir := t.TempDir()
	if err := CopyTree(filepath.Join(dir, "nope"), filepath.Join(dir, "dst")); err != nil {
		t.Fatal(err)
	}
}

func TestNewProjectAndShared(t *testing.T) {
	dir := t.TempDir()
	root, err := NewProject(dir, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveShared(root, "notes.txt", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	files, err := ListShared(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "notes.txt" {
		t.Fatalf("%+v", files)
	}
	dst := filepath.Join(dir, "copy")
	if err := CopyTree(ProjectSharedDir(root), dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}
