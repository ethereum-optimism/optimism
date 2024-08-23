package interopgen

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/beacondeposit"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/deployers"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

var (
	// sysGenesisDeployer is used as tx.origin/msg.sender on system genesis script calls.
	// At the end we verify none of the deployed contracts persist (there may be temporary ones, to insert bytecode).
	sysGenesisDeployer = common.Address(crypto.Keccak256([]byte("System genesis deployer"))[12:])
)

func Deploy(logger log.Logger, fa *foundry.ArtifactsFS, srcFS *foundry.SourceMapFS, cfg *WorldConfig) (*WorldDeployment, *WorldOutput, error) {
	// Sanity check all L2s have consistent chain ID and attach to the same L1
	for id, l2Cfg := range cfg.L2s {
		if fmt.Sprintf("%d", l2Cfg.L2ChainID) != id {
			return nil, nil, fmt.Errorf("chain L2 %s declared different L2 chain ID %d in config", id, l2Cfg.L2ChainID)
		}
		if !cfg.L1.ChainID.IsUint64() || cfg.L1.ChainID.Uint64() != l2Cfg.L1ChainID {
			return nil, nil, fmt.Errorf("chain L2 %s declared different L1 chain ID %d in config than global %d", id, l2Cfg.L1ChainID, cfg.L1.ChainID)
		}
	}

	deployments := &WorldDeployment{
		L2s: make(map[string]*L2Deployment),
	}

	l1Host := createL1(logger, fa, srcFS, cfg.L1)
	if err := l1Host.EnableCheats(); err != nil {
		return nil, nil, fmt.Errorf("failed to enable cheats in L1 state: %w", err)
	}

	l1Deployment, err := initialL1(l1Host, cfg.L1)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deploy initial L1 content: %w", err)
	}
	deployments.L1 = l1Deployment

	superDeployment, err := deploySuperchainToL1(l1Host, cfg.Superchain)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deploy superchain to L1: %w", err)
	}
	deployments.Superchain = superDeployment

	for l2ChainID, l2Cfg := range cfg.L2s {
		l2Deployment, err := deployL2ToL1(l1Host, cfg.Superchain, superDeployment, l2Cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to deploy L2 %d to L1: %w", &l2ChainID, err)
		}
		deployments.L2s[l2ChainID] = l2Deployment
	}

	out := &WorldOutput{
		L2s: make(map[string]*L2Output),
	}
	l1Out, err := completeL1(l1Host, cfg.L1)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to complete L1: %w", err)
	}
	out.L1 = l1Out

	l1GenesisBlock := l1Out.Genesis.ToBlock()
	genesisTimestamp := l1Out.Genesis.Timestamp

	for l2ChainID, l2Cfg := range cfg.L2s {
		l2Host := createL2(logger, fa, srcFS, l2Cfg, genesisTimestamp)
		if err := l2Host.EnableCheats(); err != nil {
			return nil, nil, fmt.Errorf("failed to enable cheats in L2 state %s: %w", l2ChainID, err)
		}
		if err := genesisL2(l2Host, l2Cfg, deployments.L2s[l2ChainID]); err != nil {
			return nil, nil, fmt.Errorf("failed to apply genesis data to L2 %s: %w", l2ChainID, err)
		}
		l2Out, err := completeL2(l2Host, l2Cfg, l1GenesisBlock, deployments.L2s[l2ChainID])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to complete L2 %s: %w", l2ChainID, err)
		}
		out.L2s[l2ChainID] = l2Out
	}
	return deployments, out, nil
}

func createL1(logger log.Logger, fa *foundry.ArtifactsFS, srcFS *foundry.SourceMapFS, cfg *L1Config) *script.Host {
	l1Context := script.Context{
		ChainID:      cfg.ChainID,
		Sender:       sysGenesisDeployer,
		Origin:       sysGenesisDeployer,
		FeeRecipient: common.Address{},
		GasLimit:     script.DefaultFoundryGasLimit,
		BlockNum:     uint64(cfg.L1GenesisBlockNumber),
		Timestamp:    uint64(cfg.L1GenesisBlockTimestamp),
		PrevRandao:   cfg.L1GenesisBlockMixHash,
		BlobHashes:   nil,
	}
	l1Host := script.NewHost(logger.New("role", "l1", "chain", cfg.ChainID), fa, srcFS, l1Context)
	return l1Host
}

