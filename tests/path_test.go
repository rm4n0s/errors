package tests

import (
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
