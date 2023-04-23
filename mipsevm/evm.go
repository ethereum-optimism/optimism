package main

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
)

var StepBytes4 = crypto.Keccak256Hash([]byte("Step(bytes32,bytes,bytes)")).Bytes()[:4]

func LoadContracts() (*Contracts, error) {
	mips, err := LoadContract("MIPS")
	if err != nil {
		return nil, err
	}
	mipsMem, err := LoadContract("MIPSMemory")
	if err != nil {
		return nil, err
	}
	challenge, err := LoadContract("Challenge")
	if err != nil {
		return nil, err
	}
	return &Contracts{
		MIPS:       mips,
		MIPSMemory: mipsMem,
		Challenge:  challenge,
	}, nil
}

func LoadContract(name string) (*Contract, error) {
	// TODO change to forge build output
	dat, err := os.ReadFile(fmt.Sprintf("../contracts/out/%s.sol/%s.json", name, name))
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

func (c *Contract) SourceMap(sourcePaths []string) (*SourceMap, error) {
	return ParseSourceMap(sourcePaths, c.DeployedBytecode.Object, c.DeployedBytecode.SourceMap)
}

type Contracts struct {
	MIPS       *Contract
	MIPSMemory *Contract
	Challenge  *Contract
}

type Addresses struct {
	MIPS       common.Address
	MIPSMemory common.Address
	Challenge  common.Address
}

func NewEVMEnv(contracts *Contracts, addrs *Addresses) (*vm.EVM, *state.StateDB) {
	chainCfg := params.MainnetChainConfig
	bc := &testChain{}
	header := bc.GetHeader(common.Hash{}, 100)
	db := rawdb.NewMemoryDatabase()
	statedb := state.NewDatabase(db)
	state, err := state.New(types.EmptyRootHash, statedb, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create memory state db: %w", err))
	}
	blockContext := core.NewEVMBlockContext(header, bc, nil)
	vmCfg := vm.Config{}

	env := vm.NewEVM(blockContext, vm.TxContext{}, state, chainCfg, vmCfg)
	// pre-deploy the contracts

	env.StateDB.SetCode(addrs.MIPS, contracts.MIPS.DeployedBytecode.Object)
	env.StateDB.SetCode(addrs.MIPSMemory, contracts.MIPSMemory.DeployedBytecode.Object)
	env.StateDB.SetCode(addrs.Challenge, contracts.Challenge.DeployedBytecode.Object)
	// TODO: any state to set, or immutables to replace, to link the contracts together?
	return env, state
}

type testChain struct {
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
		Time:            1337,
		Extra:           nil,
		MixDigest:       common.Hash{},
		Nonce:           types.BlockNonce{},
		BaseFee:         big.NewInt(7),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
	}
}

func Calldata(st *State, accessList [][32]byte) []byte {
	input := crypto.Keccak256Hash([]byte("Steps(bytes32,uint256)")).Bytes()[:4]
	input = append(input, common.BigToHash(common.Big0).Bytes()...)
	input = append(input, common.BigToHash(big.NewInt(int64(st.Step))).Bytes()...)

	return input
}
