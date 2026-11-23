package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Log is the global logger instance.
var Log *Logger

// Logger wraps the standard log.Logger with levels and daily rotation.
type Logger struct {
	mu     sync.Mutex
	logger *log.Logger
	debug  bool
	file   *os.File
	dir    string
	day    string
}

// NewLogger creates a Logger that writes to the provided writer.
func NewLogger(writer io.Writer, debug bool) *Logger {
	return &Logger{
		logger: log.New(writer, "", log.LstdFlags|log.Lmicroseconds),
		debug:  debug,
	}
}

// InitLogger configures the global logger to write to the app log directory,
// creating a new file per day. Logs are readable only by the owner.
func InitLogger(debug bool) error {
	dir, err := LogDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, day, err := openLogFile(dir)
	if err != nil {
		return err
	}

	writers := []io.Writer{file}
	if debug {
		writers = append(writers, os.Stderr)
	}

	Log = &Logger{
		logger: log.New(io.MultiWriter(writers...), "", log.LstdFlags|log.Lmicroseconds),
		debug:  debug,
		file:   file,
		dir:    dir,
		day:    day,
	}
	return nil
}

func openLogFile(dir string) (*os.File, string, error) {
	day := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, fmt.Sprintf("gower-%s.log", day))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, "", err
	}
	return file, day, nil
}

// Close cierra el fichero de log. Es seguro llamarlo varias veces.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// CloseLogger cierra el logger global, si existe.
func CloseLogger() error {
	if Log == nil {
		return nil
	}
	return Log.Close()
}

// rotateLocked cambia al fichero del día actual. Debe llamarse con l.mu tomado.
func (l *Logger) rotateLocked() {
	if l.dir == "" {
		return
	}
	file, day, err := openLogFile(l.dir)
	if err != nil {
		return
	}
	if l.file != nil {
		_ = l.file.Close()
	}
	l.file = file
	l.day = day

	writers := []io.Writer{file}
	if l.debug {
		writers = append(writers, os.Stderr)
	}
	l.logger.SetOutput(io.MultiWriter(writers...))
}

func (l *Logger) output(level, format string, v ...interface{}) {
	if l == nil || l.logger == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil && time.Now().Format("2006-01-02") != l.day {
		l.rotateLocked()
	}
	l.logger.Printf(level+" "+format, v...)
}

// Info logs informational messages.
func (l *Logger) Info(format string, v ...interface{}) {
	l.output("[INFO]", format, v...)
}

// Error logs error messages.
func (l *Logger) Error(format string, v ...interface{}) {
	l.output("[ERROR]", format, v...)
}

// Debug logs debug messages if debug mode is enabled.
func (l *Logger) Debug(format string, v ...interface{}) {
	if l == nil || l.logger == nil || !l.debug {
		return
	}
	l.output("[DEBUG]", format, v...)
}