func createL2(logger log.Logger, fa *foundry.ArtifactsFS, srcFS *foundry.SourceMapFS, l2Cfg *L2Config, genesisTimestamp uint64) *script.Host {
	l2Context := script.Context{
		ChainID:      new(big.Int).SetUint64(l2Cfg.L2ChainID),
		Sender:       sysGenesisDeployer,
		Origin:       sysGenesisDeployer,
		FeeRecipient: common.Address{},
		GasLimit:     script.DefaultFoundryGasLimit,
		BlockNum:     uint64(l2Cfg.L2GenesisBlockNumber),
		Timestamp:    genesisTimestamp,
		PrevRandao:   l2Cfg.L2GenesisBlockMixHash,
		BlobHashes:   nil,
	}
	l2Host := script.NewHost(logger.New("role", "l2", "chain", l2Cfg.L2ChainID), fa, srcFS, l2Context)
	l2Host.SetEnvVar("OUTPUT_MODE", "none") // we don't use the cheatcode, but capture the state outside of EVM execution
	l2Host.SetEnvVar("FORK", "granite")     // latest fork
	return l2Host
}

// initialL1 deploys basics such as preinstalls to L1  (incl. EIP-4788)
func initialL1(l1Host *script.Host, cfg *L1Config) (*L1Deployment, error) {
	l1Host.SetTxOrigin(sysGenesisDeployer)

	return &L1Deployment{
		// any contracts we need to register here?
	}, nil
}

