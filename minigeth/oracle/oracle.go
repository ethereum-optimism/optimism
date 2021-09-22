package oracle

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func GetProvedAccountBytes(blockNumber *big.Int, stateRoot common.Hash, addr common.Address) []byte {
	fmt.Println("ORACLE GetProvedAccountBytes:", blockNumber, stateRoot, addr)
	return []byte("12")
}
