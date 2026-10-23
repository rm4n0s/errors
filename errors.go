package errors

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strings"
)

var (
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
)

type StackFrame struct {
	FuncName    string
	StructName  string
	PackageName string
}

type Error struct {
	Tag         string
	Message     string
	Metadata    map[string]any
	OriginalErr error
	pcs         []uintptr    // max 32 frames
	resolved    []StackFrame // set instead of pcs when rebuilt via FromJson()
}

type ErrorJson struct {
	Tag         string
	Message     string
	OriginalErr string
	Metadata    map[string]any
	StackFrames []StackFrame
}

func New(tag, message string, metadata ...any) *Error {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	md := argsToMap(metadata...)
	return &Error{Tag: tag, Message: message, pcs: pcs[:n], Metadata: md}
}

func NewErr(tag string, err error) *Error {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	return &Error{Tag: tag, Message: err.Error(), pcs: pcs[:n], OriginalErr: err}
}

func NewErrf(tag, format string, a ...any) *Error {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	originalErr := fmt.Errorf(format, a...)
	return &Error{Tag: tag, Message: originalErr.Error(), pcs: pcs[:n], OriginalErr: originalErr}
}

func FromError(err error) (*Error, bool) {
	res, ok := err.(*Error)
	if !ok {
		return nil, false
	}
	return res, true
}

func FromJson(errj ErrorJson) *Error {
	e := &Error{Tag: errj.Tag, Message: errj.Message,
		resolved: errj.StackFrames, Metadata: errj.Metadata}
	if len(errj.OriginalErr) > 0 {
		e.OriginalErr = errors.New(errj.OriginalErr)
	}
	return e
}

func (err *Error) Error() string {
	return err.Message
}

func (err *Error) Unwrap() error {
	return err.OriginalErr
}

func (err *Error) SetMetadata(metadata ...any) *Error {
	if err.Metadata == nil {
		err.Metadata = make(map[string]any)
	}
	maps.Copy(err.Metadata, argsToMap(metadata...))
	return err
}

func (err *Error) RemoveMetadata(keys ...string) *Error {
	for _, k := range keys {
		delete(err.Metadata, k)
	}
	return err
}

func (err *Error) SetMessage(msg string) *Error {
	err.Message = msg
	return err
}

func (e *Error) StackFrames(all bool) []StackFrame {
	return e.stackFrames(getCallerFrame(), all)
}

// stackFrames returns the frames from where the error was created up to, but
// not including, the caller frame. The creation frame is always kept.
func (e *Error) stackFrames(caller StackFrame, all bool) []StackFrame {
	if e.resolved != nil {
		return slices.Clone(e.resolved)
	}

	sfs := make([]StackFrame, 0, len(e.pcs))
	if len(e.pcs) == 0 {
		return sfs
	}
	frames := runtime.CallersFrames(e.pcs)
	for {
		f, more := frames.Next()
		sf := frameFromRuntime(f)
		if !all && sf == caller {
			if len(sfs) == 0 {
				sfs = append(sfs, sf)
			}
			break
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
	if e.resolved != nil {
		for _, f := range e.resolved {
			fmt.Fprintf(&sb, "%s.%s\n", f.PackageName, f.qualifiedName())
		}
		return sb.String()
	}
	if len(e.pcs) == 0 {
		return ""
	}
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

func (e *Error) ToJson() *ErrorJson {
	return e.toJson(getCallerFrame())
}

func (e *Error) toJson(caller StackFrame) *ErrorJson {
	errj := ErrorJson{
		Tag:     e.Tag,
		Message: e.Message,
	}
	if e.OriginalErr != nil {
		errj.OriginalErr = e.OriginalErr.Error()
	}

	errj.StackFrames = e.stackFrames(caller, false)
	errj.Metadata = maps.Clone(e.Metadata)
	return &errj
}

// LogValue implements slog.LogValuer. The keys match the ErrorJson fields, so a
// logged error can be unmarshaled into ErrorJson and replayed with FromJson.
func (e *Error) LogValue() slog.Value {
	errj := e.toJson(getCallerFrameOutsideSlog())
	attrs := []slog.Attr{
		slog.String("Tag", errj.Tag),
		slog.String("Message", errj.Message),
		slog.String("Route", errj.Route()),
	}
	if errj.OriginalErr != "" {
		attrs = append(attrs, slog.String("OriginalErr", errj.OriginalErr))
	}
	if len(errj.Metadata) > 0 {
		attrs = append(attrs, slog.Any("Metadata", errj.Metadata))
	}
	attrs = append(attrs, slog.Any("StackFrames", errj.StackFrames))
	return slog.GroupValue(attrs...)
}

// Format implements fmt.Formatter. %+v prints the message followed by the stack trace.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%s\n%s", e.Message, e.StackTrace())
			return
		}
		if s.Flag('#') {
			type plain Error
			fmt.Fprintf(s, "%#v", (*plain)(e))
			return
		}
		io.WriteString(s, e.Message)
	case 's':
		io.WriteString(s, e.Message)
	case 'q':
		fmt.Fprintf(s, "%q", e.Message)
	}
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Tag == t.Tag
}

func (e *Error) Route() string {
	return buildRoute(e.stackFrames(getCallerFrame(), false), e.Tag)
}

func (e *Error) HasRoute(route string) bool {
	return strings.Contains(buildRoute(e.stackFrames(getCallerFrame(), false), e.Tag), route)
}

func (e *ErrorJson) Route() string {
	return buildRoute(e.StackFrames, e.Tag)
}

func (e *ErrorJson) HasRoute(route string) bool {
	return strings.Contains(e.Route(), route)
}
