package system

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
)

// Backend serves the backend functionality of a fully composed L1-L2 setup,
// meant to serve the system-tests, without hardcoding the system setup itself.
type Backend interface {
	l1.Backend
	l1el.Backend
	// TODO

	//RequestProposer(name, ...opt)
	//RequestBatcher(name, ...opt)
	//RequestChallenger(name, ...opt)
	//RequestRollupNode(name, ...opt)
	//RequestEngine(name, ...opt)

	// etc. more ways to request services

	//Address(name) common.Address
	//Secret(name) *SecretKey
	//RPC(name) client.RPC
	//EthClient(name) *ethclient.Client
	//RollupClient(name) *sources.Client
}
