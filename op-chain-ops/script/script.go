package script

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// Prank represents an active prank task for the next sub-call.
type Prank struct {
	// Sender overrides msg.sender
	Sender *common.Address
	// Origin overrides tx.origin (set to actual origin if not part of the prank)
	Origin *common.Address
	// PrevOrigin is the tx.origin to restore after the prank
	PrevOrigin common.Address
	// Repeat is true if the prank persists after returning from a sub-call
	Repeat bool
	// A Prank may be a broadcast also.
	Broadcast bool
}

// CallFrame encodes the scope context of the current call
type CallFrame struct {
	Depth  int
	Opener vm.OpCode
	Ctx    *vm.ScopeContext

	// Prank overrides the msg.sender, and optionally the origin.
	// Forge script does not support nested pranks on the same call-depth.
	// Pranks can also be broadcasting.
	Prank *Prank
}

// Host is an EVM executor that runs Forge scripts.
type Host struct {
	log      log.Logger
	af       *foundry.ArtifactsFS
	chainCfg *params.ChainConfig
	env      *vm.EVM
	state    *state.StateDB
	stateDB  state.Database
	rawDB    ethdb.Database

	cheatcodes *Precompile[*CheatCodesPrecompile]
	console    *Precompile[*ConsolePrecompile]

	callStack []CallFrame

	// serializerStates are in-progress JSON payloads by name,
	// for the serializeX family of cheat codes, see:
	// https://book.getfoundry.sh/cheatcodes/serialize-json
	serializerStates map[string]json.RawMessage

	envVars map[string]string
	labels  map[common.Address]string
}

// NewHost creates a Host that can load contracts from the given Artifacts FS,
// and with an EVM initialized to the given executionContext.
func NewHost(logger log.Logger, fs *foundry.ArtifactsFS, executionContext Context) *Host {
	h := &Host{
		log:              logger,
		af:               fs,
		serializerStates: make(map[string]json.RawMessage),
		envVars:          make(map[string]string),
		labels:           make(map[common.Address]string),
	}

	// Init a default chain config, with all the mainnet L1 forks activated
	h.chainCfg = &params.ChainConfig{
		ChainID: executionContext.chainID,
		// Ethereum forks in proof-of-work era.
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		MergeNetsplitBlock:  big.NewInt(0),
		// Ethereum forks in proof-of-stake era.
		TerminalTotalDifficulty:       big.NewInt(1),
		TerminalTotalDifficultyPassed: true,
		ShanghaiTime:                  new(uint64),
		CancunTime:                    new(uint64),
		PragueTime:                    nil,
		VerkleTime:                    nil,
		// OP-Stack forks are disabled, since we use this for L1.
		BedrockBlock: nil,
		RegolithTime: nil,
		CanyonTime:   nil,
		EcotoneTime:  nil,
		FjordTime:    nil,
		GraniteTime:  nil,
		InteropTime:  nil,
		Optimism:     nil,
	}

	// Create an in-memory database, to host our temporary script state changes
	h.rawDB = rawdb.NewMemoryDatabase()
	h.stateDB = state.NewDatabaseWithConfig(h.rawDB, &triedb.Config{
		Preimages: true, // To be able to iterate the state we need the Preimages
		IsVerkle:  false,
		HashDB:    hashdb.Defaults,
		PathDB:    nil,
	})
	var err error
	h.state, err = state.New(types.EmptyRootHash, h.stateDB, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create memory state db: %w", err))
	}

	// Initialize a block-context for the EVM to access environment variables.
	// The block context (after embedding inside of the EVM environment) may be mutated later.
	blockContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash: func(n uint64) (out common.Hash) {
			binary.BigEndian.PutUint64(out[:8], n)
			return crypto.Keccak256Hash(out[:])
		},
		L1CostFunc:  nil,
		Coinbase:    executionContext.feeRecipient,
		GasLimit:    executionContext.gasLimit,
		BlockNumber: new(big.Int).SetUint64(executionContext.blockNum),
		Time:        executionContext.timestamp,
		Difficulty:  nil, // not used anymore post-merge
		BaseFee:     big.NewInt(0),
		BlobBaseFee: big.NewInt(0),
		Random:      &executionContext.prevRandao,
	}

	// Initialize a transaction-context for the EVM to access environment variables.
	// The transaction context (after embedding inside of the EVM environment) may be mutated later.
	txContext := vm.TxContext{
		Origin:       executionContext.origin,
		GasPrice:     big.NewInt(0),
		BlobHashes:   executionContext.blobHashes,
		BlobFeeCap:   big.NewInt(0),
		AccessEvents: state.NewAccessEvents(h.stateDB.PointCache()),
	}

	// Hook up the Host to capture the EVM environment changes
	trHooks := &tracing.Hooks{
		OnExit:          h.onExit,
		OnOpcode:        h.onOpcode,
		OnFault:         h.onFault,
		OnStorageChange: h.onStorageChange,
		OnLog:           h.onLog,
	}

	// Configure the EVM without basefee (because scripting), our trace hooks, and runtime precompile overrides.
	vmCfg := vm.Config{
		NoBaseFee:           true,
		Tracer:              trHooks,
		PrecompileOverrides: h.getPrecompile,
	}

	h.env = vm.NewEVM(blockContext, txContext, h.state, h.chainCfg, vmCfg)

	return h
}

