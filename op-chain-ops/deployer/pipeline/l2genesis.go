package pipeline

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

func GenerateL2Genesis(ctx context.Context, env *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "generate-l2-genesis")

	lgr.Info("generating L2 genesis", "id", chainID.Hex())

	var artifactsFS foundry.StatDirFs
	var err error
	if intent.ContractArtifactsURL.Scheme == "file" {
		fs := os.DirFS(intent.ContractArtifactsURL.Path)
		artifactsFS = fs.(foundry.StatDirFs)
	} else {
		return fmt.Errorf("only file:// artifacts URLs are supported")
	}

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	initCfg, err := state.CombineDeployConfig(intent, thisIntent, st, thisChainState)
	if err != nil {
		return fmt.Errorf("failed to combine L2 init config: %w", err)
	}

	var dump *foundry.ForgeAllocs
	err = CallScriptBroadcast(
		ctx,
		CallScriptBroadcastOpts{
			L1ChainID:   big.NewInt(int64(intent.L1ChainID)),
			Logger:      lgr,
			ArtifactsFS: artifactsFS,
			Deployer:    env.Deployer,
			Signer:      env.Signer,
			Client:      env.L1Client,
			Broadcaster: DiscardBroadcaster,
			Handler: func(host *script.Host) error {
				err := opsm.L2Genesis(host, &opsm.L2GenesisInput{
					L1Deployments: opsm.L1Deployments{
						L1CrossDomainMessengerProxy: thisChainState.L1CrossDomainMessengerProxyAddress,
						L1StandardBridgeProxy:       thisChainState.L1StandardBridgeProxyAddress,
						L1ERC721BridgeProxy:         thisChainState.L1ERC721BridgeProxyAddress,
					},
					L2Config: initCfg.L2InitializationConfig,
				})
				if err != nil {
					return fmt.Errorf("failed to call L2Genesis script: %w", err)
				}

				host.Wipe(env.Deployer)

				dump, err = host.StateDump()
				if err != nil {
					return fmt.Errorf("failed to dump state: %w", err)
				}

				return nil
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to call L2Genesis script: %w", err)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(gw).Encode(dump); err != nil {
		return fmt.Errorf("failed to encode state dump: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	thisChainState.Genesis = buf.Bytes()
	startHeader, err := env.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get start block: %w", err)
	}
	thisChainState.StartBlock = startHeader

	if err := env.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
