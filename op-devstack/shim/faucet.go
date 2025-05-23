package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type FaucetConfig struct {
	CommonConfig
	ID     stack.FaucetID
	Client client.RPC
}

// presetFaucet wraps around a faucet-service,
// and is meant to fund users by making faucet RPC requests.
// This deconflicts funding requests by parallel tests from the same funding account.
type presetFaucet struct {
	commonImpl
	id           stack.FaucetID
	faucetClient *sources.FaucetClient
}

var _ stack.Faucet = (*presetFaucet)(nil)

func NewFaucet(cfg FaucetConfig) stack.Faucet {
	cfg.T = cfg.T.WithCtx(stack.ContextWithKind(cfg.T.Ctx(), stack.FaucetKind), "id", cfg.ID)
	return &presetFaucet{
		id:           cfg.ID,
		commonImpl:   newCommon(cfg.CommonConfig),
		faucetClient: sources.NewFaucetClient(cfg.Client),
	}
}

func (p *presetFaucet) ID() stack.FaucetID {
	return p.id
}

func (p *presetFaucet) API() apis.Faucet {
	return p.faucetClient
}
