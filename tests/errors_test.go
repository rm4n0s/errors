package tests

import (
	stdErrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// ---------------------------------------------------------------------
// New
// ---------------------------------------------------------------------

func TestNew_SetsTagAndMessage(t *testing.T) {
	e := errors.New("SomeTag", "something went wrong")

	if e.Tag != "SomeTag" {
		t.Errorf("Tag = %q, want %q", e.Tag, "SomeTag")
	}
	if e.Message != "something went wrong" {
		t.Errorf("Message = %q, want %q", e.Message, "something went wrong")
	}
}

func TestNew_NoMetadataYieldsEmptyMap(t *testing.T) {
	e := errors.New("Tag", "msg")
	if len(e.Metadata) != 0 {
		t.Errorf("Metadata = %v, want empty", e.Metadata)
	}
}

// NOTE: this assumes argsToMap treats the variadic metadata as alternating
// key/value pairs (a common convention for this kind of helper). If your
// actual argsToMap implementation uses a different convention, adjust the
// assertions below accordingly.
func TestNew_WithMetadataPairs(t *testing.T) {
	e := errors.New("Tag", "msg", "userID", 42, "action", "delete")

	if got, want := e.Metadata["userID"], any(42); got != want {
		t.Errorf("Metadata[userID] = %v, want %v", got, want)
	}
	if got, want := e.Metadata["action"], "delete"; got != want {
		t.Errorf("Metadata[action] = %v, want %v", got, want)
	}
}

func TestNew_CapturesCallStack(t *testing.T) {
	e := errors.New("Tag", "msg")

	trace := e.StackTrace()
	if trace == "" {
		t.Fatal("StackTrace() returned empty string, expected captured frames")
	}
	if !strings.Contains(trace, "TestNew_CapturesCallStack") {
		t.Errorf("StackTrace() = %q, want it to contain the calling test function", trace)
	}
}

// ---------------------------------------------------------------------
// FromError
// ---------------------------------------------------------------------

func TestFromError_Success(t *testing.T) {
	original := errors.New("Tag", "msg")
	var err error = original

	got, ok := errors.FromError(err)
	if !ok {
		t.Fatal("FromError() ok = false, want true")
	}
	if got != original {
		t.Errorf("FromError() returned %p, want %p", got, original)
	}
}

func TestFromError_NonErrorType(t *testing.T) {
	stdErr := stdErrors.New("plain error")

	got, ok := errors.FromError(stdErr)
	if ok {
		t.Fatal("FromError() ok = true, want false for a non-*Error")
	}
	if got != nil {
		t.Errorf("FromError() = %v, want nil", got)
	}
}

func TestFromError_NilError(t *testing.T) {
	got, ok := errors.FromError(nil)
	if ok {
		t.Fatal("FromError(nil) ok = true, want false")
	}
	if got != nil {
		t.Errorf("FromError(nil) = %v, want nil", got)
	}
}

func TestFromError_WrappedErrorNotUnwrapped(t *testing.T) {
	// FromError performs a plain type assertion, not errors.As, so a
	// *Error hidden behind fmt.Errorf's %w should NOT be recognized.
	inner := errors.New("Tag", "msg")
	wrapped := fmt.Errorf("context: %w", inner)

	_, ok := errors.FromError(wrapped)
	if ok {
		t.Error("FromError() ok = true for a wrapped error, want false (no unwrapping)")
	}
}

// ---------------------------------------------------------------------
// FromJson
// ---------------------------------------------------------------------

func TestFromJson_CopiesTagAndMessage(t *testing.T) {
	e, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M"})
	if err != nil {
		t.Fatalf("FromJson() error = %v", err)
	}
	if e.Tag != "T" || e.Message != "M" {
		t.Errorf("got Tag=%q Message=%q, want Tag=%q Message=%q", e.Tag, e.Message, "T", "M")
	}
}

func TestFromJson_SetsOriginalErr(t *testing.T) {
	e, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M", OriginalErr: "boom"})
	if err != nil {
		t.Fatalf("FromJson() error = %v", err)
	}
	if e.OriginalErr == nil {
		t.Fatal("OriginalErr = nil, want an error")
	}
	if e.OriginalErr.Error() != "boom" {
		t.Errorf("OriginalErr.Error() = %q, want %q", e.OriginalErr.Error(), "boom")
	}
}

func TestFromJson_NoOriginalErr(t *testing.T) {
	e, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M"})
	if err != nil {
		t.Fatalf("FromJson() error = %v", err)
	}
	if e.OriginalErr != nil {
		t.Errorf("OriginalErr = %v, want nil", e.OriginalErr)
	}
}

func TestFromJson_EmptyMetadataIsSkipped(t *testing.T) {
	e, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M"})
	if err != nil {
		t.Fatalf("FromJson() error = %v", err)
	}
	if len(e.Metadata) != 0 {
		t.Errorf("Metadata = %v, want empty", e.Metadata)
	}
}

func TestFromJson_CapturesCallStack(t *testing.T) {
	e, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M"})
	if err != nil {
		t.Fatalf("FromJson() error = %v", err)
	}
	if e.StackTrace() == "" {
		t.Error("StackTrace() = empty, want captured frames")
	}
}

// ---------------------------------------------------------------------
// Error() / Unwrap()
// ---------------------------------------------------------------------

func TestError_ErrorReturnsMessage(t *testing.T) {
	e := &errors.Error{Message: "boom"}
	if e.Error() != "boom" {
		t.Errorf("Error() = %q, want %q", e.Error(), "boom")
	}
}

func TestError_UnwrapReturnsOriginalErr(t *testing.T) {
	inner := stdErrors.New("inner")
	e := &errors.Error{Message: "outer", OriginalErr: inner}

	if e.Unwrap() != inner {
		t.Errorf("Unwrap() = %v, want %v", e.Unwrap(), inner)
	}
}

func TestError_UnwrapNilWhenNoOriginalErr(t *testing.T) {
	e := &errors.Error{Message: "outer"}
	if e.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", e.Unwrap())
	}
}