// EnableCheats enables the Forge/HVM cheat-codes precompile and the Hardhat-style console2 precompile.
func (h *Host) EnableCheats() error {
	vmPrecompile, err := NewPrecompile[*CheatCodesPrecompile](&CheatCodesPrecompile{h: h})
	if err != nil {
		return fmt.Errorf("failed to init VM cheatcodes precompile: %w", err)
	}
	h.cheatcodes = vmPrecompile
	// Solidity does EXTCODESIZE checks on functions without return-data.
	// We need to insert some placeholder code to prevent it from aborting calls.
	// Emulates Forge script: https://github.com/foundry-rs/foundry/blob/224fe9cbf76084c176dabf7d3b2edab5df1ab818/crates/evm/evm/src/executors/mod.rs#L108
	h.state.SetCode(VMAddr, []byte{0x00})

	consolePrecompile, err := NewPrecompile[*ConsolePrecompile](&ConsolePrecompile{
		logger: h.log,
		sender: h.MsgSender,
	})
	if err != nil {
		return fmt.Errorf("failed to init console precompile: %w", err)
	}
	h.console = consolePrecompile
	// The Console precompile does not need bytecode,
	// calls all go through a console lib, which avoids the EXTCODESIZE.
	return nil
}

// prelude is a helper function to prepare the Host for a new call/create on the EVM environment.
func (h *Host) prelude(from common.Address, to *common.Address) {
	rules := h.chainCfg.Rules(h.env.Context.BlockNumber, true, h.env.Context.Time)
	activePrecompiles := vm.ActivePrecompiles(rules)
	h.env.StateDB.Prepare(rules, from, h.env.Context.Coinbase, to, activePrecompiles, nil)
}

// Call calls a contract in the EVM. The state changes persist.
func (h *Host) Call(from common.Address, to common.Address, input []byte, gas uint64, value *uint256.Int) (returnData []byte, leftOverGas uint64, err error) {
	h.prelude(from, &to)
	return h.env.Call(vm.AccountRef(from), to, input, gas, value)
}

// LoadContract loads the bytecode of a contract, and deploys it with regular CREATE.
func (h *Host) LoadContract(artifactName, contractName string) (common.Address, error) {
	artifact, err := h.af.ReadArtifact(artifactName, contractName)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to load %s / %s: %w", artifactName, contractName, err)
	}
	h.prelude(h.env.TxContext.Origin, nil)
	ret, addr, _, err := h.env.Create(vm.AccountRef(h.env.TxContext.Origin),
		artifact.Bytecode.Object, DefaultFoundryGasLimit, uint256.NewInt(0))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create contract, return: %x, err: %w", ret, err)
	}
	return addr, nil
}

