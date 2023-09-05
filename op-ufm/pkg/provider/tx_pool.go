package provider

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TransactionPool is used locally to share transactions between providers under the same pool
type TransactionPool map[string]*NetworkTransactionPool

// NetworkTransactionPool is used locally to share transactions between providers under the same network
type NetworkTransactionPool struct {
	M            sync.Mutex
	Transactions map[string]*TransactionState
	Expected     int
	Nonce        uint64
}

type TransactionState struct {
	// Transaction hash
	Hash common.Hash

	// Mutex
	M sync.Mutex

	SentAt         time.Time
	ProviderSource string

	FirstSeen time.Time

	// Map of providers that have seen this transaction, and when
	// Once all providers have seen the transaction it is removed from the pool
	SeenBy map[string]time.Time
}
