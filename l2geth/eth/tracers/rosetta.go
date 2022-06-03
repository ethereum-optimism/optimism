package tracers

import (
	"encoding/json"
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/common/hexutil"
	"github.com/ethereum-optimism/optimism/l2geth/core/vm"
	"github.com/ethereum-optimism/optimism/l2geth/log"
)

type contextType int

const (
	callContextType contextType = iota
	createContextType
)

var errExecutionReverted = errors.New("execution reverted")

type ctx struct {
	Type    contextType
	From    common.Address
	To      common.Address
	Input   []byte
	Gas     uint64
	Value   *big.Int
	Output  []byte
	GasUsed uint64
	Time    time.Duration
	Err     error
}

type call struct {
	Type    vm.OpCode
	From    common.Address
	To      common.Address
	Input   string
	Err     error
	Calls   []*call
	GasUsed uint64
	Value   *big.Int

	// ephemeral fields for accounting
	GasIn   uint64
	GasCost uint64
	Gas     uint64
	Output  []byte
	OutOff  int64
	OutLen  int64
}

type RosettaTracer struct {
	inited    bool   // Flag whether the context was already inited from the EVM
	err       error  // Error, if one has occurred
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // reason for the interruption
	ctx       ctx

	stk       []*call // the current recursive call stack of the EVM execution
	descended bool    // tracks whether we've just descended from aan outer transaction into an inner call
}

func NewRosettaTracer() *RosettaTracer {
	stk := make([]*call, 1)
	stk[0] = &call{}
	return &RosettaTracer{
		stk: stk,
	}
}

func (r *RosettaTracer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	r.ctx.Type = callContextType
	if create {
		r.ctx.Type = createContextType
	}
	r.ctx.From = from
	r.ctx.To = to
	r.ctx.Input = input
	r.ctx.Gas = gas
	r.ctx.Value = new(big.Int).Set(value)
	return nil
}

func (r *RosettaTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	if r.err == nil {
		// Initialize the context if it wasn't done yet
		if !r.inited {
			r.inited = true
		}
		// If tracing was interrupted, set the error and stop
		if atomic.LoadUint32(&r.interrupt) > 0 {
			r.err = r.reason
			return nil
		}

		if err != nil {
			r.captureFault(err)
			return nil
		}

		r.doStep(op, stack, memory, contract, env.StateDB, pc, gas, cost, depth, env.StateDB.GetRefund())
	}
	return nil
}

func (r *RosettaTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	if r.err == nil {
		r.captureFault(err)
	}
	return nil
}

func (r *RosettaTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	r.ctx.Output = output
	r.ctx.GasUsed = gasUsed
	r.ctx.Time = t
	r.ctx.Err = err
	return nil
}

func (r *RosettaTracer) Stop(err error) {
	r.reason = err
	atomic.StoreUint32(&r.interrupt, 1)
}

func (r *RosettaTracer) doStep(op vm.OpCode, stk *vm.Stack, mem *vm.Memory, contract *vm.Contract, state vm.StateDB, pc uint64, gas uint64, cost uint64, depth int, refund uint64) {
	switch op {
	case vm.CREATE, vm.CREATE2:
		inOff := peekVMStack(stk, 1).Int64()
		inEnd := inOff + peekVMStack(stk, 2).Int64()

		call := &call{
			Type:    op,
			From:    contract.Address(),
			Input:   hexutil.Encode(sliceVMMemory(mem, inOff, inEnd)),
			GasIn:   gas,
			GasCost: cost,
			Value:   new(big.Int).Set(peekVMStack(stk, 0)),
		}
		r.stk = append(r.stk, call)
		r.descended = true

	// If a contract is being self destructed, gather that as a subcall too
	case vm.SELFDESTRUCT:
		call := &call{
			Type:    op,
			From:    contract.Address(),
			To:      common.BigToAddress(peekVMStack(stk, 0)),
			GasIn:   gas,
			GasCost: cost,
			Value:   state.GetBalance(contract.Address()),
		}
		r.pushStack(call)

	// If a new method invocation si being done, add to the call stack
	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		// We don't skip any pre-compile invocations unlike the official
		// geth tracer. This can silence meaningful transfers.

		off := 0
		if op != vm.DELEGATECALL && op != vm.STATICCALL {
			off = 1
		}
		inOff := peekVMStack(stk, 2+off).Int64()
		inEnd := inOff + peekVMStack(stk, 3+off).Int64()

		call := &call{
			Type:    op,
			From:    contract.Address(),
			To:      common.BigToAddress(peekVMStack(stk, 1)),
			Input:   hexutil.Encode(sliceVMMemory(mem, inOff, inEnd)),
			GasIn:   gas,
			GasCost: cost,
			OutOff:  peekVMStack(stk, 4+off).Int64(),
			OutLen:  peekVMStack(stk, 5+off).Int64(),
		}
		if op != vm.DELEGATECALL && op != vm.STATICCALL {
			call.Value = new(big.Int).Set(peekVMStack(stk, 2))
		}
		r.stk = append(r.stk, call)
		r.descended = true

	default:
		// If we've just descended into an inner call, retrieve it's true allowance. We
		// need to extract if from within the call as there may be funky gas dynamics
		// with regard to requested and actually given gas (2300 stipend, 63/64 rule).
		if r.descended {
			if depth >= len(r.stk) {
				r.stk[len(r.stk)-1].Gas = gas
			} else {
				// TODO: The call was made to a plain account. We currently don't
				// have access to the true gas amount inside the call and so any amount will
				// mostly be wrong since it depends on a lot of input args. Skip gas for now.
			}
			r.descended = false
		}
		// If an existing call is returning, pop off the call stack
		if op == vm.REVERT {
			r.stk[len(r.stk)-1].Err = errExecutionReverted
		} else if depth == len(r.stk)-1 {
			// Pop off the last call and get the execution results
			call := r.popStack()
			if call.Type == vm.CREATE || call.Type == vm.CREATE2 {
				// If the call was a CREATE, retrieve the contract address and output code
				call.GasUsed = call.GasIn - call.GasCost - gas
				call.GasIn = 0
				call.GasCost = 0
				if ret := peekVMStack(stk, 0); ret.Int64() != 0 {
					call.To = common.BigToAddress(ret)
					call.Output = state.GetCode(call.To)
				} else if call.Err == nil {
					call.Err = errors.New("internal failure")
				}
			} else {
				// If the call was a contract call, retrieve the gas usage and output
				if call.Gas != 0 {
					call.GasUsed = call.GasIn - call.GasCost + call.Gas - gas
				}
				if ret := peekVMStack(stk, 0); ret.Int64() != 0 {
					call.Output = sliceVMMemory(mem, call.OutOff, call.OutOff+call.OutLen)
				} else if call.Err == nil {
					call.Err = errors.New("internal failure")
				}
				call.GasIn = 0
				call.GasCost = 0
				call.OutOff = 0
				call.OutLen = 0
			}
			// Inject the call into the previous one
			r.pushStack(call)
		}
	}
}