// getPrecompile overrides any accounts during runtime, to insert special precompiles, if activated.
func (h *Host) getPrecompile(rules params.Rules, original vm.PrecompiledContract, addr common.Address) vm.PrecompiledContract {
	switch addr {
	case VMAddr:
		return h.cheatcodes // nil if cheats are not enabled
	case ConsoleAddr:
		return h.console // nil if cheats are not enabled
	default:
		return original
	}
}

// onExit is a trace-hook, which we use to maintain an accurate view of functions, and log any revert warnings.
func (h *Host) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// Note: onExit runs also when going deeper, exiting the context into a nested context.
	addr := h.SelfAddress()
	h.unwindCallstack(depth)
	if reverted {
		if msg, revertInspectErr := abi.UnpackRevert(output); revertInspectErr == nil {
			h.log.Warn("Revert", "addr", addr, "err", err, "revertMsg", msg)
		} else {
			h.log.Warn("Revert", "addr", addr, "err", err, "revertData", hexutil.Bytes(output))
		}
	}
}

// onFault is a trace-hook, catches things more generic than regular EVM reverts.
func (h *Host) onFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	h.log.Warn("Fault", "addr", scope.Address(), "err", err)
}

// unwindCallstack is a helper to remove call-stack entries.
func (h *Host) unwindCallstack(depth int) {
	// pop the callstack until the depth matches
	for len(h.callStack) > 0 && h.callStack[len(h.callStack)-1].Depth > depth {
		// unset the prank, if the parent call-frame had set up a prank that does not repeat
		if len(h.callStack) > 1 {
			parentCallFrame := h.callStack[len(h.callStack)-2]
			if parentCallFrame.Prank != nil {
				// While going back to the parent, restore the tx.origin.
				// It will later be re-applied on sub-calls if the prank persists (if Repeat == true).
				if parentCallFrame.Prank.Origin != nil {
					h.env.TxContext.Origin = parentCallFrame.Prank.PrevOrigin
				}
				if !parentCallFrame.Prank.Repeat {
					parentCallFrame.Prank = nil
				}
			}
		}
		// Now pop the call-frame
		h.callStack[len(h.callStack)-1] = CallFrame{} // don't hold on to the underlying call-frame resources
		h.callStack = h.callStack[:len(h.callStack)-1]
	}
}

// onOpcode is a trace-hook, used to maintain a view of the call-stack, and do any per op-code overrides.
func (h *Host) onOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	h.unwindCallstack(depth)
	scopeCtx := scope.(*vm.ScopeContext)
	// Check if we are entering a new depth, add it to the call-stack if so.
	// We do this here, instead of onEnter, to capture an initialized scope.
	if len(h.callStack) == 0 || h.callStack[len(h.callStack)-1].Depth < depth {
		h.callStack = append(h.callStack, CallFrame{
			Depth:  depth,
			Opener: vm.OpCode(op),
			Ctx:    scopeCtx,
		})
		// apply prank, if parent call-frame set up a prank
		if len(h.callStack) > 1 {
			parentCallFrame := h.callStack[len(h.callStack)-2]
			if parentCallFrame.Prank != nil {
				if parentCallFrame.Prank.Sender != nil {
					scopeCtx.Contract.CallerAddress = *parentCallFrame.Prank.Sender
				}
				if parentCallFrame.Prank.Origin != nil {
					h.env.TxContext.Origin = *parentCallFrame.Prank.Origin
				}
			}
		}
	}
	// Sanity check that top of the call-stack matches the scope context now
	if len(h.callStack) == 0 || h.callStack[len(h.callStack)-1].Ctx != scopeCtx {
		panic("scope context changed without call-frame pop/push")
	}
}

// onStorageChange is a trace-hook to capture state changes
func (h *Host) onStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash) {
	h.log.Debug("storage change", "addr", addr, "slot", slot, "prev_value", prev, "new_value", new)
	// future storage recording
}

