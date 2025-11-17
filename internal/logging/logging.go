package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	initOnce sync.Once
	initErr  error

	logger   *log.Logger
	levelMu  sync.RWMutex
	logLevel = LevelInfo
)

// Init sets up the logger output. Safe to call multiple times; only the first
// call performs initialization.
func Init(logPath string) error {
	initOnce.Do(func() {
		var writer io.Writer = os.Stderr

		if logPath != "" {
			if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
				initErr = err
				return
			}

			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				initErr = err
				return
			}
			writer = file
		}

		logger = log.New(writer, "", log.LstdFlags|log.Lmicroseconds)
	})

	return initErr
}

// SetLevel updates the global log level (debug, info, warn, error).
func SetLevel(value string) {
	levelMu.Lock()
	defer levelMu.Unlock()
	logLevel = parseLevel(value)
}

func parseLevel(value string) Level {
	switch strings.ToLower(value) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func Debugf(format string, args ...interface{}) {
	logf(LevelDebug, "DEBUG", format, args...)
}

func Infof(format string, args ...interface{}) {
	logf(LevelInfo, "INFO", format, args...)
}

func Warnf(format string, args ...interface{}) {
	logf(LevelWarn, "WARN", format, args...)
}

func Errorf(format string, args ...interface{}) {
	logf(LevelError, "ERROR", format, args...)
}

func logf(entryLevel Level, prefix, format string, args ...interface{}) {
	levelMu.RLock()
	current := logLevel
	levelMu.RUnlock()

	if entryLevel < current || logger == nil {
		return
	}

	logger.Printf("[%s] %s", prefix, fmt.Sprintf(format, args...))
}
