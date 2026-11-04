package opcm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ForgeEnv runs the deploy scripts with forge against a live L1.
type ForgeEnv struct {
	Client               *forge.Client
	Context              context.Context
	L1RPCUrl             string
	PrivateKey           string
	Logger               log.Logger
	ReceiptQueryInterval time.Duration
	// sentBlock is the block of the last receipt of a send. Later scripts fork at it or after it, so
	// that they see the send when a backend of a load-balanced RPC lags the one that served it.
	sentBlock uint64
}

func (env *ForgeEnv) logger() log.Logger {
	if env.Logger == nil {
		return log.Root()
	}
	return env.Logger
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript simulates a script and sends its transactions to L1 with txmgr.
func broadcastScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	var out O
	if env.Client == nil {
		return out, errors.New("forge client is nil")
	}
	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if env.L1RPCUrl == "" {
		return out, errors.New("L1 RPC URL is required for Forge deployments")
	}
	return broadcastLiveScript(ctx, env, newCaller, input)
}
