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

	nerr := errors.NewWithMetadata("dbproblem", "didn't connect to db", "system", "sql", "failed", 3)
	fmt.Printf("%#v\n\n", nerr)
	fmt.Printf("%#v\n\n", nerr.StackFrames(true))
	t.Error()
}
