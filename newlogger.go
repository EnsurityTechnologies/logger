package logger

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
)

const TimeFormat = "2006-01-02T15:04:05.000Z0700"

const errJsonUnsupportedTypeMsg = "logging contained values that don't serialize to json"

const MissingKey = "EXTRA_VALUE_AT_END"

var (
	_levelToBracket = map[Level]string{
		Debug: "[DEBUG]",
		Trace: "[TRACE]",
		Info:  "[INFO] ",
		Warn:  "[WARN] ",
		Error: "[ERROR]",
	}

	_levelToColor = map[Level]*color.Color{
		Debug: color.New(color.FgHiWhite),
		Trace: color.New(color.FgHiGreen),
		Info:  color.New(color.FgHiBlue),
		Warn:  color.New(color.FgHiYellow),
		Error: color.New(color.FgHiRed),
	}
)

var _ Logger = &newLogger{}

type newLogger struct {
	json       bool
	caller     bool
	name       string
	timeFormat string
	dir        string
	fw         *fileWrite
	ctx        context.Context

	mutex  Locker
	writer *writer
	level  *int32

	implied []interface{}

	exclude func(level Level, msg string, args ...interface{}) bool
}

func New(opts *LoggerOptions) Logger {

	if opts == nil {
		opts = &LoggerOptions{}
	}

	output := opts.Output
	if output == nil {
		output = []io.Writer{DefaultOutput}
	}

	level := opts.Level
	if level == NoLevel {
		level = DefaultLevel
	}

	mutex := opts.Mutex
	if mutex == nil {
		mutex = new(sync.Mutex)
	}

	if opts.DailyLogDir == "" {
		opts.DailyLogDir = "./"
	}

	if opts.ctx == nil {
		opts.ctx = context.Background()
	}

	l := &newLogger{
		json:       opts.JSONFormat,
		caller:     opts.IncludeLocation,
		name:       opts.Name,
		timeFormat: TimeFormat,
		writer:     newWriter(output, opts.Color),
		mutex:      mutex,
		dir:        opts.DailyLogDir,
		ctx:        opts.ctx,
		level:      new(int32),
		exclude:    opts.Exclude,
	}

	l.setColorization(opts)

	if opts.EnableDailyLog {
		l.createDailyLog()
		go l.dailyThread()
	}

	if opts.DisableTime {
		l.timeFormat = ""
	} else if opts.TimeFormat != "" {
		l.timeFormat = opts.TimeFormat
	}

	atomic.StoreInt32(l.level, int32(level))

	return l
}

func (l *newLogger) dailyThread() {
	for {
		now := time.Now()
		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		dur := nextDay.Sub(now)
		ticker := time.NewTicker(dur)
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.createDailyLog()
		}
	}
}

func (l *newLogger) createDailyLog() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.fw != nil {
		l.fw.Close()
	}
	var err error
	t := time.Now()
	fn := l.dir + fmt.Sprintf("%02d-%02d-%04d", t.Day(), t.Month(), t.Year()) + ".log"
	l.fw, err = newFileWrite(fn)
	if err != nil {
		return false
	}
	l.writer.UpdateWriter(l.fw)
	return true
}

func NewDefaultLog(ctx context.Context, name string, level Level, dir string) Logger {
	opt := &LoggerOptions{
		Name:  name,
		Level: level,
		Output: []io.Writer{
			DefaultOutput,
		},
		Color: []ColorOption{
			AutoColor,
		},
		EnableDailyLog: true,
		DailyLogDir:    dir,
		ctx:            ctx,
	}
	return New(opt)
}
func (l *newLogger) log(name string, level Level, msg string, args ...interface{}) {
	if level < Level(atomic.LoadInt32(l.level)) {
		return
	}

	t := time.Now()

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.json {
		l.logJSON(t, name, level, msg, args...)
	} else {
		l.logPlain(t, name, level, msg, args...)
	}

	l.writer.Flush(level)
}

func trimCallerPath(path string) string {

	idx := strings.LastIndexByte(path, '/')
	if idx == -1 {
		return path
	}

	idx = strings.LastIndexByte(path[:idx], '/')
	if idx == -1 {
		return path
	}

	return path[idx+1:]
}

