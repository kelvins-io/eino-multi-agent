package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// dailyWriter appends to prefix-YYYY-MM-DD.log and switches files at local midnight.
type dailyWriter struct {
	dir      string
	prefix   string
	keepDays int

	mu  sync.Mutex
	day string
	f   *os.File
}

func newDailyWriter(path string, keepDays int) (*dailyWriter, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("log file path: %w", err)
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	base := filepath.Base(abs)
	prefix := strings.TrimSuffix(base, filepath.Ext(base))
	if prefix == "" || prefix == "." {
		prefix = "app"
	}
	w := &dailyWriter{dir: dir, prefix: prefix, keepDays: keepDays}
	if err := w.rotate(time.Now()); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotate(time.Now()); err != nil {
		return 0, err
	}
	return w.f.Write(p)
}

func (w *dailyWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	return w.f.Sync()
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

func (w *dailyWriter) rotate(now time.Time) error {
	day := now.Format("2006-01-02")
	if w.f != nil && w.day == day {
		return nil
	}
	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}
	path := filepath.Join(w.dir, w.prefix+"-"+day+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.f = f
	w.day = day
	w.purge(now)
	return nil
}

func (w *dailyWriter) purge(now time.Time) {
	if w.keepDays <= 0 {
		return
	}
	cutoff := now.AddDate(0, 0, -w.keepDays)
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	prefix := w.prefix + "-"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".log") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".log")
		day, err := time.ParseInLocation("2006-01-02", datePart, now.Location())
		if err != nil {
			continue
		}
		if day.Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, name))
		}
	}
}
