package derive

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

var ReorgErr = errors.New("reorg")

// Progress represents the progress of a derivation stage:
// the input L1 block that is being processed, and whether it's fully processed yet.
type Progress struct {
	Origin eth.L1BlockRef
	// Closed means that the Current has no more data that the stage may need.
	Closed bool
}

func (l1s *Progress) Update(outer Progress) (changed bool, err error) {
	if l1s.Closed {
		if outer.Closed {
			if l1s.Origin != outer.Origin {
				return true, fmt.Errorf("outer stage changed origin from %s to %s without opening it", l1s.Origin, outer.Origin)
			}
			return false, nil
		} else {
			if l1s.Origin.Hash != outer.Origin.ParentHash {
				return true, fmt.Errorf("detected internal pipeline reorg of L1 origin data from %s to %s: %w", l1s.Origin, outer.Origin, ReorgErr)
			}
			l1s.Origin = outer.Origin
			l1s.Closed = false
			return true, nil
		}
	} else {
		if l1s.Origin != outer.Origin {
			return true, fmt.Errorf("outer stage changed origin from %s to %s before closing it", l1s.Origin, outer.Origin)
		}
		if outer.Closed {
			l1s.Closed = true
			return true, nil
		} else {
			return false, nil
		}
	}
}