var logImplFile = regexp.MustCompile(`.+newLogger.go|.+interceptlogger.go$`)

func (l *newLogger) logPlain(t time.Time, name string, level Level, msg string, args ...interface{}) {
	if len(l.timeFormat) > 0 {
		l.writer.WriteString(t.Format(l.timeFormat))
		l.writer.WriteByte(' ')
	}

	s, ok := _levelToBracket[level]
	if ok {
		l.writer.WriteString(s)
	} else {
		l.writer.WriteString("[?????]")
	}

	offset := 3
	if l.caller {
		if _, file, _, ok := runtime.Caller(3); ok {
			match := logImplFile.MatchString(file)
			if match {
				offset = 4
			}
		}

		if _, file, line, ok := runtime.Caller(offset); ok {
			l.writer.WriteByte(' ')
			l.writer.WriteString(trimCallerPath(file))
			l.writer.WriteByte(':')
			l.writer.WriteString(strconv.Itoa(line))
			l.writer.WriteByte(':')
		}
	}

	l.writer.WriteByte(' ')

	if name != "" {
		l.writer.WriteString(name)
		l.writer.WriteString(": ")
	}

	l.writer.WriteString(msg)

	args = append(l.implied, args...)

	var stacktrace CapturedStacktrace

	if args != nil && len(args) > 0 {
		if len(args)%2 != 0 {
			cs, ok := args[len(args)-1].(CapturedStacktrace)
			if ok {
				args = args[:len(args)-1]
				stacktrace = cs
			} else {
				extra := args[len(args)-1]
				args = append(args[:len(args)-1], MissingKey, extra)
			}
		}

		l.writer.WriteByte(':')

	FOR:
		for i := 0; i < len(args); i = i + 2 {
			var (
				val string
				raw bool
			)

			switch st := args[i+1].(type) {
			case string:
				val = st
			case int:
				val = strconv.FormatInt(int64(st), 10)
			case int64:
				val = strconv.FormatInt(int64(st), 10)
			case int32:
				val = strconv.FormatInt(int64(st), 10)
			case int16:
				val = strconv.FormatInt(int64(st), 10)
			case int8:
				val = strconv.FormatInt(int64(st), 10)
			case uint:
				val = strconv.FormatUint(uint64(st), 10)
			case uint64:
				val = strconv.FormatUint(uint64(st), 10)
			case uint32:
				val = strconv.FormatUint(uint64(st), 10)
			case uint16:
				val = strconv.FormatUint(uint64(st), 10)
			case uint8:
				val = strconv.FormatUint(uint64(st), 10)
			case Hex:
				val = "0x" + strconv.FormatUint(uint64(st), 16)
			case Octal:
				val = "0" + strconv.FormatUint(uint64(st), 8)
			case Binary:
				val = "0b" + strconv.FormatUint(uint64(st), 2)
			case CapturedStacktrace:
				stacktrace = st
				continue FOR
			case Format:
				val = fmt.Sprintf(st[0].(string), st[1:]...)
			default:
				v := reflect.ValueOf(st)
				if v.Kind() == reflect.Slice {
					val = l.renderSlice(v)
					raw = true
				} else {
					val = fmt.Sprintf("%v", st)
				}
			}

			l.writer.WriteByte(' ')
			switch st := args[i].(type) {
			case string:
				l.writer.WriteString(st)
			default:
				l.writer.WriteString(fmt.Sprintf("%s", st))
			}
			l.writer.WriteByte('=')

			if !raw && strings.ContainsAny(val, " \t\n\r") {
				l.writer.WriteByte('"')
				l.writer.WriteString(val)
				l.writer.WriteByte('"')
			} else {
				l.writer.WriteString(val)
			}
		}
	}

	l.writer.WriteString("\n")

	if stacktrace != "" {
		l.writer.WriteString(string(stacktrace))
	}
}

func (l *newLogger) renderSlice(v reflect.Value) string {
	var buf bytes.Buffer

	buf.WriteRune('[')

	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}

		sv := v.Index(i)

		var val string

		switch sv.Kind() {
		case reflect.String:
			val = sv.String()
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			val = strconv.FormatInt(sv.Int(), 10)
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val = strconv.FormatUint(sv.Uint(), 10)
		default:
			val = fmt.Sprintf("%v", sv.Interface())
		}

		if strings.ContainsAny(val, " \t\n\r") {
			buf.WriteByte('"')
			buf.WriteString(val)
			buf.WriteByte('"')
		} else {
			buf.WriteString(val)
		}
	}

	buf.WriteRune(']')

	return buf.String()
}

