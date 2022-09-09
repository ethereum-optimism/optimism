package derive

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// Progress represents the progress of a derivation stage:
// the input L1 block that is being processed, and whether it's fully processed yet.
type Progress struct {
	Origin eth.L1BlockRef
	// Closed means that the Current has no more data that the stage may need.
	Closed bool
}

func (pr *Progress) Update(outer Progress) (changed bool, err error) {
	if outer.Origin.Number < pr.Origin.Number {
		return false, nil
	}
	if pr.Closed {
		if outer.Closed {
			if pr.Origin.ID() != outer.Origin.ID() {
				return true, NewResetError(fmt.Errorf("outer stage changed origin from %s to %s without opening it", pr.Origin, outer.Origin))
			}
			return false, nil
		} else {
			if pr.Origin.Hash != outer.Origin.ParentHash {
				return true, NewResetError(fmt.Errorf("detected internal pipeline reorg of L1 origin data from %s to %s", pr.Origin, outer.Origin))
			}
			pr.Origin = outer.Origin
			pr.Closed = false
			return true, nil
		}
	} else {
		if pr.Origin.ID() != outer.Origin.ID() {
			return true, NewResetError(fmt.Errorf("outer stage changed origin from %s to %s before closing it", pr.Origin, outer.Origin))
		}
		if outer.Closed {
			pr.Closed = true
			return true, nil
		} else {
			return false, nil
		}
	}
}
