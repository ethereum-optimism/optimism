package opcm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ForgeEnv runs the deploy scripts with forge, against a live L1 or offline on a GenesisState. On a
// live L1, a keyless forge dry run produces the transactions, and txmgr signs and sends them.
type ForgeEnv struct {
	Client               *forge.Client
	Context              context.Context
	L1RPCUrl             string
	PrivateKey           string
	Logger               log.Logger
	ReceiptQueryInterval time.Duration
	// UnlockedSender, when set, makes forge itself send from this unlocked node account, such as
	// one of an anvil fork, instead of txmgr sending with PrivateKey.
	UnlockedSender common.Address
	// Genesis, when set, is the offline L1 state that the scripts run on.
	Genesis *GenesisState
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript runs a script that sends transactions: to the live L1 through txmgr, or on the
// genesis state.
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
	if env.Genesis != nil {
		err := env.Genesis.run(env.Client, broadcast, func(client *forge.Client, flags []string) error {
			var err error
			out, err = newCaller(client)(ctx, input, flags...)
			return err
		})
		return out, err
	}

	if env.L1RPCUrl == "" {
		return out, errors.New("L1 RPC URL is required for Forge deployments")
	}
	if !broadcast {
		return newCaller(env.Client)(ctx, input, "--rpc-url", env.L1RPCUrl)
	}
	if env.UnlockedSender != (common.Address{}) {
		return newCaller(env.Client)(ctx, input, "--rpc-url", env.L1RPCUrl, "--broadcast", "--unlocked", "--sender", env.UnlockedSender.Hex())
	}
	return broadcastLiveScript(ctx, env, newCaller, input)
}
