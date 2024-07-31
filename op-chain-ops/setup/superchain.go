package setup

import "github.com/ethereum/go-ethereum/common"

type superchain struct {
	name             string
	SuperchainConfig common.Address
	ProtocolVersions common.Address
	Implementations  struct {
		// TODO list of implementation addresses, for proxies to point to
	}
}
