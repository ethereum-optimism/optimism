package actions

import "github.com/ethereum/go-ethereum/common"

type DeploymentsL1 struct {
	L1CrossDomainMessengerProxy common.Address
	L1StandardBridgeProxy       common.Address
	L2OutputOracleProxy         common.Address
	OptimismPortalProxy         common.Address
}

type DeploymentsL2 struct {
	L1Block common.Address
}

type Deployments struct {
	DeploymentsL1
	DeploymentsL2
}
