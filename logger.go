package logger

import (
	"context"
	"io"
	"os"
	"strings"
)

var (
	DefaultOutput io.Writer = os.Stderr
	DefaultLevel            = Info
)

type Format []interface{}

func Fmt(str string, args ...interface{}) Format {
	return append(Format{str}, args...)
}

type Hex int

type Octal int

type Binary int

type Level int32

const (
	NoLevel Level = 0

	Trace Level = 1

	Debug Level = 2

	Info Level = 3

	Warn Level = 4

	Error Level = 5
)

type ColorOption uint8

const (
	ColorOff ColorOption = iota
	AutoColor
	ForceColor
)

func LevelFromString(levelStr string) Level {
	levelStr = strings.ToLower(strings.TrimSpace(levelStr))
	switch levelStr {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	default:
		return NoLevel
	}
}

func (l Level) String() string {
	switch l {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	case NoLevel:
		return "none"
	default:
		return "unknown"
	}
}

type Logger interface {
	Log(level Level, msg string, args ...interface{})

	Trace(msg string, args ...interface{})

	Debug(msg string, args ...interface{})

	Info(msg string, args ...interface{})

	Warn(msg string, args ...interface{})

	Error(msg string, args ...interface{})

	Panic(msg string, args ...interface{})

	ErrorPanic(err error, args ...interface{})

	IsTrace() bool

	IsDebug() bool

	IsInfo() bool
	IsWarn() bool

	IsError() bool

	ImpliedArgs() []interface{}

	With(args ...interface{}) Logger

	Name() string

	Named(name string) Logger

	ResetNamed(name string) Logger

	SetLevel(level Level)

	Close()
}

type LoggerOptions struct {
	Name string

	Level Level

	Output []io.Writer

	Mutex Locker

	JSONFormat bool

	IncludeLocation bool

	TimeFormat string

	DisableTime bool

	EnableDailyLog bool

	DailyLogDir string

	KeepNumDays int

	ctx context.Context

	Color []ColorOption

	Exclude func(level Level, msg string, args ...interface{}) bool
}

type Locker interface {
	Lock()

	Unlock()
}

type Flushable interface {
	Flush() error
}

type OutputResettable interface {
	ResetOutput(opts *LoggerOptions) error
	ResetOutputWithFlush(opts *LoggerOptions, flushable Flushable) error
}

type NoopLocker struct{}

func (n NoopLocker) Lock() {}

func (n NoopLocker) Unlock() {}

var _ Locker = (*NoopLocker)(nil)
