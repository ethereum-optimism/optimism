package oracle

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var preimages = make(map[common.Hash][]byte)

func Preimage(hash common.Hash) []byte {
	val, ok := preimages[hash]
	if !ok {
		fmt.Println("can't find preimage", hash)
		panic("preimage missing")
	}
	return val
}
