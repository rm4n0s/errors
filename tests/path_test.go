package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/errors/tests/crash2"
)

func TestErrorRoute_Filepath(t *testing.T) {
	obj := crash2.Crash2Struct{}
	err := obj.Crash2()
	e, ok := errors.FromError(err)
	if !ok {
		t.Errorf("error wasn't correct")
	}

	if !e.HasRoute("Crash2Struct.Crash2->Crash1.Crash1Tag") {
		t.Errorf("route not correct")
	}
}

func TestHasRoute_TrimmedLikeRoute(t *testing.T) {
	obj := crash2.Crash2Struct{}
	e, _ := errors.FromError(obj.Crash2())

	if got, want := e.Route(), "Crash2Struct.Crash2->Crash1.Crash1Tag"; got != want {
		t.Fatalf("Route() = %q, want %q", got, want)
	}
	if e.HasRoute("TestHasRoute_TrimmedLikeRoute->Crash2Struct.Crash2") {
		t.Error("HasRoute() matched a frame above its caller, want it trimmed like Route()")
	}
}

func TestStackFrames_TrimmedLikeRoute(t *testing.T) {
	obj := crash2.Crash2Struct{}
	e, _ := errors.FromError(obj.Crash2())

	frames := e.StackFrames(false)
	if len(frames) != 2 || frames[0].FuncName != "Crash1" || frames[1].FuncName != "Crash2" {
		t.Errorf("StackFrames(false) = %+v, want [Crash1 Crash2]", frames)
	}
}

func createAndInspect() (string, *errors.ErrorJson) {
	e := errors.New("Local", "created and inspected in the same function")
	return e.Route(), e.ToJson()
}

func TestRoute_InFunctionThatCreatedTheError(t *testing.T) {
	route, errj := createAndInspect()

	if want := "createAndInspect.Local"; route != want {
		t.Errorf("Route() = %q, want %q", route, want)
	}
	if len(errj.StackFrames) != 1 || errj.StackFrames[0].FuncName != "createAndInspect" {
		t.Errorf("ToJson().StackFrames = %+v, want [createAndInspect]", errj.StackFrames)
	}
}

func TestRoute_ErrorWithoutStack(t *testing.T) {
	sentinel := &errors.Error{Tag: "NotFound"}

	if got := sentinel.Route(); got != "" {
		t.Errorf("Route() = %q, want empty", got)
	}
	if got := sentinel.StackFrames(true); len(got) != 0 {
		t.Errorf("StackFrames(true) = %+v, want empty", got)
	}
	if got := sentinel.StackTrace(); got != "" {
		t.Errorf("StackTrace() = %q, want empty", got)
	}
}

func TestToJson_CopiesMetadata(t *testing.T) {
	e := errors.New("T", "m", "k", 1)

	errj := e.ToJson()
	errj.Metadata["k"] = 2
	errj.Metadata["added"] = true

	if e.Metadata["k"] != 1 || len(e.Metadata) != 1 {
		t.Errorf("Metadata = %v, want map[k:1] (unchanged by edits to ToJson result)", e.Metadata)
	}
}

func logCrash(logger *slog.Logger) {
	obj := crash2.Crash2Struct{}
	err := obj.Crash2()
	if e, ok := errors.FromError(err); ok {
		e.SetMetadata("id", 7)
	}
	logger.Error("crash failed", "err", err)
}

func TestLogValue_ReplaysAsErrorJson(t *testing.T) {
	var buf bytes.Buffer
	logCrash(slog.New(slog.NewJSONHandler(&buf, nil)))

	var entry struct {
		Err struct {
			errors.ErrorJson
			Route string
		} `json:"err"`
	}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", buf.String(), err)
	}

	want := "Crash2Struct.Crash2->Crash1.Crash1Tag"
	if entry.Err.Route != want {
		t.Errorf("logged Route = %q, want %q", entry.Err.Route, want)
	}
	if got := entry.Err.ErrorJson.Route(); got != want {
		t.Errorf("ErrorJson.Route() = %q, want %q", got, want)
	}
	if entry.Err.Tag != "Crash1Tag" || entry.Err.Metadata["id"] != float64(7) {
		t.Errorf("logged ErrorJson = %+v", entry.Err.ErrorJson)
	}

	replayed := errors.FromJson(entry.Err.ErrorJson)
	if !replayed.HasRoute(want) {
		t.Errorf("FromJson().Route() = %q, want %q", replayed.Route(), want)
	}
}

func TestFormat(t *testing.T) {
	e := errors.New("T", "boom")

	if got := fmt.Sprintf("%v|%s|%q", e, e, e); got != `boom|boom|"boom"` {
		t.Errorf("%%v|%%s|%%q = %q", got)
	}
	plus := fmt.Sprintf("%+v", e)
	if !strings.HasPrefix(plus, "boom\n") || !strings.Contains(plus, "TestFormat") {
		t.Errorf("%%+v = %q, want message followed by stack trace", plus)
	}
	if hash := fmt.Sprintf("%#v", e); !strings.Contains(hash, `Tag:"T"`) {
		t.Errorf("%%#v = %q, want struct fields", hash)
	}
}