func TestError_CompatibleWithStdlibUnwrapChain(t *testing.T) {
	inner := stdErrors.New("root cause")
	e := &errors.Error{Tag: "T", Message: "outer", OriginalErr: inner}

	if !errors.Is(e, inner) {
		t.Error("errors.Is(e, inner) = false, want true via the Unwrap chain")
	}
	if got := errors.Unwrap(error(e)); got != inner {
		t.Errorf("errors.Unwrap(e) = %v, want %v", got, inner)
	}
}

// ---------------------------------------------------------------------
// Is()
// ---------------------------------------------------------------------

func TestError_IsMatchesOnTag(t *testing.T) {
	a := &errors.Error{Tag: "NotFound", Message: "first"}
	b := &errors.Error{Tag: "NotFound", Message: "second, different message/instance"}

	if !a.Is(b) {
		t.Error("a.Is(b) = false, want true (same Tag)")
	}
}

func TestError_IsFailsOnDifferentTag(t *testing.T) {
	a := &errors.Error{Tag: "NotFound", Message: "m"}
	b := &errors.Error{Tag: "Unauthorized", Message: "m"}

	if a.Is(b) {
		t.Error("a.Is(b) = true, want false (different Tag)")
	}
}

func TestError_IsFailsOnNonErrorTarget(t *testing.T) {
	a := &errors.Error{Tag: "NotFound", Message: "m"}
	if a.Is(stdErrors.New("plain")) {
		t.Error("a.Is(plain error) = true, want false")
	}
}

func TestPackageIs_DispatchesToCustomIs(t *testing.T) {
	sentinel := &errors.Error{Tag: "NotFound", Message: "sentinel"}
	occurrence := &errors.Error{Tag: "NotFound", Message: "occurred while doing X"}

	if !errors.Is(occurrence, sentinel) {
		t.Error("Is(occurrence, sentinel) = false, want true (Tag-based match)")
	}

	other := &errors.Error{Tag: "Unauthorized", Message: "occurred while doing X"}
	if errors.Is(other, sentinel) {
		t.Error("Is(other, sentinel) = true, want false (different Tag)")
	}
}

func TestPackageAs_FindsErrorInChain(t *testing.T) {
	original := &errors.Error{Tag: "T", Message: "m"}
	wrapped := fmt.Errorf("context: %w", original)

	var target *errors.Error
	if !errors.As(wrapped, &target) {
		t.Fatal("As() = false, want true")
	}
	if target != original {
		t.Errorf("As() target = %p, want %p", target, original)
	}
}

// ---------------------------------------------------------------------
// StackFrames() / StackTrace()
// ---------------------------------------------------------------------

func TestError_StackFrames_AllTrueReturnsFrames(t *testing.T) {
	e := errors.New("T", "m")

	frames := e.StackFrames(true)
	if len(frames) == 0 {
		t.Fatal("StackFrames(true) returned no frames")
	}
	if len(frames) > 32 {
		t.Errorf("StackFrames(true) returned %d frames, want <= 32", len(frames))
	}
}

func TestError_StackFrames_AllFalseNeverExceedsAllTrue(t *testing.T) {
	e := errors.New("T", "m")

	all := e.StackFrames(true)
	trimmed := e.StackFrames(false)

	if len(trimmed) > len(all) {
		t.Errorf("StackFrames(false) returned %d frames, more than StackFrames(true)'s %d", len(trimmed), len(all))
	}
}

