package logger

import (
	"bytes"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	_stacktraceIgnorePrefixes = []string{
		"runtime.goexit",
		"runtime.main",
	}
	_stacktracePool = sync.Pool{
		New: func() interface{} {
			return newProgramCounters(64)
		},
	}
)

type CapturedStacktrace string

func Stacktrace() CapturedStacktrace {
	return CapturedStacktrace(takeStacktrace())
}

func takeStacktrace() string {
	programCounters := _stacktracePool.Get().(*programCounters)
	defer _stacktracePool.Put(programCounters)

	var buffer bytes.Buffer

	for {

		n := runtime.Callers(2, programCounters.pcs)
		if n < cap(programCounters.pcs) {
			programCounters.pcs = programCounters.pcs[:n]
			break
		}
		programCounters = newProgramCounters(len(programCounters.pcs) * 2)
	}

	i := 0
	frames := runtime.CallersFrames(programCounters.pcs)
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		if shouldIgnoreStacktraceFunction(frame.Function) {
			continue
		}
		if i != 0 {
			buffer.WriteByte('\n')
		}
		i++
		buffer.WriteString(frame.Function)
		buffer.WriteByte('\n')
		buffer.WriteByte('\t')
		buffer.WriteString(frame.File)
		buffer.WriteByte(':')
		buffer.WriteString(strconv.Itoa(int(frame.Line)))
	}

	return buffer.String()
}

func shouldIgnoreStacktraceFunction(function string) bool {
	for _, prefix := range _stacktraceIgnorePrefixes {
		if strings.HasPrefix(function, prefix) {
			return true
		}
	}
	return false
}

type programCounters struct {
	pcs []uintptr
}

func newProgramCounters(size int) *programCounters {
	return &programCounters{make([]uintptr, size)}
}
