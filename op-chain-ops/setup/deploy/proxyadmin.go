package deploy

import (
	"github.com/ethereum/go-ethereum/common"
)

// L1ProxyAdmin defines how to deploy a ProxyAdmin contract to L1
type L1ProxyAdmin struct {
	Args struct {
		ProxyAdminOwner common.Address `json:"proxyAdminOwner"`
	}

	Addresses struct{}
}

func (cfg *L1ProxyAdmin) ScriptTarget() string {
	return "Deploy.s.sol"
}

func (cfg *L1ProxyAdmin) ScriptSig() string {
	return "deployProxyAdmin"
}

func (cfg *L1ProxyAdmin) ScriptDependencies() []string {
	return []string{"ProxyAdmin.sol"}
}

func (cfg *L1ProxyAdmin) ScriptAddresses() any {
	return &cfg.Addresses
}

func (cfg *L1ProxyAdmin) ScriptArgs() any {
	return &cfg.Args
}
