package stack

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
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
	// ChainConfig returns the chain's OP Stack fork schedule and genesis configuration. Its
	// EIP-1559 parameters are the genesis values: from Holocene, the parameters in effect are
	// set by the SystemConfig and carried in each block's extraData.
	ChainConfig() *opparams.ChainConfig
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
}
