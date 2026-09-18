package deployer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// LocalDeployment embeds common contracts in genesis and deploys chains on the local L1.
// It shares the production deployment stages without the CLI's future scheduling policy.
type LocalDeployment struct {
	opts           ApplyPipelineOpts
	bundle         artifacts.Bundle
	anchorGameType *uint32
}

// PrepareLocalDeployment seals L1 genesis before it predicts or creates any chain contracts.
// The caller must finalize L2 allocations before it calls Deploy.
func PrepareLocalDeployment(ctx context.Context, opts ApplyPipelineOpts, anchorGameType *uint32) (*LocalDeployment, error) {
	if opts.DeployerPrivateKey == nil {
		return nil, fmt.Errorf("local deployment requires a deployer key")
	}
	intent, st := opts.Intent, opts.State
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return nil, err
	}
	bundle, err := artifacts.DownloadBundle(ctx, intent.L1ContractsLocator, intent.L2ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
	if err != nil {
		return nil, err
	}
	deployer := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
	host, err := env.DefaultScriptHost(broadcaster.NoopBroadcaster(), opts.Logger, deployer, bundle.L1, script.WithNoMaxCodeSize())
	if err != nil {
		return nil, err
	}
	scripts, err := opcm.NewScripts(host)
	if err != nil {
		return nil, err
	}
	pEnv := &pipeline.Env{
		StateWriter: pipeline.NoopStateWriter(), L1ScriptHost: host, Logger: opts.Logger,
		Broadcaster: broadcaster.NoopBroadcaster(), Deployer: deployer, Scripts: scripts,
		IsGenesis: true, DeployMockSP1Verifier: opts.DeployMockSP1Verifier,
		AllowUnoptimizedContracts: true, Context: ctx,
	}
	for _, stage := range []func(*pipeline.Env, *state.Intent, *state.State) error{
		pipeline.InitGenesisStrategy, pipeline.DeploySuperchain, pipeline.DeployImplementations,
		// Preinstalls clear the deployer account. Restore its funds before the live deployment.
		pipeline.PreinstallL1DevGenesis, pipeline.PrefundL1DevGenesis, pipeline.SealL1DevGenesis,
	} {
		if err := stage(pEnv, intent, st); err != nil {
			return nil, err
		}
	}
	if err := pipeline.GenerateInteropDepset(ctx, pEnv, intent, st); err != nil {
		return nil, err
	}
	predictionIntent := *intent
	predictionIntent.OPCMAddress = &st.ImplementationsDeployment.OpcmV2Impl
	predictionIntent.SuperchainConfigProxy = &st.SuperchainDeployment.SuperchainConfigProxy
	for _, chain := range intent.Chains {
		input, err := makePredictionInput(&predictionIntent, st, chain)
		if err != nil {
			return nil, err
		}
		out, err := scripts.DeployOPChain.Run(input)
		if err != nil {
			return nil, fmt.Errorf("predict chain %s: %w", chain.ID, err)
		}
		st.SetChainContracts(chain.ID, pipeline.OpChainContractsFromDeployOutput(out), false)
		if err := pipeline.SetStartBlockGenesisStrategy(pEnv, intent, st, chain.ID); err != nil {
			return nil, err
		}
		if err := pipeline.DeployAltDA(pEnv, intent, st, chain.ID); err != nil {
			return nil, err
		}
		if err := pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID); err != nil {
			return nil, err
		}
		if err := pipeline.PrefundL2DevGenesis(pEnv, intent, st, chain.ID); err != nil {
			return nil, err
		}
	}
	st.AppliedIntent = intent
	if err := opts.StateWriter.WriteState(st); err != nil {
		return nil, err
	}
	return &LocalDeployment{opts: opts, bundle: bundle, anchorGameType: anchorGameType}, nil
}

