package fakepos

import "github.com/ethereum/go-ethereum/eth"

type Config struct {
	GethBackend       *eth.Ethereum
	Beacon            Beacon
	FinalizedDistance uint64
	SafeDistance      uint64
	BlockTime         uint64
}
