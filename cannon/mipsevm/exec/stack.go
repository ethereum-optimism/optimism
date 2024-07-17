package exec

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

type StackTracker interface {
	PushStack(target uint32)
	PopStack()
}

type StackTrackerImpl struct {
	state mipsevm.FPVMState

	stack        []uint32
	caller       []uint32
	meta         *program.Metadata
	debugEnabled bool
}

func NewStackTracker(state mipsevm.FPVMState) *StackTrackerImpl {
	return &StackTrackerImpl{state: state}
}

func (s *StackTrackerImpl) InitDebug(meta *program.Metadata) error {
	if meta == nil {
		return errors.New("metadata is nil")
	}
	s.debugEnabled = true
	s.meta = meta
	return nil
}

func (s *StackTrackerImpl) PushStack(target uint32) {
	if !s.debugEnabled {
		return
	}
	s.stack = append(s.stack, target)
	s.caller = append(s.caller, s.state.GetPC())
}

func (s *StackTrackerImpl) PopStack() {
	if !s.debugEnabled {
		return
	}
	if len(s.stack) != 0 {
		fn := s.meta.LookupSymbol(s.state.GetPC())
		topFn := s.meta.LookupSymbol(s.stack[len(s.stack)-1])
		if fn != topFn {
			// most likely the function was inlined. Snap back to the last return.
			i := len(s.stack) - 1
			for ; i >= 0; i-- {
				if s.meta.LookupSymbol(s.stack[i]) == fn {
					s.stack = s.stack[:i]
					s.caller = s.caller[:i]
					break
				}
			}
		} else {
			s.stack = s.stack[:len(s.stack)-1]
			s.caller = s.caller[:len(s.caller)-1]
		}
	} else {
		fmt.Printf("ERROR: stack underflow at pc=%x. step=%d\n", s.state.GetPC(), s.state.GetStep())
	}
}

func (s *StackTrackerImpl) Traceback() {
	fmt.Printf("traceback at pc=%x. step=%d\n", s.state.GetPC(), s.state.GetStep())
	for i := len(s.stack) - 1; i >= 0; i-- {
		jumpAddr := s.stack[i]
		idx := len(s.stack) - i - 1
		fmt.Printf("\t%d %x in %s caller=%08x\n", idx, jumpAddr, s.meta.LookupSymbol(jumpAddr), s.caller[i])
	}
}
