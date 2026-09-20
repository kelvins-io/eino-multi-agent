package workspace

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type Sandbox struct {
	Root string
}

func New(baseDir, taskID string) (*Sandbox, error) {
	root, err := filepath.Abs(filepath.Join(baseDir, taskID))
	if err != nil {
		return nil, err
	}
	for _, sub := range []string{"input", "output", "tmp"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return nil, err
		}
	}
	return &Sandbox{Root: root}, nil
}

func Open(root string) (*Sandbox, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Root: abs}, nil
}

func (s *Sandbox) InputDir() string  { return filepath.Join(s.Root, "input") }
func (s *Sandbox) OutputDir() string { return filepath.Join(s.Root, "output") }
func (s *Sandbox) TmpDir() string    { return filepath.Join(s.Root, "tmp") }

func (s *Sandbox) Resolve(p string) (string, error) {
	return Confine(s.Root, p)
}

// Confine maps a model-supplied path into the sandbox.
// Relative paths and mistyped absolute prefixes are accepted as long as the
// result stays under root. The task directory name (usually the task ID) is
// used to recover paths like /wrong/prefix/<taskID>/input/a.csv.
func Confine(root, p string) (string, error) {
	root = filepath.Clean(root)
	p = strings.TrimSpace(p)
	if p == "" || p == "." || p == string(filepath.Separator) {
		return root, nil
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if inside(root, abs) {
		return abs, nil
	}
	if mapped, ok := remapViaTaskID(root, abs); ok {
		return mapped, nil
	}
	return "", fmt.Errorf("path outside workspace %s: %s; use this exact root or a relative path like input/", root, p)
}

func remapViaTaskID(root, abs string) (string, bool) {
	taskID := filepath.Base(root)
	if taskID == "" || taskID == "." || taskID == string(filepath.Separator) {
		return "", false
	}
	marker := string(filepath.Separator) + taskID
	idx := strings.LastIndex(abs, marker)
	if idx < 0 {
		if filepath.Base(abs) == taskID {
			return root, true
		}
		return "", false
	}
	rest := strings.TrimPrefix(abs[idx+len(marker):], string(filepath.Separator))
	mapped := filepath.Join(root, rest)
	mapped, err := filepath.Abs(mapped)
	if err != nil || !inside(root, mapped) {
		return "", false
	}
	return mapped, true
}

func inside(root, abs string) bool {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (s *Sandbox) SaveUpload(name string, r io.Reader) (string, error) {
	name = filepath.Base(name)
	if name == "" || name == "." || name == string(os.PathSeparator) {
		return "", fmt.Errorf("invalid file name")
	}
	dst := filepath.Join(s.InputDir(), name)
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return dst, nil
}

func (s *Sandbox) ListInputs() ([]string, error) {
	entries, err := os.ReadDir(s.InputDir())
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, filepath.Join(s.InputDir(), e.Name()))
	}
	return files, nil
}

type FileInfo struct {
	RelPath string
	Name    string
	Size    int64
	Mime    string
}

func (s *Sandbox) ScanOutput() ([]FileInfo, error) {
	var files []FileInfo
	err := filepath.Walk(s.OutputDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		files = append(files, FileInfo{
			RelPath: filepath.ToSlash(rel),
			Name:    info.Name(),
			Size:    info.Size(),
			Mime:    mimeType,
		})
		return nil
	})
	return files, err
}

func (s *Sandbox) ReadArtifact(relPath string) (string, error) {
	abs, err := s.Resolve(relPath)
	if err != nil {
		return "", err
	}
	outRoot := s.OutputDir()
	rel, err := filepath.Rel(outRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("artifact must be under output/")
	}
	return abs, nil
}
