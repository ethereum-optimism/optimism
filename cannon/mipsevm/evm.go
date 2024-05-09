package mipsevm

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// LoadArtifacts loads the Cannon contracts, from the contracts package.
func LoadArtifacts() (*Artifacts, error) {
	mips, err := foundry.ReadArtifact("../../packages/contracts-bedrock/forge-artifacts/MIPS.sol/MIPS.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load MIPS contract: %w", err)
	}

	oracle, err := foundry.ReadArtifact("../../packages/contracts-bedrock/forge-artifacts/PreimageOracle.sol/PreimageOracle.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load Oracle contract: %w", err)
	}

	return &Artifacts{
		MIPS:   mips,
		Oracle: oracle,
	}, nil
}

type Artifacts struct {
	MIPS   *foundry.Artifact
	Oracle *foundry.Artifact
}

type Addresses struct {
	MIPS         common.Address
	Oracle       common.Address
	Sender       common.Address
	FeeRecipient common.Address
}

func NewEVMEnv(artifacts *Artifacts, addrs *Addresses) (*vm.EVM, *state.StateDB) {
	// Temporary hack until Cancun is activated on mainnet
	cpy := *params.MainnetChainConfig
	chainCfg := &cpy // don't modify the global chain config
	// Activate Cancun for EIP-4844 KZG point evaluation precompile
	cancunActivation := *chainCfg.ShanghaiTime + 10
	chainCfg.CancunTime = &cancunActivation
	offsetBlocks := uint64(1000) // blocks after cancun fork
	bc := &testChain{startTime: *chainCfg.CancunTime + offsetBlocks*12}
	header := bc.GetHeader(common.Hash{}, 17034870+offsetBlocks)
	db := rawdb.NewMemoryDatabase()
	statedb := state.NewDatabase(db)
	state, err := state.New(types.EmptyRootHash, statedb, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create memory state db: %w", err))
	}
	blockContext := core.NewEVMBlockContext(header, bc, nil, chainCfg, state)
	vmCfg := vm.Config{}

	env := vm.NewEVM(blockContext, vm.TxContext{}, state, chainCfg, vmCfg)
	// pre-deploy the contracts
	env.StateDB.SetCode(addrs.Oracle, artifacts.Oracle.DeployedBytecode.Object)

	var mipsCtorArgs [32]byte
	copy(mipsCtorArgs[12:], addrs.Oracle[:])
	mipsDeploy := append(hexutil.MustDecode(artifacts.MIPS.Bytecode.Object.String()), mipsCtorArgs[:]...)
	startingGas := uint64(30_000_000)
	_, deployedMipsAddr, leftOverGas, err := env.Create(vm.AccountRef(addrs.Sender), mipsDeploy, startingGas, common.U2560)
	if err != nil {
		panic(fmt.Errorf("failed to deploy MIPS contract: %w. took %d gas", err, startingGas-leftOverGas))
	}
	addrs.MIPS = deployedMipsAddr

	rules := env.ChainConfig().Rules(header.Number, true, header.Time)
	env.StateDB.Prepare(rules, addrs.Sender, addrs.FeeRecipient, &addrs.MIPS, vm.ActivePrecompiles(rules), nil)
	return env, state
}

type testChain struct {
	startTime uint64
}

func (d *testChain) Engine() consensus.Engine {
	return ethash.NewFullFaker()
}

func (d *testChain) GetHeader(h common.Hash, n uint64) *types.Header {
	parentHash := common.Hash{0: 0xff}
	binary.BigEndian.PutUint64(parentHash[1:], n-1)
	return &types.Header{
		ParentHash:      parentHash,
		UncleHash:       types.EmptyUncleHash,
		Coinbase:        common.Address{},
		Root:            common.Hash{},
		TxHash:          types.EmptyTxsHash,
		ReceiptHash:     types.EmptyReceiptsHash,
		Bloom:           types.Bloom{},
		Difficulty:      big.NewInt(0),
		Number:          new(big.Int).SetUint64(n),
		GasLimit:        30_000_000,
		GasUsed:         0,
		Time:            d.startTime + n*12,
		Extra:           nil,
		MixDigest:       common.Hash{},
		Nonce:           types.BlockNonce{},
		BaseFee:         big.NewInt(7),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
	}
}
