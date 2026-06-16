package logger

import "context"

type Entry struct {
	isOnError bool
	err       error
	fields    Fields
}

const errKey string = "err"

var initDone bool = false

var log Logger

func toFields(fields ...interface{}) Fields {
	if len(fields)%2 != 0 {
		return Fields{"oddFields": len(fields)}
	}
	logFields := make(Fields, len(fields)%2)
	for i := 0; i < len(fields); i = i + 2 {
		key := fields[i].(string)
		logFields[key] = fields[i+1]
	}
	return logFields
}

func Init(ctx context.Context, name string, level Level, dir string, keepNumDays int) {
	if initDone {
		return
	}
	log = NewDefaultLog(ctx, name, level, dir, keepNumDays)
	initDone = true
}

func NewLog() {
	if initDone {
		return
	}
	log = NewDefaultLog(context.Background(), "logger", Info, "./logs/", 10)
	initDone = true
}

func NewEntry() *Entry {
	NewLog()
	return &Entry{}
}

func OnError(err error) *Entry {
	e := NewEntry()
	return e.OnError(err)
}

func WithError(err error) *Entry {
	e := NewEntry()
	return e.WithError(err)
}

// WithFields creates a new entry without an id and the given fields
func WithFields(fields ...interface{}) *Entry {
	return NewEntry().SetFields(fields...)
}

// OnError sets the error. The log will only be printed if err is not nil
func (e *Entry) OnError(err error) *Entry {
	e.err = err
	e.isOnError = true
	return e.WithError(err)
}

// SetFields sets the given fields on the entry. It panics if length of fields is odd
func (e *Entry) SetFields(fields ...interface{}) *Entry {
	logFields := toFields(fields...)
	return e.WithFields(logFields)
}

func (e *Entry) WithField(key string, value interface{}) *Entry {
	if e.fields == nil {
		e.fields = make(Fields)
	}
	e.fields[key] = value
	return e
}

func (e *Entry) WithFields(fields Fields) *Entry {
	if e.fields == nil {
		e.fields = make(Fields)
	}
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

func (e *Entry) WithError(err error) *Entry {
	if e.fields == nil {
		e.fields = make(Fields)
	}
	e.fields[errKey] = err
	return e
}

func (e *Entry) logFunc(level Level, msg string, args ...interface{}) {
	if e.isOnError {
		if e.err == nil {
			return
		}
		level = Error
	}
	switch level {
	case Trace:
		log.Trace(msg, args...)
	case Debug:
		log.Debug(msg, args...)
	case Info:
		log.Info(msg, args...)
	case Warn:
		log.Warn(msg, args...)
	case Error:
		log.Error(msg, args...)
	}
}

func (e *Entry) expandFields(args ...interface{}) []interface{} {
	if e.fields == nil {
		return args
	}
	fields := toFields(args...)
	expanded := make([]interface{}, 0, len(fields)*2+len(e.fields)*2)
	for k, v := range e.fields {
		expanded = append(expanded, k, v)
	}
	for k, v := range fields {
		expanded = append(expanded, k, v)
	}
	return expanded
}

func (e *Entry) Trace(msg string, args ...interface{}) {
	e.logFunc(Trace, msg, e.expandFields(args...)...)
}

func (e *Entry) Debug(msg string, args ...interface{}) {
	e.logFunc(Debug, msg, e.expandFields(args...)...)
}

func (e *Entry) Info(msg string, args ...interface{}) {
	e.logFunc(Info, msg, e.expandFields(args...)...)
}

func (e *Entry) Warn(msg string, args ...interface{}) {
	e.logFunc(Warn, msg, e.expandFields(args...)...)
}

func (e *Entry) Error(msg string, args ...interface{}) {
	e.logFunc(Error, msg, e.expandFields(args...)...)
}

func (e *Entry) Panic(msg string, args ...interface{}) {
	e.logFunc(Error, msg, e.expandFields(args...)...)
	if e.isOnError {
		if e.err == nil {
			return
		}
	}
	panic(msg)
}

func (e *Entry) Fatal(msg string, args ...interface{}) {
	e.logFunc(Error, msg, e.expandFields(args...)...)
	if e.isOnError {
		if e.err == nil {
			return
		}
	}
	panic(msg)
}