type Result struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Input   string         `json:"input"`
	Gas     hexutil.Uint64 `json:"gas"`
	Value   *hexutil.Big   `json:"value"`
	Output  string         `json:"output"`
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Time    time.Duration  `json:"time"`
	Err     string         `json:"error"`
	Calls   []*call        `json:"calls"`
}

func (r *RosettaTracer) GetResult() (json.RawMessage, error) {
	typ := "CALL"
	if r.ctx.Type == createContextType {
		typ = "CREATE"
	}
	result := Result{
		Type:    typ,
		From:    r.ctx.From,
		To:      r.ctx.To,
		Value:   (*hexutil.Big)(r.ctx.Value),
		Gas:     hexutil.Uint64(r.ctx.Gas),
		GasUsed: hexutil.Uint64(r.ctx.GasUsed),
		Input:   hexutil.Encode(r.ctx.Input),
		Output:  hexutil.Encode(r.ctx.Output),
		Time:    r.ctx.Time,
	}
	if r.stk[0].Calls != nil {
		// TODO: omitempty nil fields in Calls to save bandwidth
		result.Calls = r.stk[0].Calls
	}
	var err error
	if r.stk[0].Err != nil {
		err = r.stk[0].Err
	} else if r.ctx.Err != nil {
		err = r.ctx.Err
	}
	if err != nil && (errors.Is(err, errExecutionReverted) || result.Output == "0x") {
		result.Output = ""
	}
	if err != nil {
		result.Err = err.Error()
	}

	return json.Marshal(&result)
}

func (r *RosettaTracer) captureFault(err error) {
	// If the topmost call already reverted, don't handle the additional fault again
	if top := r.stk[len(r.stk)-1]; top.Err != nil {
		return
	}
	// Pop off the just failed call
	call := r.popStack()
	call.Err = err

	// Consume all available gas and clean any leftovers
	if call.Gas != 0 {
		call.GasUsed = call.Gas
	}
	call.GasIn = 0 // TOOD: set to nil
	call.GasCost = 0
	call.OutOff = 0

	// Flatten the failed call into its parent
	if len(r.stk) > 0 {
		r.pushStack(call)
		return
	}

	// Last call failed too, leave it in the stack
	r.stk = append(r.stk, call)
}

func (r *RosettaTracer) popStack() *call {
	c := r.stk[len(r.stk)-1]
	r.stk = r.stk[:len(r.stk)-1]
	return c
}

func (r *RosettaTracer) pushStack(call *call) {
	left := len(r.stk)
	calls := r.stk[left-1].Calls
	calls = append(calls, call)
	r.stk[left-1].Calls = calls
}

func peekVMStack(stack *vm.Stack, idx int) *big.Int {
	if len(stack.Data()) <= idx {
		// TODO: panic here instead
		log.Warn("Tracer accessed out of bound stack", "size", len(stack.Data()), "index", idx)
		return new(big.Int)
	}
	return stack.Data()[len(stack.Data())-idx-1]
}

func sliceVMMemory(memory *vm.Memory, begin, end int64) []byte {
	if memory.Len() < int(end) {
		log.Warn("Tracer accessed out of bound memory", "available", memory.Len(), "offset", begin, "size", end-begin)
		return nil
	}
	return memory.GetCopy(begin, end-begin)
}

type callmarshal struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from,omitempty"`
	To      common.Address `json:"to,omitempty"`
	Input   string         `json:"input,omitempty"`
	Err     string         `json:"error,omitempty"`
	Calls   []*callmarshal `json:"calls,omitempty"`
	GasUsed hexutil.Uint64 `json:"gasUsed,omitempty"`
	Value   *hexutil.Big   `json:"value,omitempty"`
}

func makecallmarshal(c *call) *callmarshal {
	cm := callmarshal{
		Type:    c.Type.String(),
		From:    c.From,
		To:      c.To,
		Input:   c.Input,
		GasUsed: hexutil.Uint64(c.GasUsed),
		Value:   (*hexutil.Big)(c.Value),
	}
	if c.Err != nil {
		cm.Err = c.Err.Error()
	}
	for _, x := range c.Calls {
		cm.Calls = append(cm.Calls, makecallmarshal(x))
	}
	return &cm
}

func (c *call) MarshalJSON() ([]byte, error) {
	enc := makecallmarshal(c)
	return json.Marshal(&enc)
}
