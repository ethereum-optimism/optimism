package script

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Prank represents an active prank task for the next sub-call.
// This is embedded into a call-frame, to then influence the sub-call through a caller-override.
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

// prankRef implements the vm.ContractRef interface, to mock a caller.
type prankRef struct {
	prank common.Address
	ref   vm.ContractRef
}

var _ vm.ContractRef = (*prankRef)(nil)

func (p *prankRef) Address() common.Address {
	return p.prank
}

// Value returns the value send into this contract context.
// The delegate call tracer implicitly relies on this being implemented on ContractRef
func (p *prankRef) Value() *uint256.Int {
	return p.ref.(interface{ Value() *uint256.Int }).Value()
}

func (h *Host) handleCaller(caller vm.ContractRef) vm.ContractRef {
	// apply prank, if top call-frame had set up a prank
	if len(h.callStack) > 0 {
		parentCallFrame := h.callStack[len(h.callStack)-1]
		if parentCallFrame.Prank != nil && caller.Address() != VMAddr { // pranks do not apply to the cheatcode precompile
			if parentCallFrame.Prank.Sender != nil {
				return &prankRef{
					prank: *parentCallFrame.Prank.Sender,
					ref:   caller,
				}
			}
			if parentCallFrame.Prank.Origin != nil {
				h.env.TxContext.Origin = *parentCallFrame.Prank.Origin
			}
		}
	}
	return caller
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
	if broadcast {
		h.log.Debug("starting broadcast", "sender", msgSender, "repeat", repeat)
	} else {
		h.log.Debug("starting prank", "sender", msgSender, "repeat", repeat)
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
	if broadcast {
		h.log.Debug("stopping broadcast")
	} else {
		h.log.Debug("stopping prank")
	}
	cf.Prank = nil
	return nil
}

// CallerMode returns the type of the top-most callframe,
// i.e. if we are in regular operation, a prank, or a broadcast (special kind of prank).
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

// CallerMode matches the CallerMode forge cheatcode enum.
type CallerMode uint8

func (cm CallerMode) Big() *big.Int {
	return big.NewInt(int64(cm))
}

const (
	CallerModeNone CallerMode = iota
	CallerModeBroadcast
	CallerModeRecurrentBroadcast
	CallerModePrank
	CallerModeRecurrentPrank
)
