package e2eutils

import (
	"math/big"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var testingJWTSecret = [32]byte{123}

// WriteDefaultJWT writes a testing JWT to the temporary directory of the test and returns the path to the JWT file.
func WriteDefaultJWT(t TestingBase) string {
	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	if err := os.WriteFile(jwtPath, []byte(hexutil.Encode(testingJWTSecret[:])), 0600); err != nil {
		t.Fatalf("failed to prepare jwt file for geth: %v", err)
	}
	return jwtPath
}

func uint64ToBig(in uint64) *hexutil.Big {
	return (*hexutil.Big)(new(big.Int).SetUint64(in))
}

// DeployParams bundles the deployment parameters to generate further testing inputs with.
type DeployParams struct {
	DeployConfig   *genesis.DeployConfig
	MnemonicConfig *MnemonicConfig
	Secrets        *Secrets
	Addresses      *Addresses
}

// TestParams parametrizes the most essential rollup configuration parameters
type TestParams struct {
	MaxSequencerDrift   uint64
	SequencerWindowSize uint64
	ChannelTimeout      uint64
	L1BlockTime         uint64
}

func MakeDeployParams(t require.TestingT, tp *TestParams) *DeployParams {
	mnemonicCfg := DefaultMnemonicConfig
	secrets, err := mnemonicCfg.Secrets()
	require.NoError(t, err)
	addresses := secrets.Addresses()
	deployConfig := &genesis.DeployConfig{
		L1ChainID:   901,
		L2ChainID:   902,
		L2BlockTime: 2,

		MaxSequencerDrift:   tp.MaxSequencerDrift,
		SequencerWindowSize: tp.SequencerWindowSize,
		ChannelTimeout:      tp.ChannelTimeout,
		P2PSequencerAddress: addresses.SequencerP2P,
		BatchInboxAddress:   common.Address{0: 0x42, 19: 0xff}, // tbd
		BatchSenderAddress:  addresses.Batcher,

		L2OutputOracleSubmissionInterval: 6,
		L2OutputOracleStartingTimestamp:  -1,
		L2OutputOracleProposer:           addresses.Proposer,
		L2OutputOracleChallenger:         common.Address{}, // tbd

		FinalSystemOwner: addresses.SysCfgOwner,

		L1BlockTime:                 tp.L1BlockTime,
		L1GenesisBlockNonce:         0,
		CliqueSignerAddress:         common.Address{}, // proof of stake, no clique
		L1GenesisBlockTimestamp:     hexutil.Uint64(time.Now().Unix()),
		L1GenesisBlockGasLimit:      30_000_000,
		L1GenesisBlockDifficulty:    uint64ToBig(1),
		L1GenesisBlockMixHash:       common.Hash{},
		L1GenesisBlockCoinbase:      common.Address{},
		L1GenesisBlockNumber:        0,
		L1GenesisBlockGasUsed:       0,
		L1GenesisBlockParentHash:    common.Hash{},
		L1GenesisBlockBaseFeePerGas: uint64ToBig(1000_000_000), // 1 gwei
		FinalizationPeriodSeconds:   12,

		L2GenesisBlockNonce:         0,
		L2GenesisBlockGasLimit:      30_000_000,
		L2GenesisBlockDifficulty:    uint64ToBig(0),
		L2GenesisBlockMixHash:       common.Hash{},
		L2GenesisBlockNumber:        0,
		L2GenesisBlockGasUsed:       0,
		L2GenesisBlockParentHash:    common.Hash{},
		L2GenesisBlockBaseFeePerGas: uint64ToBig(1000_000_000),

		GasPriceOracleOverhead:      2100,
		GasPriceOracleScalar:        1000_000,
		DeploymentWaitConfirmations: 1,

		SequencerFeeVaultRecipient:               common.Address{19: 1},
		BaseFeeVaultRecipient:                    common.Address{19: 2},
		L1FeeVaultRecipient:                      common.Address{19: 3},
		BaseFeeVaultMinimumWithdrawalAmount:      uint64ToBig(1000_000_000), // 1 gwei
		L1FeeVaultMinimumWithdrawalAmount:        uint64ToBig(1000_000_000), // 1 gwei
		SequencerFeeVaultMinimumWithdrawalAmount: uint64ToBig(1000_000_000), // 1 gwei
		BaseFeeVaultWithdrawalNetwork:            uint8(1),                  // L2 withdrawal network
		L1FeeVaultWithdrawalNetwork:              uint8(1),                  // L2 withdrawal network
		SequencerFeeVaultWithdrawalNetwork:       uint8(1),                  // L2 withdrawal network

		EIP1559Elasticity:  10,
		EIP1559Denominator: 50,

		FundDevAccounts: false,
	}

	// Configure the DeployConfig with the expected developer L1
	// addresses.
	if err := deployConfig.InitDeveloperDeployedAddresses(); err != nil {
		panic(err)
	}

	return &DeployParams{
		DeployConfig:   deployConfig,
		MnemonicConfig: mnemonicCfg,
		Secrets:        secrets,
		Addresses:      addresses,
	}
}

// DeploymentsL1 captures the L1 addresses used in the deployment,
// commonly just the developer predeploys during testing,
// but later deployed contracts may be used in some tests too.
type DeploymentsL1 struct {
	L1CrossDomainMessengerProxy common.Address
	L1StandardBridgeProxy       common.Address
	L2OutputOracleProxy         common.Address
	OptimismPortalProxy         common.Address
	SystemConfigProxy           common.Address
}

// SetupData bundles the L1, L2, rollup and deployment configuration data: everything for a full test setup.
type SetupData struct {
	L1Cfg         *core.Genesis
	L2Cfg         *core.Genesis
	RollupCfg     *rollup.Config
	DeploymentsL1 DeploymentsL1
}

// AllocParams defines genesis allocations to apply on top of the genesis generated by deploy parameters.
// These allocations override existing allocations per account,
// i.e. the allocations are merged with AllocParams having priority.
type AllocParams struct {
	L1Alloc          core.GenesisAlloc
	L2Alloc          core.GenesisAlloc
	PrefundTestUsers bool
}

var etherScalar = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

// Ether converts a uint64 Ether amount into a *big.Int amount in wei units, for allocating test balances.
func Ether(v uint64) *big.Int {
	return new(big.Int).Mul(new(big.Int).SetUint64(v), etherScalar)
}

// Setup computes the testing setup configurations from deployment configuration and optional allocation parameters.
func Setup(t require.TestingT, deployParams *DeployParams, alloc *AllocParams) *SetupData {
	deployConf := deployParams.DeployConfig
	l1Genesis, err := genesis.BuildL1DeveloperGenesis(deployConf)
	require.NoError(t, err, "failed to create l1 genesis")
	if alloc.PrefundTestUsers {
		for _, addr := range deployParams.Addresses.All() {
			l1Genesis.Alloc[addr] = core.GenesisAccount{
				Balance: Ether(1e12),
			}
		}
	}
	for addr, val := range alloc.L1Alloc {
		l1Genesis.Alloc[addr] = val
	}

	l1Block := l1Genesis.ToBlock()

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(deployConf, l1Block)
	require.NoError(t, err, "failed to create l2 genesis")
	if alloc.PrefundTestUsers {
		for _, addr := range deployParams.Addresses.All() {
			l2Genesis.Alloc[addr] = core.GenesisAccount{
				Balance: Ether(1e12),
			}
		}
	}
	for addr, val := range alloc.L2Alloc {
		l2Genesis.Alloc[addr] = val
	}

	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   l1Block.Hash(),
				Number: 0,
			},
			L2: eth.BlockID{
				Hash:   l2Genesis.ToBlock().Hash(),
				Number: 0,
			},
			L2Time:       uint64(deployConf.L1GenesisBlockTimestamp),
			SystemConfig: SystemConfigFromDeployConfig(deployConf),
		},
		BlockTime:              deployConf.L2BlockTime,
		MaxSequencerDrift:      deployConf.MaxSequencerDrift,
		SeqWindowSize:          deployConf.SequencerWindowSize,
		ChannelTimeout:         deployConf.ChannelTimeout,
		L1ChainID:              new(big.Int).SetUint64(deployConf.L1ChainID),
		L2ChainID:              new(big.Int).SetUint64(deployConf.L2ChainID),
		BatchInboxAddress:      deployConf.BatchInboxAddress,
		DepositContractAddress: predeploys.DevOptimismPortalAddr,
		L1SystemConfigAddress:  predeploys.DevSystemConfigAddr,
		RegolithTime:           deployConf.RegolithTime(uint64(deployConf.L1GenesisBlockTimestamp)),
	}

	deploymentsL1 := DeploymentsL1{
		L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
		L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
		L2OutputOracleProxy:         predeploys.DevL2OutputOracleAddr,
		OptimismPortalProxy:         predeploys.DevOptimismPortalAddr,
		SystemConfigProxy:           predeploys.DevSystemConfigAddr,
	}

	return &SetupData{
		L1Cfg:         l1Genesis,
		L2Cfg:         l2Genesis,
		RollupCfg:     rollupCfg,
		DeploymentsL1: deploymentsL1,
	}
}

