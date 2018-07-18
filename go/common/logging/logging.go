// Package logging implements support for structured logging.
//
// This package is inspired heavily by go-logging, kit/log and the
// tendermint libs/log packages, and is oriented towards making
// the structured logging experience somewhat easier to use.
package logging

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var backend = logBackend{
	baseLogger: log.NewNopLogger(),
	level:      LevelError,
}

// Format is a logging format.
type Format uint

const (
	// FmtLogfmt is the "logfmt" logging format.
	FmtLogfmt Format = iota
	// FmtJSON is the JSON logging format.
	FmtJSON
)

// LogFormat returns the Format corresponding to the provided string.
func LogFormat(s string) (Format, error) {
	switch strings.ToUpper(s) {
	case "LOGFMT":
		return FmtLogfmt, nil
	case "JSON":
		return FmtJSON, nil
	default:
	}
	return FmtLogfmt, fmt.Errorf("logging: invalid log format: '%s'", s)
}

// Level is a log level.
type Level uint

const (
	// LevelDebug is the log level for debug messages.
	LevelDebug Level = iota
	// LevelInfo is the log level for informative messages.
	LevelInfo
	// LevelWarn is the log level for warning messages.
	LevelWarn
	// LevelError is the log level for error messages.
	LevelError
)

func (l Level) toOption() level.Option {
	switch l {
	case LevelDebug:
		return level.AllowDebug()
	case LevelInfo:
		return level.AllowInfo()
	case LevelWarn:
		return level.AllowWarn()
	case LevelError:
		return level.AllowError()
	default:
		panic("logging: unsupported log level")
	}
}

// LogLevel returns the Level corrsponding to the provided string.
func LogLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	default:
	}
	return LevelError, fmt.Errorf("logging: invalid log level: '%s'", s)
}

// Logger is a logger instance.
type Logger struct {
	logger log.Logger
}

// Debug logs the message and key value pairs at the Debug log level.
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	if backend.level > LevelDebug {
		return
	}
	keyvals = append([]interface{}{"msg", msg}, keyvals...)
	_ = level.Debug(l.logger).Log(keyvals...)
}

// Info logs the message and key value pairs at the Debug log level.
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	if backend.level > LevelInfo {
		return
	}
	keyvals = append([]interface{}{"msg", msg}, keyvals...)
	_ = level.Info(l.logger).Log(keyvals...)
}

// Warn logs the message and key value pairs at the Debug log level.
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	if backend.level > LevelWarn {
		return
	}
	keyvals = append([]interface{}{"msg", msg}, keyvals...)
	_ = level.Warn(l.logger).Log(keyvals...)
}

// Error logs the message and key value pairs at the Debug log level.
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	if backend.level > LevelError {
		return
	}
	keyvals = append([]interface{}{"msg", msg}, keyvals...)
	_ = level.Error(l.logger).Log(keyvals...)
}

// With returns a clone of the logger with the provided key/value pairs
// added via log.WithPrefix.
func (l *Logger) With(keyvals ...interface{}) *Logger {
	return &Logger{
		logger: log.With(l.logger, keyvals...),
	}
}

// GetLogger creates a new logger instance with the specified module.
//
// This may be called from any point, including before Initialize is
// called, allowing for the construction of a package level Logger.
func GetLogger(module string) *Logger {
	return backend.getLogger(module)
}

// Initialize initializes the logging backend to write to the provided
// Writer with the default log level and format.  If the Writer is nil,
// all log output will be silently discarded.
func Initialize(w io.Writer, lvl Level, format Format) error {
	backend.Lock()
	defer backend.Unlock()

	if backend.initialized {
		return fmt.Errorf("logging: already initialized")
	}

	var logger log.Logger = backend.baseLogger
	if w != nil {
		w = log.NewSyncWriter(w)
		switch format {
		case FmtLogfmt:
			logger = log.NewLogfmtLogger(w)
		case FmtJSON:
			// TODO: This uses encoding/json, which may be too slow.
			// The go-codec encoder should be faster.
			logger = log.NewJSONLogger(w)
		default:
			return fmt.Errorf("logging: unsupported log format: %v", format)
		}
	}

	logger = level.NewFilter(logger, lvl.toOption())
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	backend.baseLogger = logger
	backend.level = lvl
	backend.initialized = true

	// Swap all the early loggers to the initialized backend.
	for _, l := range backend.earlyLoggers {
		l.Swap(backend.baseLogger)
	}
	backend.earlyLoggers = nil

	return nil
}

type logBackend struct {
	sync.Mutex

	baseLogger   log.Logger
	earlyLoggers []*log.SwapLogger
	level        Level

	initialized bool
}

func (b *logBackend) getLogger(module string) *Logger {
	b.Lock()
	defer b.Unlock()

	logger := b.baseLogger
	if !b.initialized {
		logger = &log.SwapLogger{}
	}

	// The caller is log.DefaultCaller with an extra level of stack
	// unwinding due to this module's leveling wrapper.
	l := &Logger{
		logger: log.WithPrefix(logger, "module", module, "caller", log.Caller(4)),
	}

	if !b.initialized {
		// Stash the logger so that it can be instantiated once logging
		// is actually initialized.
		sLog := logger.(*log.SwapLogger)
		backend.earlyLoggers = append(backend.earlyLoggers, sLog)
	}

	return l
}