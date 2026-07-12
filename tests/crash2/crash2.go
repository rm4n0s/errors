package crash2

import "github.com/rm4n0s/errors/tests/crash2/crash1"

type Crash2Struct struct {
}

func (c *Crash2Struct) Crash2() error {
	return crash1.Crash1()
}
