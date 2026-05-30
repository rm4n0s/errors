package errors

import (
	"fmt"
	"runtime"
	"strings"
)

type StackFrame struct {
	FuncName    string
	StructName  string
	PackageName string
}

// Error is an object to carry the original error or create a new one, but always with a specific format of stack trace that can be easily filtered
type Error struct {
	Tag         string
	Message     string
	Metadata    map[string]any
	OriginalErr error
	pcs         []uintptr
}

func New(tag, message string) *Error {
	pcs := make([]uintptr, 32) // max 32 frames
	n := runtime.Callers(2, pcs)
	return &Error{Tag: tag, Message: message, pcs: pcs[:n]}
}

func Newf(tag string, format string, a ...any) *Error {
	err := fmt.Errorf(format, a...)
	pcs := make([]uintptr, 32) // max 32 frames
	n := runtime.Callers(2, pcs)
	return &Error{Tag: tag, Message: err.Error(), OriginalErr: err, pcs: pcs[:n]}
}

func NewWithMetadata(tag, message string, metadata ...any) *Error {
	pcs := make([]uintptr, 32) // max 32 frames
	n := runtime.Callers(2, pcs)
	err := &Error{Tag: tag, Message: message, pcs: pcs[:n]}
	err.Metadata = argsToMap(metadata...)
	return err
}

func NewFromErr(tag string, err error) *Error {
	pcs := make([]uintptr, 32) // max 32 frames
	n := runtime.Callers(2, pcs)
	return &Error{Tag: tag, Message: err.Error(), OriginalErr: err, pcs: pcs[:n]}
}

// Error returns the underlying error's message.
func (err *Error) Error() string {
	return err.Message
}

func (e *Error) StackFrames(all bool) []StackFrame {
	callerFrame := getCallerFrame()
	sfs := make([]StackFrame, 0)
	frames := runtime.CallersFrames(e.pcs)
	for {
		f, more := frames.Next()
		sf := frameFromRuntime(f)
		if !all {
			if sf.FuncName == callerFrame.FuncName &&
				sf.StructName == callerFrame.StructName &&
				sf.PackageName == callerFrame.PackageName {
				break
			}
		}
		sfs = append(sfs, sf)
		if !more {
			break
		}
	}
	return sfs
}

func (e *Error) StackTrace() string {
	var sb strings.Builder
	frames := runtime.CallersFrames(e.pcs)
	for {
		f, more := frames.Next()
		fmt.Fprintf(&sb, "%s\n\t%s:%d\n", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return sb.String()
}