func SystemConfigFromDeployConfig(deployConfig *genesis.DeployConfig) eth.SystemConfig {
	return eth.SystemConfig{
		BatcherAddr: deployConfig.BatchSenderAddress,
		Overhead:    eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(deployConfig.GasPriceOracleOverhead))),
		Scalar:      eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(deployConfig.GasPriceOracleScalar))),
		GasLimit:    uint64(deployConfig.L2GenesisBlockGasLimit),
	}
}

// ForkedDeployConfig returns a deploy config that's suitable for use with a
// forked L1.
func ForkedDeployConfig(t require.TestingT, mnemonicCfg *MnemonicConfig, startBlock *types.Block) *genesis.DeployConfig {
	startTag := rpc.BlockNumberOrHashWithHash(startBlock.Hash(), true)
	secrets, err := mnemonicCfg.Secrets()
	require.NoError(t, err)
	addrs := secrets.Addresses()
	marshalable := genesis.MarshalableRPCBlockNumberOrHash(startTag)
	out := &genesis.DeployConfig{
		L1StartingBlockTag:               &marshalable,
		L1ChainID:                        1,
		L2ChainID:                        10,
		L2BlockTime:                      2,
		MaxSequencerDrift:                3600,
		SequencerWindowSize:              100,
		ChannelTimeout:                   40,
		P2PSequencerAddress:              addrs.SequencerP2P,
		BatchInboxAddress:                common.HexToAddress("0xff00000000000000000000000000000000000000"),
		BatchSenderAddress:               addrs.Batcher,
		FinalSystemOwner:                 addrs.SysCfgOwner,
		L1GenesisBlockDifficulty:         uint64ToBig(0),
		L1GenesisBlockBaseFeePerGas:      uint64ToBig(0),
		L2OutputOracleSubmissionInterval: 10,
		L2OutputOracleStartingTimestamp:  int(startBlock.Time()),
		L2OutputOracleProposer:           addrs.Proposer,
		L2OutputOracleChallenger:         addrs.Deployer,
		L2GenesisBlockGasLimit:           hexutil.Uint64(15_000_000),
		// taken from devnet, need to check this
		L2GenesisBlockBaseFeePerGas: uint64ToBig(0x3B9ACA00),
		L2GenesisBlockDifficulty:    uint64ToBig(0),
		L1BlockTime:                 12,
		CliqueSignerAddress:         addrs.CliqueSigner,
		FinalizationPeriodSeconds:   2,
		DeploymentWaitConfirmations: 1,
		EIP1559Elasticity:           10,
		EIP1559Denominator:          50,
		GasPriceOracleOverhead:      2100,
		GasPriceOracleScalar:        1_000_000,
		FundDevAccounts:             true,
	}
	return out
}
