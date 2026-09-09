// Package atomic constructs witness-backed cross-chain calls without changing
// the EVM, CrossL2Inbox, or the interop verification rules.
package atomic

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Identifier uses Solidity's uint256 fields for ABI encoding.
type Identifier struct {
	Origin                                    common.Address
	BlockNumber, LogIndex, Timestamp, ChainId *big.Int
}

type ResultWitness struct {
	Identifier Identifier
	Success    bool
	ReturnData []byte
}

type RemoteCall struct {
	Identifier     Identifier
	Sequence       *big.Int
	Sender, Target common.Address
	Data           []byte
}

type Transaction struct {
	From, To   common.Address
	Data       []byte
	AccessList types.AccessList
	Gas        uint64
}

type Execution struct {
	Output   []byte
	Logs     []*types.Log
	Reverted bool
}

// Executor always executes from its pinned block-prefix state. Discovery may
// relax checksum warming, but must preserve the inbox's emitted logs. Replay
// must use the real bytecode and supplied access list. Neither method may
// mutate the pinned state.
type Executor interface {
	Discover(context.Context, Transaction) (Execution, error)
	Replay(context.Context, Transaction) (Execution, error)
}

type Chain struct {
	ID                     eth.ChainID
	BlockNumber, Timestamp uint64
	FirstLogIndex          uint32
	Sender                 common.Address
	Executor               Executor
}

type Builder struct {
	Router   common.Address
	ABI      abi.ABI
	Chains   map[eth.ChainID]Chain
	MaxCalls int
	Gas      uint64
}

type Plan struct {
	// Reverted plans contain only transactions that canonically revert.
	// Successful destination transactions are omitted; nonce and gas effects remain.
	Reverted       bool
	BundleID       common.Hash
	RootChain      eth.ChainID
	Transactions   map[eth.ChainID]Transaction
	Executions     map[eth.ChainID]Execution
	Witnesses      []ResultWitness
	Calls          map[eth.ChainID][]RemoteCall
	RootCompletion Identifier
}