// Deploy creates each chain with one OPCM transaction before local L2 nodes start.
func (d *LocalDeployment) Deploy(ctx context.Context, rpcURL string) error {
	intent, st := d.opts.Intent, d.opts.State
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)
	if err := verifyContinuationHead(ctx, client, big.NewInt(0), st.L1DevGenesis.ToBlock().Hash()); err != nil {
		return err
	}
	rootIntent := *intent
	rootIntent.Chains = nil
	for _, chain := range intent.Chains {
		params, err := pipeline.ResolveChainProofParams(intent, chain)
		if err != nil {
			return err
		}
		requirements, err := pipeline.ResolveInitialDeployRequirements(params.DisputeGameType)
		if err != nil {
			return err
		}
		if requirements.Permissionless || d.anchorGameType != nil {
			if d.anchorGameType != nil {
				rootChain := *chain
				rootChain.DeployOverrides = make(map[string]any, len(chain.DeployOverrides)+1)
				for key, value := range chain.DeployOverrides {
					rootChain.DeployOverrides[key] = value
				}
				rootChain.DeployOverrides["respectedGameType"] = *d.anchorGameType
				chain = &rootChain
			}
			rootIntent.Chains = append(rootIntent.Chains, chain)
		}
	}
	if len(rootIntent.Chains) > 0 {
		rootState := *st
		superRoots, err := pipeline.DeploymentUsesSuperRoots(&rootIntent, st)
		if err != nil {
			return err
		}
		if superRoots {
			if d.anchorGameType == nil {
				rootIntent.Chains = intent.Chains
			}
			rootState.InteropDepSet, err = pipeline.BuildInteropDepSet(intent.Chains)
			if err != nil {
				return err
			}
		}
		if err := pipeline.ComputeGenesisOutputRoots(&pipeline.Env{Logger: d.opts.Logger, IsGenesis: true}, &rootIntent, &rootState); err != nil {
			return err
		}
	}
	deployer := crypto.PubkeyToAddress(d.opts.DeployerPrivateKey.PublicKey)
	capture := new(broadcaster.CaptureBroadcaster)
	host, err := env.DefaultForkedScriptHost(ctx, capture, d.opts.Logger, deployer, d.bundle.L1, rpcClient)
	if err != nil {
		return err
	}
	scripts, err := opcm.NewScripts(host)
	if err != nil {
		return err
	}
	pEnv := &pipeline.Env{
		StateWriter: pipeline.NoopStateWriter(), L1ScriptHost: host, L1Client: client,
		Logger: d.opts.Logger, Broadcaster: capture, Deployer: deployer, Scripts: scripts, Context: ctx,
	}
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(d.opts.DeployerPrivateKey, intent.L1ChainIDBig()))
	live, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger: d.opts.Logger, ChainID: intent.L1ChainIDBig(), Client: client,
		Signer: signer, From: deployer, ReceiptQueryInterval: d.opts.ReceiptQueryInterval,
	})
	if err != nil {
		return err
	}
	for _, chain := range intent.Chains {
		chainState, err := st.Chain(chain.ID)
		if err != nil {
			return err
		}
		params, err := pipeline.ResolveChainProofParams(intent, chain)
		if err != nil {
			return err
		}
		requirements, err := pipeline.ResolveInitialDeployRequirements(params.DisputeGameType)
		if err != nil {
			return err
		}
		anchor := opcm.DefaultStartingAnchorProposal()
		if requirements.Permissionless || d.anchorGameType != nil {
			anchor = opcm.Proposal{Root: chainState.StartingAnchorRoot.Root,
				L2SequenceNumber: new(big.Int).SetUint64(uint64(chainState.StartingAnchorRoot.L2SequenceNumber))}
		}
		input := pipeline.BuildDeployOPChainInput(params, chain.Roles, st.ImplementationsDeployment.OpcmV2Impl,
			st.SuperchainDeployment.SuperchainConfigProxy, chain.ID, st.Create2Salt.String(), chain.GasLimit, anchor, chain)
		result, err := pipeline.ExecuteOPChainDeployment(pEnv, st, chain.ID, input)
		if err != nil {
			return err
		}
		expected := *chainState
		expected.AltDAChallengeProxy, expected.AltDAChallengeImpl = common.Address{}, common.Address{}
		if result.Contracts() != expected.OpChainContracts {
			return fmt.Errorf("chain %s deployment addresses differ from prediction", chain.ID)
		}
		calls := capture.Drain()
		if err := validateContinuationBroadcast(calls, deployer, input.Opcm); err != nil {
			return err
		}
		live.Hook(calls[0])
		results, broadcastErr := live.Broadcast(ctx)
		receipt, err := successfulContinuationBroadcast(results, broadcastErr)
		if err != nil {
			return err
		}
		if err := validateContinuationReceiptCanonicality(ctx, rpcClient, receipt.Receipt); err != nil {
			return err
		}
		if err := verifyContinuationDeployment(ctx, client, result.Contracts(), &expected, input); err != nil {
			return err
		}
		if err := pipeline.RecordOPChainDeployment(st, result); err != nil {
			return err
		}
		if err := pipeline.DeployAltDA(pEnv, intent, st, chain.ID); err != nil {
			return err
		}
		if err := pipeline.DeployAdditionalDisputeGames(pEnv, intent, st, chain.ID); err != nil {
			return err
		}
		for _, call := range capture.Drain() {
			live.Hook(call)
		}
		if _, err := live.Broadcast(ctx); err != nil {
			return err
		}
	}
	return d.opts.StateWriter.WriteState(st)
}
