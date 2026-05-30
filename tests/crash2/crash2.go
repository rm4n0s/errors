package crash2

import "github.com/rm4n0s/errors/tests/crash2/crash1"

type CrashStruct struct {
}

func (c *CrashStruct) Crash2() error {
	return crash1.Crash1()
}
