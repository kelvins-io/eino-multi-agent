package logx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestInit(t *testing.T) {
	logger, err := Init("debug", "console", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !logger.Core().Enabled(zapcore.DebugLevel) {
		t.Fatal("debug should be enabled")
	}

	logger, err = Init("error", "json", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if logger.Core().Enabled(zapcore.InfoLevel) {
		t.Fatal("info should be disabled at error level")
	}
	if !logger.Core().Enabled(zapcore.ErrorLevel) {
		t.Fatal("error should be enabled")
	}

	if L() == nil {
		t.Fatal("global logger")
	}
	if Named("x") == nil {
		t.Fatal("named logger")
	}
}

func TestInitWithDailyFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "nested", "eino.log")
	old := filepath.Join(dir, "nested", "eino-2000-01-01.log")
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger, err := Init("info", "json", base, 14)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello-file")
	Sync()

	today := filepath.Join(dir, "nested", "eino-"+time.Now().Format("2006-01-02")+".log")
	raw, err := os.ReadFile(today)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "hello-file") {
		t.Fatalf("log file content=%q", raw)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expected old log to be purged")
	}
}