func TestError_StackTrace_ContainsCallerAndLocation(t *testing.T) {
	e := errors.New("T", "m")
	trace := e.StackTrace()

	if !strings.Contains(trace, "TestError_StackTrace_ContainsCallerAndLocation") {
		t.Errorf("StackTrace() = %q, want it to mention the calling test", trace)
	}
	if !strings.Contains(trace, "errors_test.go") {
		t.Errorf("StackTrace() = %q, want it to reference errors_test.go", trace)
	}
}

// ---------------------------------------------------------------------
// FromJson
// ---------------------------------------------------------------------

func TestFromJson_InvalidJsonMetadata(t *testing.T) {
	_, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M", Metadata: "not valid json at all"})
	if err == nil {
		t.Fatal("FromJson() error = nil, want an error for invalid JSON")
	}
	decErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if decErr.Tag != "FailedJsonDecodeMetadata" {
		t.Errorf("Tag = %q, want %q", decErr.Tag, "FailedJsonDecodeMetadata")
	}
}

func TestFromJson_WrongShapedJsonMetadata(t *testing.T) {
	// Valid JSON, but an array can't unmarshal into map[string]any.
	_, err := errors.FromJson(errors.ErrorJson{Tag: "T", Message: "M", Metadata: `["a","b"]`})
	if err == nil {
		t.Fatal("FromJson() error = nil, want an error for a JSON array where an object was expected")
	}
	decErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if decErr.Tag != "FailedJsonDecodeMetadata" {
		t.Errorf("Tag = %q, want %q", decErr.Tag, "FailedJsonDecodeMetadata")
	}
}

// ---------------------------------------------------------------------
// Route() / HasRoute() -- *Error
// ---------------------------------------------------------------------

func makeNestedError(tag, msg string) *errors.Error {
	return errors.New(tag, msg)
}

func TestError_Route_EndsWithTag(t *testing.T) {
	e := makeNestedError("Boom", "m")

	route := e.Route()
	if route == "" {
		t.Fatal("Route() returned empty string")
	}
	if !strings.HasSuffix(route, ".Boom") {
		t.Errorf("Route() = %q, want it to end with %q", route, ".Boom")
	}
}

func TestError_HasRoute_MatchesSubstring(t *testing.T) {
	e := makeNestedError("Boom", "m")
	route := e.Route()

	if !e.HasRoute(route) {
		t.Error("HasRoute(full route) = false, want true")
	}
	if e.HasRoute("DefinitelyNotInTheRoute") {
		t.Error("HasRoute(unrelated string) = true, want false")
	}
}

// ---------------------------------------------------------------------
// Route() / HasRoute() -- ErrorJson
//
// ErrorJson.StackFrames is a plain exported field, so these are fully
// deterministic: no dependency on runtime stack capture at all.
// ---------------------------------------------------------------------

func TestErrorJson_Route_BuildsExpectedString(t *testing.T) {
	ej := errors.ErrorJson{
		Tag: "MyTag",
		StackFrames: []errors.StackFrame{
			{FuncName: "First", PackageName: "pkg1"},
			{FuncName: "Second", StructName: "MyStruct", PackageName: "pkg2"},
			{FuncName: "Third", PackageName: "pkg3"},
		},
	}

	want := "Third->MyStruct.Second->First.MyTag"
	if got := ej.Route(); got != want {
		t.Errorf("Route() = %q, want %q", got, want)
	}
}

func TestErrorJson_Route_SingleFrame(t *testing.T) {
	ej := errors.ErrorJson{
		Tag:         "OnlyTag",
		StackFrames: []errors.StackFrame{{FuncName: "Solo", PackageName: "pkg"}},
	}

	want := "Solo.OnlyTag"
	if got := ej.Route(); got != want {
		t.Errorf("Route() = %q, want %q", got, want)
	}
}

func TestErrorJson_Route_EmptyFrames(t *testing.T) {
	ej := errors.ErrorJson{Tag: "T", StackFrames: []errors.StackFrame{}}

	if got := ej.Route(); got != "" {
		t.Errorf("Route() = %q, want empty string for no frames", got)
	}
}

func TestErrorJson_HasRoute(t *testing.T) {
	ej := errors.ErrorJson{
		Tag: "MyTag",
		StackFrames: []errors.StackFrame{
			{FuncName: "First", PackageName: "pkg1"},
			{FuncName: "Second", StructName: "MyStruct", PackageName: "pkg2"},
		},
	}

	if !ej.HasRoute("MyStruct.Second") {
		t.Error(`HasRoute("MyStruct.Second") = false, want true`)
	}
	if ej.HasRoute("MyStruct.Second.MyTag") {
		t.Error(`HasRoute("MyStruct.Second.MyTag") = true, want false`)
	}
	if ej.HasRoute("NotPresent") {
		t.Error(`HasRoute("NotPresent") = true, want false`)
	}
}