func deploySuperchainToL1(l1Host *script.Host, superCfg *SuperchainConfig) (*SuperchainDeployment, error) {
	l1Host.SetTxOrigin(superCfg.Deployer)

	superDeployment, err := deployers.DeploySuperchain(l1Host, &deployers.DeploySuperchainInput{
		ProxyAdminOwner:            superCfg.ProxyAdminOwner,
		ProtocolVersionsOwner:      superCfg.ProtocolVersionsOwner,
		Guardian:                   superCfg.SuperchainConfigGuardian,
		Paused:                     superCfg.Paused,
		RequiredProtocolVersion:    superCfg.RequiredProtocolVersion,
		RecommendedProtocolVersion: superCfg.RecommendedProtocolVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Superchain contracts: %w", err)
	}

	implementationsDeployment, err := deployers.DeployImplementations(l1Host, &deployers.DeployImplementationsInput{
		WithdrawalDelaySeconds:          superCfg.Implementations.FaultProof.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            superCfg.Implementations.FaultProof.MinProposalSizeBytes,
		ChallengePeriodSeconds:          superCfg.Implementations.FaultProof.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       superCfg.Implementations.FaultProof.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: superCfg.Implementations.FaultProof.DisputeGameFinalityDelaySeconds,
		Release:                         superCfg.Implementations.Release,
		SuperchainConfigProxy:           superDeployment.SuperchainConfigProxy,
		ProtocolVersionsProxy:           superDeployment.ProtocolVersionsProxy,
		UseInterop:                      superCfg.Implementations.UseInterop,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Implementations contracts: %w", err)
	}

	// Collect deployment addresses
	// This could all be automatic once we have better output-contract typing/scripting
	return &SuperchainDeployment{
		Implementations:       Implementations(*implementationsDeployment),
		ProxyAdmin:            superDeployment.SuperchainProxyAdmin,
		ProtocolVersions:      superDeployment.ProtocolVersionsImpl,
		ProtocolVersionsProxy: superDeployment.ProtocolVersionsProxy,
		SuperchainConfig:      superDeployment.SuperchainConfigImpl,
		SuperchainConfigProxy: superDeployment.SuperchainConfigProxy,
	}, nil
}

func deployL2ToL1(l1Host *script.Host, superCfg *SuperchainConfig, superDeployment *SuperchainDeployment, cfg *L2Config) (*L2Deployment, error) {
	if cfg.UseAltDA {
		return nil, errors.New("alt-da mode not supported yet")
	}

	l1Host.SetTxOrigin(cfg.Deployer)

	output, err := deployers.DeployOPChain(l1Host, &deployers.DeployOPChainInput{
		OpChainProxyAdminOwner: cfg.ProxyAdminOwner,
		SystemConfigOwner:      cfg.SystemConfigOwner,
		Batcher:                cfg.BatchSenderAddress,
		UnsafeBlockSigner:      cfg.P2PSequencerAddress,
		Proposer:               cfg.Proposer,
		Challenger:             cfg.Challenger,
		BasefeeScalar:          cfg.GasPriceOracleBaseFeeScalar,
		BlobBaseFeeScalar:      cfg.GasPriceOracleBlobBaseFeeScalar,
		L2ChainId:              new(big.Int).SetUint64(cfg.L2ChainID),
		Opsm:                   superDeployment.Opsm,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to deploy L2 OP chain: %w", err)
	}

	// Collect deployment addresses
	return &L2Deployment{
		L2OpchainDeployment: L2OpchainDeployment(*output),
	}, nil
}

func genesisL2(l2Host *script.Host, cfg *L2Config, deployment *L2Deployment) error {
	if err := deployers.L2Genesis(l2Host, &deployers.L2GenesisInput{
		L1Deployments: deployers.L1Deployments{
			L1CrossDomainMessengerProxy: deployment.L1CrossDomainMessengerProxy,
			L1StandardBridgeProxy:       deployment.L1StandardBridgeProxy,
			L1ERC721BridgeProxy:         deployment.L1ERC721BridgeProxy,
		},
		L2Config: cfg.L2InitializationConfig,
	}); err != nil {
		return fmt.Errorf("failed L2 genesis: %w", err)
	}

	return nil
}

func completeL1(l1Host *script.Host, cfg *L1Config) (*L1Output, error) {
	l1Genesis, err := genesis.NewL1Genesis(&genesis.DeployConfig{
		L2InitializationConfig: genesis.L2InitializationConfig{
			L2CoreDeployConfig: genesis.L2CoreDeployConfig{
				L1ChainID: cfg.ChainID.Uint64(),
			},
		},
		DevL1DeployConfig: cfg.DevL1DeployConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build L1 genesis template: %w", err)
	}
	allocs, err := l1Host.StateDump()
	if err != nil {
		return nil, fmt.Errorf("failed to dump L1 state: %w", err)
	}

	if err := noDeployed(allocs, sysGenesisDeployer); err != nil {
		return nil, fmt.Errorf("unexpected deployed account content by L1 genesis deployer: %w", err)
	}

	for addr, amount := range cfg.Prefund {
		acc := allocs.Accounts[addr]
		acc.Balance = amount
		allocs.Accounts[addr] = acc
	}

	l1Genesis.Alloc = allocs.Accounts

	// Insert an empty beaconchain deposit contract with valid empty-tree prestate.
	// This is part of dev-genesis, but not part of scripts yet.
	beaconDepositAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	if err := beacondeposit.InsertEmptyBeaconDepositContract(l1Genesis, beaconDepositAddr); err != nil {
		return nil, fmt.Errorf("failed to insert beacon deposit contract into L1 dev genesis: %w", err)
	}

	return &L1Output{
		Genesis: l1Genesis,
	}, nil
}

func completeL2(l2Host *script.Host, cfg *L2Config, l1Block *types.Block, deployment *L2Deployment) (*L2Output, error) {
	deployCfg := &genesis.DeployConfig{
		L2InitializationConfig: cfg.L2InitializationConfig,
		L1DependenciesConfig: genesis.L1DependenciesConfig{
			L1StandardBridgeProxy:       deployment.L1StandardBridgeProxy,
			L1CrossDomainMessengerProxy: deployment.L1CrossDomainMessengerProxy,
			L1ERC721BridgeProxy:         deployment.L1ERC721BridgeProxy,
			SystemConfigProxy:           deployment.SystemConfigProxy,
			OptimismPortalProxy:         deployment.OptimismPortalProxy,
			DAChallengeProxy:            common.Address{}, // unsupported for now
		},
	}
	// l1Block is used to determine genesis time.
	l2Genesis, err := genesis.NewL2Genesis(deployCfg, l1Block)
	if err != nil {
		return nil, fmt.Errorf("failed to build L2 genesis config: %w", err)
	}

	allocs, err := l2Host.StateDump()
	if err != nil {
		return nil, fmt.Errorf("failed to dump L1 state: %w", err)
	}

	if err := noDeployed(allocs, sysGenesisDeployer); err != nil {
		return nil, fmt.Errorf("unexpected deployed account content by L2 genesis deployer: %w", err)
	}

	for addr, amount := range cfg.Prefund {
		acc := allocs.Accounts[addr]
		acc.Balance = amount
		allocs.Accounts[addr] = acc
	}

	l2Genesis.Alloc = allocs.Accounts
	l2GenesisBlock := l2Genesis.ToBlock()

	rollupCfg, err := deployCfg.RollupConfig(l1Block, l2GenesisBlock.Hash(), l2GenesisBlock.NumberU64())
	if err != nil {
		return nil, fmt.Errorf("failed to build L2 rollup config: %w", err)
	}
	return &L2Output{
		Genesis:   l2Genesis,
		RollupCfg: rollupCfg,
	}, nil
}

func noDeployed(allocs *foundry.ForgeAllocs, deployer common.Address) error {
	// Sanity check we have no deploy output that's not meant to be there.
	for i := uint64(0); i <= allocs.Accounts[deployer].Nonce; i++ {
		addr := crypto.CreateAddress(deployer, i)
		if _, ok := allocs.Accounts[addr]; ok {
			return fmt.Errorf("system deployer output %s (deployed with nonce %d) was not cleaned up", addr, i)
		}
	}
	// Don't include the deployer account
	delete(allocs.Accounts, deployer)
	return nil
}
