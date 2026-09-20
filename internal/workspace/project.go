package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func NewProject(baseDir, projectID string) (string, error) {
	root, err := filepath.Abs(filepath.Join(baseDir, "projects", projectID))
	if err != nil {
		return "", err
	}
	for _, sub := range []string{"shared", "output"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return "", err
		}
	}
	return root, nil
}

func ProjectSharedDir(root string) string { return filepath.Join(root, "shared") }
func ProjectOutputDir(root string) string { return filepath.Join(root, "output") }

func SaveShared(root, name string, r io.Reader) (string, error) {
	name = filepath.Base(name)
	if name == "" || name == "." || name == string(os.PathSeparator) {
		return "", fmt.Errorf("invalid file name")
	}
	dst := filepath.Join(ProjectSharedDir(root), name)
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

func ListShared(root string) ([]FileInfo, error) {
	dir := ProjectSharedDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			RelPath: filepath.ToSlash(filepath.Join("shared", e.Name())),
			Name:    e.Name(),
			Size:    info.Size(),
		})
	}
	return files, nil
}

func CopyTree(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