// Build discovers sequential calls from the root to any number of other chains.
// Nested remote calls are rejected explicitly in this first implementation.
// Each destination's operations are replayed in one transaction, preserving its
// speculative writes between calls. A final replay authenticates every witness
// against the actual candidate logs for successful plans. A failed call instead
// returns an aborted plan whose included transactions all canonically revert.
func (b *Builder) Build(ctx context.Context, rootID eth.ChainID, nonce uint64, target common.Address, data []byte) (*Plan, error) {
	root, ok := b.Chains[rootID]
	if !ok || b.MaxCalls <= 0 || b.Gas == 0 {
		return nil, fmt.Errorf("root chain and positive call/gas bounds are required")
	}
	for id, chain := range b.Chains {
		if chain.ID != id || chain.Timestamp != root.Timestamp || chain.Executor == nil {
			return nil, fmt.Errorf("chain %s has inconsistent identity, timestamp, or executor", id)
		}
	}
	u256, _ := abi.NewType("uint256", "", nil)
	address, _ := abi.NewType("address", "", nil)
	identity, err := (abi.Arguments{{Type: u256}, {Type: address}, {Type: address}, {Type: u256}}).Pack(rootID.ToBig(), b.Router, root.Sender, new(big.Int).SetUint64(nonce))
	if err != nil {
		return nil, err
	}
	p := &Plan{BundleID: crypto.Keccak256Hash(identity), RootChain: rootID, Transactions: make(map[eth.ChainID]Transaction), Executions: make(map[eth.ChainID]Execution), Calls: make(map[eth.ChainID][]RemoteCall)}
	p.RootCompletion = b.identifier(root, 0)
	discovered := make(map[eth.ChainID]Execution)
	var rootExecution Execution
	var failedDestination *eth.ChainID
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tx, err := b.rootTx(root, nonce, target, data, p.Witnesses)
		if err != nil {
			return nil, err
		}
		rootExecution, err = root.Executor.Discover(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("discover root: %w", err)
		}
		if !rootExecution.Reverted {
			break
		}
		missing, ok := b.ABI.Errors["AtomicCallRouter_MissingWitness"]
		if !ok || len(rootExecution.Output) < 4 || !bytes.Equal(rootExecution.Output[:4], missing.ID[:4]) {
			if len(p.Witnesses) == 0 {
				return nil, fmt.Errorf("root operation reverted: 0x%x", rootExecution.Output)
			}
			discovered[rootID] = rootExecution
			return b.abort(ctx, p, tx, failedDestination, discovered)
		}
		if len(p.Witnesses) >= b.MaxCalls {
			return nil, fmt.Errorf("atomic bundle exceeds %d calls", b.MaxCalls)
		}
		args, err := missing.Inputs.Unpack(rootExecution.Output[4:])
		if err != nil {
			return nil, fmt.Errorf("decode missing witness: %w", err)
		}
		destination := eth.ChainIDFromBig(args[0].(*big.Int))
		chain, ok := b.Chains[destination]
		if !ok || destination == rootID {
			return nil, fmt.Errorf("unsupported destination %s", destination)
		}
		seq := args[4].(*big.Int)
		if !seq.IsUint64() || bigs.Uint64Strict(seq) != uint64(len(p.Witnesses)) {
			return nil, fmt.Errorf("non-sequential witness request")
		}
		call := RemoteCall{Identifier: b.identifier(root, 0), Sequence: seq, Target: args[1].(common.Address), Sender: args[2].(common.Address), Data: args[3].([]byte)}
		p.Calls[destination] = append(p.Calls[destination], call)
		remoteTx, err := b.remoteTx(chain, p.BundleID, p.Calls[destination], p.RootCompletion)
		if err != nil {
			return nil, err
		}
		remote, err := chain.Executor.Discover(ctx, remoteTx)
		if err != nil {
			return nil, fmt.Errorf("discover chain %s: %w", destination, err)
		}
		if remote.Reverted {
			failure := b.ABI.Errors["AtomicCallRouter_RemoteReverted"]
			if len(remote.Output) < 4 || !bytes.Equal(remote.Output[:4], failure.ID[:4]) {
				return nil, fmt.Errorf("remote router reverted: 0x%x", remote.Output)
			}
			args, err := failure.Inputs.Unpack(remote.Output[4:])
			if err != nil || args[0].(*big.Int).Cmp(seq) != 0 {
				return nil, fmt.Errorf("remote failure does not match current call")
			}
			reason := args[1].([]byte)
			missing := b.ABI.Errors["AtomicCallRouter_MissingWitness"]
			if len(reason) >= 4 && bytes.Equal(reason[:4], missing.ID[:4]) {
				return nil, fmt.Errorf("nested remote calls are not supported")
			}
			discovered[destination] = remote
			failedDestination = &destination
			p.Witnesses = append(p.Witnesses, ResultWitness{Identifier: b.identifier(chain, 0), ReturnData: reason})
			continue
		}
		discovered[destination] = remote
		decoded, err := b.ABI.Unpack("executeRemote", remote.Output)
		if err != nil {
			return nil, fmt.Errorf("decode remote results: %w", err)
		}
		results := decoded[0].([][]byte)
		if len(results) != len(p.Calls[destination]) {
			return nil, fmt.Errorf("remote result count mismatch")
		}
		for i, prior := range p.Calls[destination][:len(results)-1] {
			if !bytes.Equal(results[i], p.Witnesses[bigs.Uint64Strict(prior.Sequence)].ReturnData) {
				return nil, fmt.Errorf("remote prefix changed during discovery")
			}
		}
		callID := callIdentity(p.BundleID, rootID, bigs.Uint64Strict(seq))
		resultIndex, err := b.findLog(remote.Logs, "CallResult", callID)
		if err != nil {
			return nil, err
		}
		p.Witnesses = append(p.Witnesses, ResultWitness{Identifier: b.identifier(chain, resultIndex), Success: true, ReturnData: results[len(results)-1]})
	}
	if len(p.Witnesses) == 0 {
		return nil, fmt.Errorf("root did not make a remote call")
	}
	discovered[rootID] = rootExecution
	completionIndex, err := b.findLog(rootExecution.Logs, "BundleCompleted", p.BundleID)
	if err != nil {
		return nil, err
	}
	p.RootCompletion = b.identifier(root, completionIndex)
	for id, calls := range p.Calls {
		for i := range calls {
			index, err := b.findLog(rootExecution.Logs, "CallRequested", callIdentity(p.BundleID, rootID, bigs.Uint64Strict(calls[i].Sequence)))
			if err != nil {
				return nil, err
			}
			calls[i].Identifier = b.identifier(root, index)
		}
		p.Calls[id] = calls
	}
	rootTx, err := b.rootTx(root, nonce, target, data, p.Witnesses)
	if err != nil {
		return nil, err
	}
	rootTx.AccessList = b.rootAccessList(p)
	p.Transactions[rootID] = rootTx
	for id, calls := range p.Calls {
		chain := b.Chains[id]
		tx, err := b.remoteTx(chain, p.BundleID, calls, p.RootCompletion)
		if err != nil {
			return nil, err
		}
		var dependencies []messages.Message
		for _, call := range calls {
			idx := bigs.Uint64Strict(call.Identifier.LogIndex) - uint64(root.FirstLogIndex)
			dependencies = append(dependencies, message(call.Identifier, crypto.Keccak256Hash(messages.LogToMessagePayload(rootExecution.Logs[idx]))))
		}
		dependencies = append(dependencies, message(p.RootCompletion, crypto.Keccak256Hash(b.ABI.Events["BundleCompleted"].ID.Bytes(), p.BundleID.Bytes())))
		tx.AccessList = accessList(dependencies)
		p.Transactions[id] = tx
	}
	for _, id := range orderedChains(p.Transactions) {
		exec, err := b.Chains[id].Executor.Replay(ctx, p.Transactions[id])
		if err != nil {
			return nil, fmt.Errorf("replay chain %s: %w", id, err)
		}
		if exec.Reverted {
			return nil, fmt.Errorf("canonical replay reverted on chain %s: 0x%x", id, exec.Output)
		}
		prior := discovered[id]
		if !bytes.Equal(exec.Output, prior.Output) || len(exec.Logs) != len(prior.Logs) {
			return nil, fmt.Errorf("chain %s changed its result or log layout during canonical replay", id)
		}
		for i, log := range exec.Logs {
			old := prior.Logs[i]
			if log.Address != old.Address || (log.Address != predeploys.CrossL2InboxAddr && !bytes.Equal(messages.LogToMessagePayload(log), messages.LogToMessagePayload(old))) {
				return nil, fmt.Errorf("chain %s changed an application log during canonical replay", id)
			}
		}
		p.Executions[id] = exec
	}
	if err := b.Verify(p); err != nil {
		return nil, err
	}
	return p, nil
}

