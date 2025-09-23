package types

import "fmt"

type ChainID uint64

func (c ChainID) String() string {
	return fmt.Sprintf("%d", c)
}