func (l *newLogger) logJSON(t time.Time, name string, level Level, msg string, args ...interface{}) {
	vals := l.jsonMapEntry(t, name, level, msg)
	args = append(l.implied, args...)

	if args != nil && len(args) > 0 {
		if len(args)%2 != 0 {
			cs, ok := args[len(args)-1].(CapturedStacktrace)
			if ok {
				args = args[:len(args)-1]
				vals["stacktrace"] = cs
			} else {
				extra := args[len(args)-1]
				args = append(args[:len(args)-1], MissingKey, extra)
			}
		}

		for i := 0; i < len(args); i = i + 2 {
			val := args[i+1]
			switch sv := val.(type) {
			case error:
				switch sv.(type) {
				case json.Marshaler, encoding.TextMarshaler:
				default:
					val = sv.Error()
				}
			case Format:
				val = fmt.Sprintf(sv[0].(string), sv[1:]...)
			}

			var key string

			switch st := args[i].(type) {
			case string:
				key = st
			default:
				key = fmt.Sprintf("%s", st)
			}
			vals[key] = val
		}
	}

	err := json.NewEncoder(l.writer).Encode(vals)
	if err != nil {
		if _, ok := err.(*json.UnsupportedTypeError); ok {
			plainVal := l.jsonMapEntry(t, name, level, msg)
			plainVal["@warn"] = errJsonUnsupportedTypeMsg

			json.NewEncoder(l.writer).Encode(plainVal)
		}
	}
}

func (l newLogger) jsonMapEntry(t time.Time, name string, level Level, msg string) map[string]interface{} {
	vals := map[string]interface{}{
		"@message":   msg,
		"@timestamp": t.Format("2006-01-02T15:04:05.000000Z07:00"),
	}

	var levelStr string
	switch level {
	case Error:
		levelStr = "error"
	case Warn:
		levelStr = "warn"
	case Info:
		levelStr = "info"
	case Debug:
		levelStr = "debug"
	case Trace:
		levelStr = "trace"
	default:
		levelStr = "all"
	}

	vals["@level"] = levelStr

	if name != "" {
		vals["@module"] = name
	}

	if l.caller {
		if _, file, line, ok := runtime.Caller(4); ok {
			vals["@caller"] = fmt.Sprintf("%s:%d", file, line)
		}
	}
	return vals
}

func (l *newLogger) Log(level Level, msg string, args ...interface{}) {
	l.log(l.Name(), level, msg, args...)
}

func (l *newLogger) Debug(msg string, args ...interface{}) {
	l.log(l.Name(), Debug, msg, args...)
}

func (l *newLogger) Trace(msg string, args ...interface{}) {
	l.log(l.Name(), Trace, msg, args...)
}

func (l *newLogger) Info(msg string, args ...interface{}) {
	l.log(l.Name(), Info, msg, args...)
}

func (l *newLogger) Warn(msg string, args ...interface{}) {
	l.log(l.Name(), Warn, msg, args...)
}

func (l *newLogger) Error(msg string, args ...interface{}) {
	l.log(l.Name(), Error, msg, args...)
}

func (l *newLogger) Panic(msg string, args ...interface{}) {
	l.log(l.Name(), Error, msg, args...)
	panic(msg)
}

func (l *newLogger) ErrorPanic(err error, args ...interface{}) {
	if err != nil {
		l.log(l.Name(), Error, err.Error(), args...)
		panic(err)
	}
}

func (l *newLogger) IsTrace() bool {
	return Level(atomic.LoadInt32(l.level)) == Trace
}

func (l *newLogger) IsDebug() bool {
	return Level(atomic.LoadInt32(l.level)) <= Debug
}

func (l *newLogger) IsInfo() bool {
	return Level(atomic.LoadInt32(l.level)) <= Info
}

func (l *newLogger) IsWarn() bool {
	return Level(atomic.LoadInt32(l.level)) <= Warn
}

