package utils

import (
	"fmt"
	"io"
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

// NewLogger creates a new Logger instance that writes to the provided io.Writer.
func NewLogger(writer io.Writer, debug bool) *Logger {
	return &Logger{
		logger: log.New(writer, "", log.LstdFlags|log.Lmicroseconds),
		debug:  debug,
	}
}

// InitLogger initializes the global logger to write to a file.
func InitLogger(debug bool) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}

	logDir := filepath.Join(configDir, "gower", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("gower-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	var writers []io.Writer
	writers = append(writers, file) // Siempre escribir al archivo

	if debug {
		writers = append(writers, os.Stdout) // También escribir a la consola si el modo depuración está activado
	}

	Log = NewLogger(io.MultiWriter(writers...), debug)
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
