package opcm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ForgeEnv runs the deploy scripts with forge against a live L1. Live transactions are
// produced by a keyless forge dry run, then signed and sent by txmgr, not forge.
type ForgeEnv struct {
	Client               *forge.Client
	Context              context.Context
	L1RPCUrl             string
	PrivateKey           string
	Logger               log.Logger
	ReceiptQueryInterval time.Duration
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript simulates a script and sends its transactions to L1 with txmgr.
func broadcastScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, true)
}

// readScript runs a script that only reads.
func readScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, false)
}

func runScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I, broadcast bool) (O, error) {
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
	if !broadcast {
		return newCaller(env.Client)(ctx, input, "--rpc-url", env.L1RPCUrl)
	}
	return broadcastLiveScript(ctx, env, newCaller, input)
}
