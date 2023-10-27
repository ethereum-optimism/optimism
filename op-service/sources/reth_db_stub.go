//go:build !rethdb

package sources

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// FetchRethReceipts fetches the receipts for the given block hash...
func FetchRethReceipts(dbPath string, blockHash *common.Hash) (types.Receipts, error) {
	panic("unimplemented!")
}
