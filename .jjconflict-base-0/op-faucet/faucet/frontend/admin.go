package frontend

import (
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type AdminBackend interface {
	EnableFaucet(id ftypes.FaucetID)
	DisableFaucet(id ftypes.FaucetID)
	Faucets() map[ftypes.FaucetID]eth.ChainID
	Defaults() map[eth.ChainID]ftypes.FaucetID
}

type AdminFrontend struct {
	b AdminBackend
}

func NewAdminFrontend(b AdminBackend) *AdminFrontend {
	return &AdminFrontend{b: b}
}

func (a *AdminFrontend) EnableFaucet(id ftypes.FaucetID) {
	a.b.EnableFaucet(id)
}

func (a *AdminFrontend) DisableFaucet(id ftypes.FaucetID) {
	a.b.DisableFaucet(id)
}

func (a *AdminFrontend) Faucets() map[ftypes.FaucetID]eth.ChainID {
	return a.b.Faucets()
}

func (a *AdminFrontend) Defaults() map[eth.ChainID]ftypes.FaucetID {
	return a.b.Defaults()
}
