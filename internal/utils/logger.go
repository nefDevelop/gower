package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Log is the global logger instance
var Log *Logger

// Logger wraps the standard log.Logger with levels
type Logger struct {
	logger *log.Logger
	debug  bool
}

// InitLogger initializes the global logger
func InitLogger(debug bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(home, ".gower", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("gower-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	Log = &Logger{
		logger: log.New(file, "", log.LstdFlags|log.Lmicroseconds),
		debug:  debug,
	}
	return nil
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Printf("[INFO] "+format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Printf("[ERROR] "+format, v...)
}

// Debug logs debug messages if debug mode is enabled
func (l *Logger) Debug(format string, v ...interface{}) {
	if l == nil || l.logger == nil || !l.debug {
		return
	}
	l.logger.Printf("[DEBUG] "+format, v...)
}
