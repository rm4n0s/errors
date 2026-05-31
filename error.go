package errors

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"runtime"
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

// Error is an object to carry the original error or create a new one, but always with a specific format of stack trace that can be easily filtered
type Error struct {
	Tag         string
	Message     string
	Metadata    map[string]any
	OriginalErr error
	pcs         []uintptr // max 32 frames
}

type ErrorJson struct {
	Tag         string
	Message     string
	OriginalErr string
	Metadata    string
}

func New(tag, message string, metadata ...any) *Error {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	md := argsToMap(metadata...)
	return &Error{Tag: tag, Message: message, pcs: pcs[:n], Metadata: md}
}

func NewFromErr(tag, message string, err error, metadata ...any) *Error {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	md := argsToMap(metadata...)

	return &Error{Tag: tag, Message: err.Error(), OriginalErr: err, Metadata: md, pcs: pcs[:n]}
}

func NewFromJson(errj ErrorJson) (*Error, error) {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	e := &Error{Tag: errj.Tag, Message: errj.Message, pcs: pcs[:n]}
	if len(errj.Metadata) > 0 {
		gobBytes, err := base64.StdEncoding.DecodeString(errj.Metadata)
		if err != nil {
			return nil, NewFromErr("FailedBase64DecodeMetadata", "failed to decode error's metadata from Base64", err)
		}

		if err := gob.NewDecoder(bytes.NewReader(gobBytes)).Decode(&e.Metadata); err != nil {
			return nil, NewFromErr("FailedGobDecodeMetadata", "failed to decode error's metadata from GOB", err)
		}
	}
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

func (e *Error) ToJson() (*ErrorJson, error) {
	errj := ErrorJson{
		Tag:     e.Tag,
		Message: e.Message,
	}
	if e.OriginalErr != nil {
		errj.OriginalErr = e.OriginalErr.Error()
	}

	if len(e.Metadata) > 0 {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(e.Metadata); err != nil {
			return nil, NewFromErr("FailedGobEncodeMetadata", "failed to encode error's metadata to GOB", err)
		}

		errj.Metadata = base64.StdEncoding.EncodeToString(buf.Bytes())
	}

	return &errj, nil
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Tag == t.Tag
}
