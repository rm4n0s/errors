package crash1

import (
	"github.com/rm4n0s/errors"
)

func Crash1() error {
	return errors.New("Crash1Tag", "crash 1 tagged")
}
