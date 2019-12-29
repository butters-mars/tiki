package logging

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// L the global logger
var L = logrus.New()

func init() {
	L.AddHook(&ContextHook{})
	L.Level = logrus.DebugLevel
}

type logType int

const (
	trace logType = iota
	debug
	info
	warn
	err
	fatal
)

// WTrace trace log with fields
func WTrace(msg string, kvs ...interface{}) {
	withFields(trace, msg, kvs...)
}

// WDebug debug log with fields
func WDebug(msg string, kvs ...interface{}) {
	withFields(debug, msg, kvs...)
}

// WInfo info log with fields
func WInfo(msg string, kvs ...interface{}) {
	withFields(info, msg, kvs...)
}

// WWarn warn log with fields
func WWarn(msg string, kvs ...interface{}) {
	withFields(warn, msg, kvs...)
}

// WError error log with fields
func WError(msg string, kvs ...interface{}) {
	withFields(err, msg, kvs...)
}

// WFatal fatal log with fields
func WFatal(msg string, kvs ...interface{}) {
	withFields(fatal, msg, kvs...)
}

func withFields(lt logType, msg string, kvs ...interface{}) {
	fs := logrus.Fields{}
	for i := 0; i < len(kvs)/2; i++ {
		k := kvs[2*i]
		v := kvs[2*i+1]
		_k := fmt.Sprintf("%v", k)
		fs[_k] = v
	}

	wf := L.WithFields(fs)

	switch lt {
	case trace:
		wf.Trace(msg)
	case debug:
		wf.Debug(msg)
	case info:
		wf.Info(msg)
	case warn:
		wf.Warn(msg)
	case err:
		wf.Error(msg)
	case fatal:
		wf.Fatal(msg)
	}

}
