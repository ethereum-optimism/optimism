package actions

import (
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

var testingJWTSecret = [32]byte{123}

func writeDefaultJWT(t *testing.T) string {
	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	if err := os.WriteFile(jwtPath, []byte(hexutil.Encode(testingJWTSecret[:])), 0600); err != nil {
		t.Fatalf("failed to prepare jwt file for geth: %v", err)
	}
	return jwtPath
}

func uint642big(in uint64) *hexutil.Big {
	return (*hexutil.Big)(new(big.Int).SetUint64(in))
}

type DeployParams struct {
	DeployConfig   *genesis.DeployConfig
	MnemonicConfig *MnemonicConfig
	Secrets        *Secrets
	Addresses      *Addresses
}

type TestParams struct {
	MaxSequencerDrift   uint64
	SequencerWindowSize uint64
	ChannelTimeout      uint64
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

		MaxSequencerDrift:      tp.MaxSequencerDrift,
		SequencerWindowSize:    tp.SequencerWindowSize,
		ChannelTimeout:         tp.ChannelTimeout,
		P2PSequencerAddress:    addresses.SequencerP2P,
		OptimismL2FeeRecipient: common.Address{0: 0x42, 19: 0xf0}, // tbd
		BatchInboxAddress:      common.Address{0: 0x42, 19: 0xff}, // tbd
		BatchSenderAddress:     addresses.Batcher,

		L2OutputOracleSubmissionInterval: 6,
		L2OutputOracleStartingTimestamp:  -1,
		L2OutputOracleProposer:           addresses.Proposer,
		L2OutputOracleOwner:              common.Address{}, // tbd

		L1BlockTime:                 15,
		L1GenesisBlockNonce:         0,
		CliqueSignerAddress:         addresses.CliqueSigner, // TODO: remove clique, or make it optional
		L1GenesisBlockTimestamp:     hexutil.Uint64(time.Now().Unix()),
		L1GenesisBlockGasLimit:      15_000_000,
		L1GenesisBlockDifficulty:    uint642big(1),
		L1GenesisBlockMixHash:       common.Hash{},
		L1GenesisBlockCoinbase:      common.Address{},
		L1GenesisBlockNumber:        0,
		L1GenesisBlockGasUsed:       0,
		L1GenesisBlockParentHash:    common.Hash{},
		L1GenesisBlockBaseFeePerGas: uint642big(1000_000_000), // 1 gwei

		L2GenesisBlockNonce:         0,
		L2GenesisBlockExtraData:     []byte{},
		L2GenesisBlockGasLimit:      15_000_000,
		L2GenesisBlockDifficulty:    uint642big(0),
		L2GenesisBlockMixHash:       common.Hash{},
		L2GenesisBlockCoinbase:      common.Address{0: 0x42, 19: 0xf0}, // matching OptimismL2FeeRecipient
		L2GenesisBlockNumber:        0,
		L2GenesisBlockGasUsed:       0,
		L2GenesisBlockParentHash:    common.Hash{},
		L2GenesisBlockBaseFeePerGas: uint642big(1000_000_000),

		OptimismBaseFeeRecipient:    common.Address{0: 0x42, 19: 0xf1}, // tbd
		OptimismL1FeeRecipient:      addresses.Batcher,
		L2CrossDomainMessengerOwner: common.Address{0: 0x42, 19: 0xf2}, // tbd
		GasPriceOracleOwner:         common.Address{0: 0x42, 19: 0xf3}, // tbd
		GasPriceOracleOverhead:      2100,
		GasPriceOracleScalar:        1000_000,
		GasPriceOracleDecimals:      6,
		DeploymentWaitConfirmations: 1,

		EIP1559Elasticity:  10,
		EIP1559Denominator: 50,

		FundDevAccounts: false,
	}
	return &DeployParams{
		DeployConfig:   deployConfig,
		MnemonicConfig: mnemonicCfg,
		Secrets:        secrets,
		Addresses:      addresses,
	}
}

type SetupData struct {
	L1Cfg       *core.Genesis
	L2Cfg       *core.Genesis
	RollupCfg   *rollup.Config
	Deployments Deployments
}

type AllocParams struct {
	L1Alloc core.GenesisAlloc
	L2Alloc core.GenesisAlloc
}

func Setup(t require.TestingT, deployParams *DeployParams, alloc *AllocParams) *SetupData {
	deployConf := deployParams.DeployConfig
	l1Genesis, err := genesis.BuildL1DeveloperGenesis(deployConf)
	require.NoError(t, err, "failed to create l1 genesis")
	for addr, val := range alloc.L1Alloc {
		l1Genesis.Alloc[addr] = val
	}

	l1Block := l1Genesis.ToBlock()
	l2Addrs := &genesis.L2Addresses{
		ProxyAdmin:                  predeploys.DevProxyAdminAddr,
		L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
		L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
	}

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(deployConf, l1Block, l2Addrs)
	require.NoError(t, err, "failed to create l2 genesis")
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
			L2Time: uint64(deployConf.L1GenesisBlockTimestamp),
		},
		BlockTime:              deployConf.L2BlockTime,
		MaxSequencerDrift:      deployConf.MaxSequencerDrift,
		SeqWindowSize:          deployConf.SequencerWindowSize,
		ChannelTimeout:         deployConf.ChannelTimeout,
		L1ChainID:              new(big.Int).SetUint64(deployConf.L1ChainID),
		L2ChainID:              new(big.Int).SetUint64(deployConf.L2ChainID),
		P2PSequencerAddress:    deployConf.P2PSequencerAddress,
		FeeRecipientAddress:    deployConf.OptimismL2FeeRecipient,
		BatchInboxAddress:      deployConf.BatchInboxAddress,
		BatchSenderAddress:     deployConf.BatchSenderAddress,
		DepositContractAddress: predeploys.DevOptimismPortalAddr,
	}

	deploymentsL1 := DeploymentsL1{
		L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
		L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
		L2OutputOracleProxy:         predeploys.DevL2OutputOracleAddr,
		OptimismPortalProxy:         predeploys.DevOptimismPortalAddr,
	}

	deploymentsL2 := DeploymentsL2{
		L1Block: predeploys.L1BlockAddr,
	}

	return &SetupData{
		L1Cfg:     l1Genesis,
		L2Cfg:     l2Genesis,
		RollupCfg: rollupCfg,
		Deployments: Deployments{
			DeploymentsL1: deploymentsL1,
			DeploymentsL2: deploymentsL2,
		},
	}
}
