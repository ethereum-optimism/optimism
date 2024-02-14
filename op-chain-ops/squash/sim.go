package squash

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
)

type staticChain struct {
	startTime uint64
	blockTime uint64
}

func (d *staticChain) Engine() consensus.Engine {
	return ethash.NewFullFaker()
}

func (d *staticChain) GetHeader(h common.Hash, n uint64) *types.Header {
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
		Time:            d.startTime + n*d.blockTime,
		Extra:           nil,
		MixDigest:       common.Hash{},
		Nonce:           types.BlockNonce{},
		BaseFee:         big.NewInt(7),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
	}
}

type simState struct {
	*state.MemoryStateDB
	snapshotIndex  int
	tempAccessList map[common.Address]map[common.Hash]struct{}
}

var _ vm.StateDB = (*simState)(nil)

func (db *simState) AddressInAccessList(addr common.Address) bool {
	_, ok := db.tempAccessList[addr]
	return ok
}

func (db *simState) AddLog(log *types.Log) {
	// no-op
}

func (db *simState) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	// return the latest state, instead of the pre-tx state.
	acc, ok := db.Genesis().Alloc[addr]
	if !ok {
		return common.Hash{}
	}
	return acc.Storage[hash]
}

func (db *simState) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	// things like the fee-vault-address get marked as warm
	m, ok := db.tempAccessList[addr]
	if !ok {
		m = make(map[common.Hash]struct{})
		db.tempAccessList[addr] = m
	}
	m[slot] = struct{}{}
}

func (db *simState) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	m, addressOk := db.tempAccessList[addr]
	if !addressOk {
		return false, false
	}
	_, slotOk = m[slot]
	return true, slotOk
}

func (db *simState) GetRefund() uint64 {
	return 0
}

func (db *simState) AddAddressToAccessList(addr common.Address) {
	if _, ok := db.tempAccessList[addr]; !ok {
		db.tempAccessList[addr] = make(map[common.Hash]struct{})
	}
}

func (db *simState) RevertToSnapshot(int) {
	panic("RevertToSnapshot not supported")
}

func (db *simState) Snapshot() int {
	db.snapshotIndex += 1
	return db.snapshotIndex
}

// SquashSim wraps an op-chain-ops MemporyStateDB,
// and applies EVM-messages as if they all exist in the same infinite EVM block.
// The result is squashing all the EVM execution changes into the state.
type SquashSim struct {
	chainConfig *params.ChainConfig
	state       *simState
	evm         *vm.EVM
	signer      types.Signer
}

// AddMessage processes a message on top of the chain-state that is squashed into a genesis state allocation.
func (sim *SquashSim) AddMessage(msg *core.Message) (res *core.ExecutionResult, err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("critical error: %v", rErr)
		}
	}()

	// reset access-list
	sim.state.tempAccessList = make(map[common.Address]map[common.Hash]struct{})

	gp := new(core.GasPool)
	gp.AddGas(30_000_000)

	rules := sim.evm.ChainConfig().Rules(sim.evm.Context.BlockNumber, true, sim.evm.Context.Time)
	sim.evm.StateDB.Prepare(rules, msg.From, predeploys.SequencerFeeVaultAddr, msg.To, vm.ActivePrecompiles(rules), msg.AccessList)
	if !sim.state.Exist(msg.From) {
		sim.state.CreateAccount(msg.From)
	}
	return core.ApplyMessage(sim.evm, msg, gp)
}

func (sim *SquashSim) BlockContext() *vm.BlockContext {
	return &sim.evm.Context
}

// AddUpgradeTxs traverses a list of encoded deposit transactions.
// These transactions should match what would be included in the live system upgrade.
// The resulting state changes are squashed together, such that the final state can then be used as genesis state.
func (sim *SquashSim) AddUpgradeTxs(txs []hexutil.Bytes) error {
	for i, otx := range txs {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(otx); err != nil {
			return fmt.Errorf("failed to decode upgrade tx %d: %w", i, err)
		}
		msg, err := core.TransactionToMessage(&tx, sim.signer, sim.BlockContext().BaseFee)
		if err != nil {
			return fmt.Errorf("failed to turn upgrade tx %d into message: %w", i, err)
		}
		if !msg.IsDepositTx {
			return fmt.Errorf("upgrade tx %d is not a depost", i)
		}
		if res, err := sim.AddMessage(msg); err != nil {
			return fmt.Errorf("invalid upgrade tx %d, EVM invocation failed: %w", i, err)
		} else {
			if res.Err != nil {
				return fmt.Errorf("failed to successfully execute upgrade tx %d: %w", i, err)
			}
		}
	}
	return nil
}

func NewSimulator(db *state.MemoryStateDB) *SquashSim {
	offsetBlocks := uint64(0)
	genesisTime := uint64(17_000_000)
	blockTime := uint64(2)
	bc := &staticChain{startTime: genesisTime, blockTime: blockTime}
	header := bc.GetHeader(common.Hash{}, genesisTime+offsetBlocks)
	chainCfg := db.Genesis().Config
	blockContext := core.NewEVMBlockContext(header, bc, nil, chainCfg, db)
	vmCfg := vm.Config{}
	signer := types.LatestSigner(db.Genesis().Config)
	simDB := &simState{MemoryStateDB: db}
	env := vm.NewEVM(blockContext, vm.TxContext{}, simDB, chainCfg, vmCfg)

	return &SquashSim{
		chainConfig: chainCfg,
		state:       simDB,
		evm:         env,
		signer:      signer,
	}
}
