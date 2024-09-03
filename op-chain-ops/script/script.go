package script

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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
	"github.com/ethereum-optimism/optimism/op-chain-ops/srcmap"
)

// CallFrame encodes the scope context of the current call
type CallFrame struct {
	Depth int

	LastOp vm.OpCode
	LastPC uint64

	// Reverts often happen in generated code.
	// We want to fallback to logging the source-map position of
	// the non-generated code, i.e. the origin of the last successful jump.
	LastJumpPC uint64

	Ctx *vm.ScopeContext

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

	precompiles map[common.Address]vm.PrecompiledContract

	callStack []CallFrame

	// serializerStates are in-progress JSON payloads by name,
	// for the serializeX family of cheat codes, see:
	// https://book.getfoundry.sh/cheatcodes/serialize-json
	serializerStates map[string]json.RawMessage

	envVars map[string]string
	labels  map[common.Address]string

	// srcFS enables src-map loading;
	// this is a bit more expensive, but provides useful debug information.
	// src-maps are disabled if this is nil.
	srcFS   *foundry.SourceMapFS
	srcMaps map[common.Address]*srcmap.SourceMap
}

// NewHost creates a Host that can load contracts from the given Artifacts FS,
// and with an EVM initialized to the given executionContext.
// Optionally src-map loading may be enabled, by providing a non-nil srcFS to read sources from.
func NewHost(logger log.Logger, fs *foundry.ArtifactsFS, srcFS *foundry.SourceMapFS, executionContext Context) *Host {
	h := &Host{
		log:              logger,
		af:               fs,
		serializerStates: make(map[string]json.RawMessage),
		envVars:          make(map[string]string),
		labels:           make(map[common.Address]string),
		precompiles:      make(map[common.Address]vm.PrecompiledContract),
		srcFS:            srcFS,
		srcMaps:          make(map[common.Address]*srcmap.SourceMap),
	}

	// Init a default chain config, with all the mainnet L1 forks activated
	h.chainCfg = &params.ChainConfig{
		ChainID: executionContext.ChainID,
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
		Coinbase:    executionContext.FeeRecipient,
		GasLimit:    executionContext.GasLimit,
		BlockNumber: new(big.Int).SetUint64(executionContext.BlockNum),
		Time:        executionContext.Timestamp,
		Difficulty:  nil, // not used anymore post-merge
		BaseFee:     big.NewInt(0),
		BlobBaseFee: big.NewInt(0),
		Random:      &executionContext.PrevRandao,
	}

	// Initialize a transaction-context for the EVM to access environment variables.
	// The transaction context (after embedding inside of the EVM environment) may be mutated later.
	txContext := vm.TxContext{
		Origin:       executionContext.Origin,
		GasPrice:     big.NewInt(0),
		BlobHashes:   executionContext.BlobHashes,
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
		CallerOverride:      h.handleCaller,
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
	h.precompiles[VMAddr] = h.cheatcodes

	consolePrecompile, err := NewPrecompile[*ConsolePrecompile](&ConsolePrecompile{
		logger: h.log,
		sender: h.MsgSender,
	})
	if err != nil {
		return fmt.Errorf("failed to init console precompile: %w", err)
	}
	h.console = consolePrecompile
	h.precompiles[ConsoleAddr] = h.console
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
	addr, err := h.Create(h.TxOrigin(), artifact.Bytecode.Object)
	if err != nil {
		return common.Address{}, err
	}
	h.RememberArtifact(addr, artifact, contractName)
	return addr, nil
}

// RememberArtifact registers an address as originating from a particular artifact.
// This register a source-map, if the Host is configured with a source-map FS.
func (h *Host) RememberArtifact(addr common.Address, artifact *foundry.Artifact, contract string) {
	if h.srcFS == nil {
		return
	}
	code := h.state.GetCode(addr)
	if !bytes.Equal(code, artifact.DeployedBytecode.Object) {
		h.log.Warn("src map warning: state bytecode does not match artifact deployed bytecode", "addr", addr)
	}

	srcMap, err := h.srcFS.SourceMap(artifact, contract)
	if err != nil {
		h.log.Warn("failed to load srcmap", "addr", addr, "err", err)
		return
	}
	h.srcMaps[addr] = srcMap
}

// Create a contract with unlimited gas, and 0 ETH value.
// This create function helps deploy contracts quickly for scripting etc.
func (h *Host) Create(from common.Address, initCode []byte) (common.Address, error) {
	h.prelude(from, nil)
	ret, addr, _, err := h.env.Create(vm.AccountRef(from),
		initCode, DefaultFoundryGasLimit, uint256.NewInt(0))
	if err != nil {
		retStr := fmt.Sprintf("%x", ret)
		if len(retStr) > 20 {
			retStr = retStr[:20] + "..."
		}
		return common.Address{}, fmt.Errorf("failed to create contract, return: %s, err: %w", retStr, err)
	}
	return addr, nil
}

// Wipe an account: removing the code, and setting address and balance to 0. This makes the account "empty".
// Note that storage is not removed.
func (h *Host) Wipe(addr common.Address) {
	if h.state.GetCodeSize(addr) > 0 {
		h.state.SetCode(addr, nil)
	}
	h.state.SetNonce(addr, 0)
	h.state.SetBalance(addr, uint256.NewInt(0), tracing.BalanceChangeUnspecified)
}

// getPrecompile overrides any accounts during runtime, to insert special precompiles, if activated.
func (h *Host) getPrecompile(rules params.Rules, original vm.PrecompiledContract, addr common.Address) vm.PrecompiledContract {
	if p, ok := h.precompiles[addr]; ok {
		return p
	}
	return original
}

// SetPrecompile inserts a precompile at the given address.
// If the precompile is nil, it removes the precompile override from that address, and wipes the account.
func (h *Host) SetPrecompile(addr common.Address, precompile vm.PrecompiledContract) {
	if precompile == nil {
		h.log.Debug("removing precompile", "addr", addr)
		delete(h.precompiles, addr)
		h.Wipe(addr)
		return
	}
	h.log.Debug("adding precompile", "addr", addr)
	h.precompiles[addr] = precompile
	// insert non-empty placeholder bytecode, so EXTCODESIZE checks pass
	h.state.SetCode(addr, []byte{0})
}

// HasPrecompileOverride inspects if there exists an active precompile-override at the given address.
func (h *Host) HasPrecompileOverride(addr common.Address) bool {
	_, ok := h.precompiles[addr]
	return ok
}

// onExit is a trace-hook, which we use to maintain an accurate view of functions, and log any revert warnings.
func (h *Host) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// Note: onExit runs also when going deeper, exiting the context into a nested context.
	addr := h.SelfAddress()
	if reverted {
		h.LogCallStack()
		if msg, revertInspectErr := abi.UnpackRevert(output); revertInspectErr == nil {
			h.log.Warn("Revert", "addr", addr, "err", err, "revertMsg", msg, "depth", depth)
		} else {
			h.log.Warn("Revert", "addr", addr, "err", err, "revertData", hexutil.Bytes(output), "depth", depth)
		}
	}
	h.unwindCallstack(depth)
}

