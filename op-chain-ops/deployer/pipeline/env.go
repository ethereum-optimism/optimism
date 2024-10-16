package pipeline

import (
	"context"
	"fmt"
	"path"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type Env struct {
	Workdir  string
	L1Client *ethclient.Client
	Signer   opcrypto.SignerFn
	Deployer common.Address
	Logger   log.Logger
}

func ReadIntent(workdir string) (*state.Intent, error) {
	intentPath := path.Join(workdir, "intent.toml")
	intent, err := jsonutil.LoadTOML[state.Intent](intentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent file: %w", err)
	}
	return intent, nil
}

func ReadState(workdir string) (*state.State, error) {
	statePath := path.Join(workdir, "state.json")
	st, err := jsonutil.LoadJSON[state.State](statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	return st, nil
}

func WriteState(workdir string, st *state.State) error {
	statePath := path.Join(workdir, "state.json")
	return st.WriteToFile(statePath)
}

type ArtifactsBundle struct {
	L1 foundry.StatDirFs
	L2 foundry.StatDirFs
}

type Stage func(ctx context.Context, env *Env, bundle ArtifactsBundle, intent *state.Intent, st *state.State) error