func (l *newLogger) IsError() bool {
	return Level(atomic.LoadInt32(l.level)) <= Error
}

func (l *newLogger) Close() {
	l.ctx.Done()
}

func (l *newLogger) With(args ...interface{}) Logger {
	var extra interface{}

	if len(args)%2 != 0 {
		extra = args[len(args)-1]
		args = args[:len(args)-1]
	}

	sl := *l

	result := make(map[string]interface{}, len(l.implied)+len(args))
	keys := make([]string, 0, len(l.implied)+len(args))

	for i := 0; i < len(l.implied); i += 2 {
		key := l.implied[i].(string)
		keys = append(keys, key)
		result[key] = l.implied[i+1]
	}
	for i := 0; i < len(args); i += 2 {
		key := args[i].(string)
		_, exists := result[key]
		if !exists {
			keys = append(keys, key)
		}
		result[key] = args[i+1]
	}

	sort.Strings(keys)

	sl.implied = make([]interface{}, 0, len(l.implied)+len(args))
	for _, k := range keys {
		sl.implied = append(sl.implied, k)
		sl.implied = append(sl.implied, result[k])
	}

	if extra != nil {
		sl.implied = append(sl.implied, MissingKey, extra)
	}

	return &sl
}

func (l *newLogger) Named(name string) Logger {
	sl := *l

	if sl.name != "" {
		sl.name = sl.name + "." + name
	} else {
		sl.name = name
	}

	return &sl
}

func (l *newLogger) ResetNamed(name string) Logger {
	sl := *l

	sl.name = name

	return &sl
}

func (l *newLogger) ResetOutput(opts *LoggerOptions) error {
	if opts.Output == nil {
		return errors.New("given output is nil")
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.resetOutput(opts)
}

func (l *newLogger) ResetOutputWithFlush(opts *LoggerOptions, flushable Flushable) error {
	if opts.Output == nil {
		return errors.New("given output is nil")
	}
	if flushable == nil {
		return errors.New("flushable is nil")
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if err := flushable.Flush(); err != nil {
		return err
	}

	return l.resetOutput(opts)
}

func (l *newLogger) resetOutput(opts *LoggerOptions) error {
	l.writer = newWriter(opts.Output, opts.Color)
	l.setColorization(opts)
	return nil
}

func (l *newLogger) SetLevel(level Level) {
	atomic.StoreInt32(l.level, int32(level))
}

func (l *newLogger) checkWriterIsFile(wr io.Writer) *os.File {
	fi, ok := wr.(*os.File)
	if ok {
		return fi
	}
	panic("Cannot enable coloring of non-file Writers")
}

func (i *newLogger) Accept(name string, level Level, msg string, args ...interface{}) {
	i.log(name, level, msg, args...)
}

func (i *newLogger) ImpliedArgs() []interface{} {
	return i.implied
}

func (i *newLogger) Name() string {
	return i.name
}

func (l *newLogger) setColorization(opts *LoggerOptions) {
	for i, w := range l.writer.w {
		if runtime.GOOS == "windows" {
			switch opts.Color[i] {
			case ColorOff:
				return
			case ForceColor:
				fi := l.checkWriterIsFile(w)
				l.writer.w[i] = colorable.NewColorable(fi)
			case AutoColor:
				fi := l.checkWriterIsFile(w)
				isUnixTerm := isatty.IsTerminal(os.Stdout.Fd())
				isCygwinTerm := isatty.IsCygwinTerminal(os.Stdout.Fd())
				isTerm := isUnixTerm || isCygwinTerm
				if !isTerm {
					l.writer.color[i] = ColorOff
				}
				l.writer.w[i] = colorable.NewColorable(fi)
			}
		} else {
			switch opts.Color[i] {
			case ColorOff:
				fallthrough
			case ForceColor:
				return
			case AutoColor:
				fi := l.checkWriterIsFile(w)
				isUnixTerm := isatty.IsTerminal(fi.Fd())
				isCygwinTerm := isatty.IsCygwinTerminal(fi.Fd())
				isTerm := isUnixTerm || isCygwinTerm
				if !isTerm {
					l.writer.color[i] = ColorOff
				}
			}
		}
	}

}
