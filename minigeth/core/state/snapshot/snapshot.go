package snapshot

import "github.com/ethereum/go-ethereum/common"

// entirely stubs, this is never created

type Tree interface {
}

type Snapshot interface {
	// Storage directly retrieves the storage data associated with a particular hash,
	// within a particular account.
	Storage(accountHash, storageHash common.Hash) ([]byte, error)
}
