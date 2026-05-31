package tests

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/errors/tests/crash2"
)

func TestNew(t *testing.T) {
	c := crash2.CrashStruct{}
	origErr := c.Crash2()

	fmt.Printf("%#v\n\n", origErr)
	err := origErr.(*errors.Error)
	fmt.Printf("%#v\n\n", err.StackFrames(false))
	fmt.Printf("%s\n\n", err.StackTrace())

	t.Error()
}
