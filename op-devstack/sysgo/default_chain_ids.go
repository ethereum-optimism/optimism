package sysgo

import "github.com/ethereum-optimism/optimism/op-service/eth"

var (
	DefaultL1ID  = eth.ChainIDFromUInt64(900)
	DefaultL2AID = eth.ChainIDFromUInt64(901)
	DefaultL2BID = eth.ChainIDFromUInt64(902)
)
