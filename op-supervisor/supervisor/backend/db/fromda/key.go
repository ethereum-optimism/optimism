package fromda

import (
	"fmt"
)

type Key struct {
	DerivedFrom uint64
	Derived     uint64
}

func (k Key) String() string {
	return fmt.Sprintf("derivedFrom: %d, derived: %d", k.DerivedFrom, k.Derived)
}
