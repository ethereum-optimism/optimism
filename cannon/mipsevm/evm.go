package mipsevm

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/srcmap"
)

var (
	StepBytes4                      = crypto.Keccak256([]byte("Step(bytes,bytes)"))[:4]
	CheatBytes4                     = crypto.Keccak256([]byte("cheat(uint256,bytes32,bytes32,uint256)"))[:4]
	LoadKeccak256PreimagePartBytes4 = crypto.Keccak256([]byte("loadKeccak256PreimagePart(uint256,bytes)"))[:4]
)

// LoadContracts loads the Cannon contracts, from op-bindings package
func LoadContracts() (*Contracts, error) {
	var mips, oracle Contract
	mips.DeployedBytecode.Object = hexutil.MustDecode(bindings.MIPSDeployedBin)
	mips.DeployedBytecode.SourceMap = bindings.MIPSDeployedSourceMap
	oracle.DeployedBytecode.Object = hexutil.MustDecode(bindings.PreimageOracleDeployedBin)
	oracle.DeployedBytecode.SourceMap = bindings.PreimageOracleDeployedSourceMap
	return &Contracts{
		MIPS:   &mips,
		Oracle: &oracle,
	}, nil
}

// LoadContractsFromFiles loads the Cannon contracts, from local filesystem
func LoadContractsFromFiles() (*Contracts, error) {
	mips, err := LoadContract("MIPS")
	if err != nil {
		return nil, err
	}
	oracle, err := LoadContract("Oracle")
	if err != nil {
		return nil, err
	}
	return &Contracts{
		MIPS:   mips,
		Oracle: oracle,
	}, nil
}

func LoadContract(name string) (*Contract, error) {
	dat, err := os.ReadFile(fmt.Sprintf("../../packages/contracts-bedrock/forge-artifacts/%s.sol/%s.json", name, name))
	if err != nil {
		return nil, fmt.Errorf("failed to read contract JSON definition of %q: %w", name, err)
	}
	var out Contract
	if err := json.Unmarshal(dat, &out); err != nil {
		return nil, fmt.Errorf("failed to parse contract JSON definition of %q: %w", name, err)
	}
	return &out, nil
}

type Contract struct {
	DeployedBytecode struct {
		Object    hexutil.Bytes `json:"object"`
		SourceMap string        `json:"sourceMap"`
	} `json:"deployedBytecode"`

	// ignore abi,bytecode,etc.
}

func (c *Contract) SourceMap(sourcePaths []string) (*srcmap.SourceMap, error) {
	return srcmap.ParseSourceMap(sourcePaths, c.DeployedBytecode.Object, c.DeployedBytecode.SourceMap)
}

type Contracts struct {
	MIPS   *Contract
	Oracle *Contract
}

type Addresses struct {
	MIPS         common.Address
	Oracle       common.Address
	Sender       common.Address
	FeeRecipient common.Address
}

func NewEVMEnv(contracts *Contracts, addrs *Addresses) (*vm.EVM, *state.StateDB) {
	chainCfg := params.MainnetChainConfig
	offsetBlocks := uint64(1000) // blocks after shanghai fork
	bc := &testChain{startTime: *chainCfg.ShanghaiTime + offsetBlocks*12}
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
	env.StateDB.SetCode(addrs.MIPS, contracts.MIPS.DeployedBytecode.Object)
	env.StateDB.SetCode(addrs.Oracle, contracts.Oracle.DeployedBytecode.Object)
	env.StateDB.SetState(addrs.MIPS, common.Hash{}, addrs.Oracle.Hash())

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
