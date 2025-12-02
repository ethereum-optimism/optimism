package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

type ExtNetworkConfig struct {
	L2NetworkName      string
	L1ChainID          eth.ChainID
	L2ELEndpoint       string
	L1CLBeaconEndpoint string
	L1ELEndpoint       string
}