// onFault is a trace-hook, catches things more generic than regular EVM reverts.
func (h *Host) onFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	h.log.Warn("Fault", "addr", scope.Address(), "err", err, "depth", depth)
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
			Depth:      depth,
			LastOp:     vm.OpCode(op),
			LastPC:     pc,
			LastJumpPC: pc,
			Ctx:        scopeCtx,
		})
	}
	// Sanity check that top of the call-stack matches the scope context now
	if len(h.callStack) == 0 || h.callStack[len(h.callStack)-1].Ctx != scopeCtx {
		panic("scope context changed without call-frame pop/push")
	}
	cf := &h.callStack[len(h.callStack)-1]
	if vm.OpCode(op) == vm.JUMPDEST { // remember the last PC before successful jump
		cf.LastJumpPC = cf.LastPC
	}
	cf.LastOp = vm.OpCode(op)
	cf.LastPC = pc
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

func (h *Host) GetEnvVar(key string) (value string, ok bool) {
	value, ok = h.envVars[key]
	return
}

func (h *Host) SetEnvVar(key string, value string) {
	h.envVars[key] = value
}

// StateDump turns the current EVM state into a foundry-allocs dump
// (wrapping a geth Account allocs type). This is used to export the state.
// Note that upon dumping, the state-DB is committed and flushed.
// This affects any remaining self-destructs, as all accounts are flushed to persistent state.
// After flushing the EVM state also cannot revert to a previous snapshot state:
// the state should not be dumped within contract-execution that needs to revert.
func (h *Host) StateDump() (*foundry.ForgeAllocs, error) {
	// We have to commit the existing state to the trie,
	// for all the state-changes to be captured by the trie iterator.
	root, err := h.state.Commit(h.env.Context.BlockNumber.Uint64(), true)
	if err != nil {
		return nil, fmt.Errorf("failed to commit state: %w", err)
	}
	// We need a state object around the state DB
	st, err := state.New(root, h.stateDB, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create state object for state-dumping: %w", err)
	}
	// After Commit we cannot reuse the old State, so we update the host to use the new one
	h.state = st
	h.env.StateDB = st

	var allocs foundry.ForgeAllocs
	allocs.FromState(st)

	// Sanity check we have no lingering scripts.
	for i := uint64(0); i <= allocs.Accounts[ScriptDeployer].Nonce; i++ {
		scriptAddr := crypto.CreateAddress(ScriptDeployer, i)
		h.log.Info("removing script from state-dump", "addr", scriptAddr, "label", h.labels[scriptAddr])
		delete(allocs.Accounts, scriptAddr)
	}

	// Remove the script deployer from the output
	delete(allocs.Accounts, ScriptDeployer)

	// The cheatcodes VM has a placeholder bytecode,
	// because solidity checks if the code exists prior to regular EVM-calls to it.
	delete(allocs.Accounts, VMAddr)

	// Precompile overrides come with temporary state account placeholders. Ignore those.
	for addr := range h.precompiles {
		delete(allocs.Accounts, addr)
	}

	return &allocs, nil
}