// onLog is a trace-hook to capture log events
func (h *Host) onLog(ev *types.Log) {
	logger := h.log
	for i, topic := range ev.Topics {
		logger = logger.With(fmt.Sprintf("topic%d", i), topic)
	}
	logger.Debug("log event", "data", hexutil.Bytes(ev.Data))
	// future log recording
}

// CurrentCall returns the top of the callstack. Or zeroed if there was no call frame yet.
// If zeroed, the call-frame has a nil scope context.
func (h *Host) CurrentCall() CallFrame {
	if len(h.callStack) == 0 {
		return CallFrame{}
	}
	return h.callStack[len(h.callStack)-1]
}

// MsgSender returns the msg.sender of the current active EVM call-frame,
// or a zero address if there is no active call-frame.
func (h *Host) MsgSender() common.Address {
	cf := h.CurrentCall()
	if cf.Ctx == nil {
		return common.Address{}
	}
	return cf.Ctx.Caller()
}

// SelfAddress returns the current executing address of the current active EVM call-frame,
// or a zero address if there is no active call-frame.
func (h *Host) SelfAddress() common.Address {
	cf := h.CurrentCall()
	if cf.Ctx == nil {
		return common.Address{}
	}
	return cf.Ctx.Address()
}

// Prank applies a prank to the current call-frame.
// Any sub-call will apply the prank to their frame context.
func (h *Host) Prank(msgSender *common.Address, txOrigin *common.Address, repeat bool, broadcast bool) error {
	if len(h.callStack) == 0 {
		h.log.Warn("no call stack")
		return nil // cannot prank while not in a call.
	}
	cf := &h.callStack[len(h.callStack)-1]
	if cf.Prank != nil {
		if cf.Prank.Broadcast && !broadcast {
			return errors.New("you have an active broadcast; broadcasting and pranks are not compatible")
		}
		if !cf.Prank.Broadcast && broadcast {
			return errors.New("you have an active prank; broadcasting and pranks are not compatible")
		}
	}
	cf.Prank = &Prank{
		Sender:     msgSender,
		Origin:     txOrigin,
		PrevOrigin: h.env.TxContext.Origin,
		Repeat:     repeat,
		Broadcast:  broadcast,
	}
	return nil
}

// StopPrank disables the current prank. Any sub-call will not be pranked.
func (h *Host) StopPrank(broadcast bool) error {
	if len(h.callStack) == 0 {
		return nil
	}
	cf := &h.callStack[len(h.callStack)-1]
	if cf.Prank == nil {
		if broadcast {
			return errors.New("no broadcast in progress to stop")
		}
		return nil
	}
	if cf.Prank.Broadcast && !broadcast {
		// stopPrank on active broadcast is silent and no-op
		return nil
	}
	if !cf.Prank.Broadcast && broadcast {
		return errors.New("no broadcast in progress to stop")
	}
	cf.Prank = nil
	return nil
}

func (h *Host) CallerMode() CallerMode {
	if len(h.callStack) == 0 {
		return CallerModeNone
	}
	cf := &h.callStack[len(h.callStack)-1]
	if cf.Prank != nil {
		if cf.Prank.Broadcast {
			if cf.Prank.Repeat {
				return CallerModeRecurrentBroadcast
			}
			return CallerModeBroadcast
		}
		if cf.Prank.Repeat {
			return CallerModeRecurrentPrank
		}
		return CallerModePrank
	}
	return CallerModeNone
}

type CallerMode uint8

func (cm CallerMode) Big() *big.Int {
	return big.NewInt(int64(cm))
}

// CallerMode matches the CallerMode forge cheatcode enum.
const (
	CallerModeNone CallerMode = iota
	CallerModeBroadcast
	CallerModeRecurrentBroadcast
	CallerModePrank
	CallerModeRecurrentPrank
)

func (h *Host) GetEnvVar(key string) (value string, ok bool) {
	value, ok = h.envVars[key]
	return
}

func (h *Host) SetEnvVar(key string, value string) {
	h.envVars[key] = value
}
