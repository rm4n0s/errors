package errors

import (
	"errors"
	"fmt"
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

func FromJson(errj ErrorJson) (*Error, error) {
	e := &Error{Tag: errj.Tag, Message: errj.Message,
		resolved: errj.StackFrames, Metadata: errj.Metadata}
	if len(errj.OriginalErr) > 0 {
		e.OriginalErr = errors.New(errj.OriginalErr)
	}
	return e, nil
}

func (err *Error) Error() string {
	return err.Message
}

func (err *Error) Unwrap() error {
	return err.OriginalErr
}

func (err *Error) SetMetadata(metadata ...any) *Error {
	err.Metadata = argsToMap(metadata...)
	return err
}

func (err *Error) SetMessage(msg string) *Error {
	err.Message = msg
	return err
}

func (e *Error) StackFrames(all bool) []StackFrame {
	if e.resolved != nil {
		return slices.Clone(e.resolved)
	}

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
	if e.resolved != nil {
		for _, f := range e.resolved {
			fn := f.FuncName
			if f.StructName != "" {
				fn = fmt.Sprintf("%s.%s", f.StructName, f.FuncName)
			}
			fmt.Fprintf(&sb, "%s.%s\n", f.PackageName, fn)
		}
		return sb.String()
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

func (e *Error) ToJson() (*ErrorJson, error) {
	errj := ErrorJson{
		Tag:     e.Tag,
		Message: e.Message,
	}
	if e.OriginalErr != nil {
		errj.OriginalErr = e.OriginalErr.Error()
	}

	errj.StackFrames = e.StackFrames(false)
	errj.Metadata = e.Metadata
	return &errj, nil

}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Tag == t.Tag
}

func (e *Error) Route() string {
	route := ""
	frames := e.StackFrames(false)
	slices.Reverse(frames)
	for i, v := range frames {
		fn := v.FuncName
		if v.StructName != "" {
			fn = fmt.Sprintf("%s.%s", v.StructName, v.FuncName)
		}
		if len(frames)-1 == i {
			fn += "." + e.Tag
		}
		if i == 0 {
			route = fn
		} else {
			route += fn
		}

		if len(frames)-1 > i {
			route += "->"
		}
	}
	return route

}

func (e *Error) HasRoute(route string) bool {
	return strings.Contains(e.Route(), route)
}

func (e *ErrorJson) Route() string {
	route := ""
	frames := slices.Clone(e.StackFrames)
	slices.Reverse(frames)
	for i, v := range frames {
		fn := v.FuncName
		if v.StructName != "" {
			fn = fmt.Sprintf("%s.%s", v.StructName, v.FuncName)
		}
		if len(frames)-1 == i {
			fn += "." + e.Tag
		}
		if i == 0 {
			route = fn
		} else {
			route += fn
		}

		if len(frames)-1 > i {
			route += "->"
		}
	}
	return route
}

func (e *ErrorJson) HasRoute(route string) bool {
	return strings.Contains(e.Route(), route)
}