func (h *Host) SetTxOrigin(addr common.Address) {
	h.env.TxContext.Origin = addr
}

func (h *Host) TxOrigin() common.Address {
	return h.env.TxContext.Origin
}

// ScriptBackendFn is a convenience method for scripts to attach to the Host.
// It return a function pre-configured with the given destination-address,
// to call the destination script.
func (h *Host) ScriptBackendFn(to common.Address) CallBackendFn {
	return func(data []byte) ([]byte, error) {
		ret, _, err := h.Call(h.env.TxContext.Origin, to, data, DefaultFoundryGasLimit, uint256.NewInt(0))
		return ret, err
	}
}

// EnforceMaxCodeSize configures the EVM to enforce (if true), or not enforce (if false),
// the maximum contract bytecode size.
func (h *Host) EnforceMaxCodeSize(v bool) {
	h.env.Config.NoMaxCodeSize = !v
}

// LogCallStack is a convenience method for debugging,
// to log details of each call-frame (from bottom to top) to the logger.
func (h *Host) LogCallStack() {
	for _, cf := range h.callStack {
		callsite := ""
		if srcMap, ok := h.srcMaps[cf.Ctx.Address()]; ok {
			callsite = srcMap.FormattedInfo(cf.LastPC)
			if callsite == "unknown:0:0" {
				callsite = srcMap.FormattedInfo(cf.LastJumpPC)
			}
		}
		input := cf.Ctx.CallInput()
		byte4 := ""
		if len(input) >= 4 {
			byte4 = fmt.Sprintf("0x%x", input[:4])
		}
		h.log.Debug("callframe", "depth", cf.Depth, "input", hexutil.Bytes(input), "pc", cf.LastPC, "op", cf.LastOp)
		h.log.Warn("callframe", "depth", cf.Depth, "byte4", byte4,
			"addr", cf.Ctx.Address(), "callsite", callsite, "label", h.labels[cf.Ctx.Address()])
	}
}

// Label an address with a name, like the foundry vm.label cheatcode.
func (h *Host) Label(addr common.Address, label string) {
	h.log.Debug("labeling", "addr", addr, "label", label)
	h.labels[addr] = label
}
