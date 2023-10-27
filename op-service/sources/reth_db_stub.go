//go:build !rethdb

package sources

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// FetchRethReceipts stub; Not available without `rethdb` build tag.
func FetchRethReceipts(dbPath string, blockHash *common.Hash) (types.Receipts, error) {
	panic("unimplemented! Did you forget to enable the `rethdb` build tag?")
}