// abort includes A and, if applicable, the failed destination batch. Any other
// speculative successful chain is omitted. Failure hints have no authenticating
// source log: they are safe only because every included transaction must revert.
func (b *Builder) abort(ctx context.Context, p *Plan, rootTx Transaction, failed *eth.ChainID, discovered map[eth.ChainID]Execution) (*Plan, error) {
	p.Reverted = true
	rootTx.AccessList = b.rootAccessList(p)
	p.Transactions[p.RootChain] = rootTx
	if failed != nil {
		calls := p.Calls[*failed]
		tx, err := b.remoteTx(b.Chains[*failed], p.BundleID, calls, p.RootCompletion)
		if err != nil {
			return nil, err
		}
		u256, _ := abi.NewType("uint256", "", nil)
		address, _ := abi.NewType("address", "", nil)
		hash, _ := abi.NewType("bytes32", "", nil)
		var dependencies []messages.Message
		for _, call := range calls {
			encoded, err := (abi.Arguments{{Type: u256}, {Type: address}, {Type: address}, {Type: hash}}).Pack(failed.ToBig(), call.Target, call.Sender, crypto.Keccak256Hash(call.Data))
			if err != nil {
				return nil, err
			}
			payload := crypto.Keccak256Hash(b.ABI.Events["CallRequested"].ID.Bytes(), callIdentity(p.BundleID, p.RootChain, bigs.Uint64Strict(call.Sequence)).Bytes(), crypto.Keccak256(encoded))
			dependencies = append(dependencies, message(call.Identifier, payload))
		}
		tx.AccessList = accessList(dependencies)
		p.Transactions[*failed] = tx
	}
	for _, id := range orderedChains(p.Transactions) {
		execution, err := b.Chains[id].Executor.Replay(ctx, p.Transactions[id])
		if err != nil {
			return nil, fmt.Errorf("replay aborted chain %s: %w", id, err)
		}
		if !execution.Reverted || !bytes.Equal(execution.Output, discovered[id].Output) || len(execution.Logs) != 0 {
			return nil, fmt.Errorf("chain %s did not reproduce its abort", id)
		}
		p.Executions[id] = execution
	}
	if err := b.Verify(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (b *Builder) rootAccessList(p *Plan) types.AccessList {
	var dependencies []messages.Message
	for i, witness := range p.Witnesses {
		if witness.Success {
			dependencies = append(dependencies, message(witness.Identifier, crypto.Keccak256Hash(b.ABI.Events["CallResult"].ID.Bytes(), callIdentity(p.BundleID, p.RootChain, uint64(i)).Bytes(), crypto.Keccak256(witness.ReturnData))))
		}
	}
	return accessList(dependencies)
}

// Verify rejects missing, changed, mislocated, or misbound source messages in
// the fully replayed bundle. It complements, not replaces, normal block and
// interop verification when the payloads are submitted.
func (b *Builder) Verify(p *Plan) error {
	if len(p.Executions) != len(p.Transactions) || len(p.Transactions) == 0 {
		return fmt.Errorf("incomplete candidate bundle")
	}
	if _, ok := p.Transactions[p.RootChain]; !ok {
		return fmt.Errorf("missing root transaction")
	}
	if p.Reverted {
		for id := range p.Transactions {
			_, configured := b.Chains[id]
			execution, exists := p.Executions[id]
			if !configured || !exists || !execution.Reverted || len(execution.Logs) != 0 {
				return fmt.Errorf("aborted chain %s must revert without logs", id)
			}
		}
		return nil
	}
	if len(p.Transactions) < 2 {
		return fmt.Errorf("incomplete successful bundle")
	}
	for id := range p.Transactions {
		exec, exists := p.Executions[id]
		if !exists || exec.Reverted {
			return fmt.Errorf("chain %s did not execute successfully", id)
		}
	}
	for id, execution := range p.Executions {
		if uint64(b.Chains[id].FirstLogIndex)+uint64(len(execution.Logs)) > uint64(math.MaxUint32)+1 {
			return fmt.Errorf("chain %s log indexes overflow", id)
		}
		count := 0
		for _, entry := range execution.Logs {
			msg, err := messages.MessageFromLog(entry)
			if err != nil {
				return err
			}
			if msg == nil {
				continue
			}
			count++
			source, ok := b.Chains[msg.Identifier.ChainID]
			remote, exists := p.Executions[msg.Identifier.ChainID]
			index := uint64(msg.Identifier.LogIndex)
			if !ok || !exists || msg.Identifier.Timestamp != source.Timestamp || source.Timestamp != b.Chains[id].Timestamp || msg.Identifier.BlockNumber != source.BlockNumber || index < uint64(source.FirstLogIndex) || index-uint64(source.FirstLogIndex) >= uint64(len(remote.Logs)) {
				return fmt.Errorf("chain %s references a missing candidate message", id)
			}
			actual := remote.Logs[index-uint64(source.FirstLogIndex)]
			if actual.Address != msg.Identifier.Origin || crypto.Keccak256Hash(messages.LogToMessagePayload(actual)) != msg.PayloadHash {
				return fmt.Errorf("chain %s references a mismatched candidate message", id)
			}
		}
		expected := len(p.Calls[id]) + 1
		if id == p.RootChain {
			expected = len(p.Witnesses)
		}
		if count != expected {
			return fmt.Errorf("chain %s has %d executing messages, expected %d", id, count, expected)
		}
	}
	return nil
}

func (b *Builder) rootTx(chain Chain, nonce uint64, target common.Address, data []byte, witnesses []ResultWitness) (Transaction, error) {
	input, err := b.ABI.Pack("executeRoot", new(big.Int).SetUint64(nonce), target, data, witnesses)
	return Transaction{From: chain.Sender, To: b.Router, Data: input, Gas: b.Gas}, err
}

func (b *Builder) remoteTx(chain Chain, bundle common.Hash, calls []RemoteCall, completion Identifier) (Transaction, error) {
	input, err := b.ABI.Pack("executeRemote", bundle, calls, []ResultWitness{}, completion)
	return Transaction{From: chain.Sender, To: b.Router, Data: input, Gas: b.Gas}, err
}

func (b *Builder) identifier(chain Chain, relative uint32) Identifier {
	return Identifier{Origin: b.Router, BlockNumber: new(big.Int).SetUint64(chain.BlockNumber), Timestamp: new(big.Int).SetUint64(chain.Timestamp), LogIndex: new(big.Int).SetUint64(uint64(chain.FirstLogIndex) + uint64(relative)), ChainId: chain.ID.ToBig()}
}

func (b *Builder) findLog(logs []*types.Log, name string, id common.Hash) (uint32, error) {
	var found *uint32
	for i, log := range logs {
		if log.Address == b.Router && len(log.Topics) == 2 && log.Topics[0] == b.ABI.Events[name].ID && log.Topics[1] == id {
			if found != nil {
				return 0, fmt.Errorf("duplicate %s log", name)
			}
			index := uint32(i)
			found = &index
		}
	}
	if found == nil {
		return 0, fmt.Errorf("missing %s log", name)
	}
	return *found, nil
}

func callIdentity(bundle common.Hash, chain eth.ChainID, sequence uint64) common.Hash {
	chainBytes := chain.Bytes32()
	seq := common.BigToHash(new(big.Int).SetUint64(sequence))
	return crypto.Keccak256Hash(bundle[:], chainBytes[:], seq[:])
}

func message(id Identifier, payload common.Hash) messages.Message {
	return messages.Message{Identifier: messages.Identifier{Origin: id.Origin, BlockNumber: bigs.Uint64Strict(id.BlockNumber), LogIndex: uint32(bigs.Uint64Strict(id.LogIndex)), Timestamp: bigs.Uint64Strict(id.Timestamp), ChainID: eth.ChainIDFromBig(id.ChainId)}, PayloadHash: payload}
}

func accessList(msgs []messages.Message) types.AccessList {
	accesses := make([]messages.Access, len(msgs))
	for i := range msgs {
		accesses[i] = msgs[i].Access()
	}
	return types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: messages.EncodeAccessList(accesses)}}
}

func orderedChains(txs map[eth.ChainID]Transaction) []eth.ChainID {
	ids := make([]eth.ChainID, 0, len(txs))
	for id := range txs {
		ids = append(ids, id)
	}
	eth.SortChainID(ids)
	return ids
}
