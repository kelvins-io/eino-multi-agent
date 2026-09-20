package logx

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger = zap.NewNop()

// Init builds a zap logger and sets it as the process-wide default.
// level: debug|info|warn|error (default info)
// format: console|json (default console)
// file: optional base path; when set, logs also go to <name>-YYYY-MM-DD.log
// keepDays: delete dated files older than this many days; 0 keeps them all
func Init(level, format, file string, keepDays int) (*zap.Logger, error) {
	lvl := parseLevel(level)
	console := strings.EqualFold(strings.TrimSpace(format), "console") || strings.TrimSpace(format) == ""

	stdoutEnc := newEncoder(console, console)
	cores := []zapcore.Core{
		zapcore.NewCore(stdoutEnc, zapcore.AddSync(os.Stdout), lvl),
	}

	file = strings.TrimSpace(file)
	if file != "" {
		w, err := newDailyWriter(file, keepDays)
		if err != nil {
			return nil, err
		}
		// File sink stays uncolored so dated logs stay readable.
		cores = append(cores, zapcore.NewCore(newEncoder(console, false), zapcore.AddSync(w), lvl))
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.ErrorOutput(zapcore.AddSync(os.Stderr)))
	global = logger
	zap.ReplaceGlobals(logger)
	return logger, nil
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func newEncoder(console, color bool) zapcore.Encoder {
	if console {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		if color {
			cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			cfg.EncodeLevel = zapcore.CapitalLevelEncoder
		}
		return zapcore.NewConsoleEncoder(cfg)
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.TimeKey = "ts"
	return zapcore.NewJSONEncoder(cfg)
}

// L returns the global logger (never nil).
func L() *zap.Logger {
	if global == nil {
		return zap.NewNop()
	}
	return global
}

// Named returns a named child logger.
func Named(name string) *zap.Logger {
	return L().Named(name)
}

// Sync flushes buffered log entries.
func Sync() {
	_ = L().Sync()
}
