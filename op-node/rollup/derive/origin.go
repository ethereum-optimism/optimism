package derive

import (
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
)

var ReorgErr = errors.New("reorg")

type Origin struct {
	Current eth.L1BlockRef
	Closed  bool
}

func (l1s Origin) Progress() Origin {
	return l1s
}

func (l1s *Origin) UpdateOrigin(outer Origin) (changed bool, err error) {
	if l1s.Closed {
		if outer.Closed {
			if l1s.Current != outer.Current {
				return true, fmt.Errorf("outer stage changed origin from %s to %s without opening it", l1s.Current, outer.Current)
			}
			return false, nil
		} else {
			if l1s.Current.Hash != outer.Current.ParentHash {
				return true, fmt.Errorf("detected internal pipeline reorg of L1 origin data from %s to %s: %w", l1s.Current, outer.Current, ReorgErr)
			}
			l1s.Current = outer.Current
			l1s.Closed = false
			return true, nil
		}
	} else {
		if l1s.Current != outer.Current {
			return true, fmt.Errorf("outer stage changed origin from %s to %s before closing it", l1s.Current, outer.Current)
		}
		if outer.Closed {
			l1s.Closed = true
			return true, nil
		} else {
			return false, nil
		}
	}
}
