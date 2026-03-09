package stack

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type L2Deployment interface {
	SystemConfigProxyAddr() common.Address
	DisputeGameFactoryProxyAddr() common.Address
	L1StandardBridgeProxyAddr() common.Address
	// Other addresses will be added here later
}

type Keys interface {
	Secret(key devkeys.Key) *ecdsa.PrivateKey
	Address(key devkeys.Key) common.Address
}

// L2Network represents a L2 chain, a collection of configuration and node resources.
type L2Network interface {
	Network
	RollupConfig() *rollup.Config
	Deployment() L2Deployment
	Keys() Keys

	L1() L1Network

	L2Batchers() []L2Batcher
	L2Proposers() []L2Proposer
	L2Challengers() []L2Challenger
	L2CLNodes() []L2CLNode
	L2ELNodes() []L2ELNode
	Conductors() []Conductor
	RollupBoostNodes() []RollupBoostNode
	OPRBuilderNodes() []OPRBuilderNode
}
