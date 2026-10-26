package opcm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// ForgeEnv runs the deploy scripts with forge, either against a live L1 or offline on the L1 state
// of a genesis deployment.
type ForgeEnv struct {
	Client     *forge.Client
	Context    context.Context
	L1RPCUrl   string
	PrivateKey string
	// UnlockedSender, when set, sends the transactions from this account of the RPC node, which
	// must be unlocked, as on an anvil fork. PrivateKey is then not used.
	UnlockedSender common.Address
	// Genesis is the L1 state of a genesis deployment. When it is set, the scripts run on it offline.
	Genesis *GenesisState
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript runs a script that sends transactions. Against a live L1, forge broadcasts them. On
// the state of a genesis deployment, the state that the script leaves replaces it.
func broadcastScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, true)
}

// readScript runs a script that only reads the L1 state.
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
	flags := []string{"--rpc-url", env.L1RPCUrl}
	switch {
	case !broadcast:
	case env.UnlockedSender != (common.Address{}):
		flags = append(flags, "--broadcast", "--unlocked", "--sender", env.UnlockedSender.Hex())
	case env.PrivateKey != "":
		flags = append(flags, "--broadcast", "--private-key", env.PrivateKey)
	default:
		return out, errors.New("private key is required for Forge deployments")
	}
	out, err := newCaller(env.Client)(ctx, input, flags...)
	return out, err
}
