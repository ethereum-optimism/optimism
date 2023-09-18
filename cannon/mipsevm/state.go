package mipsevm

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// StateWitnessSize is the size of the state witness encoding in bytes.
var StateWitnessSize = 226

type State struct {
	Memory *Memory `json:"memory"`

	// The pre-image oracle inter-process is not multi-plexed,
	// so the pre-image handling does not support multi-threading.
	// I.e. the pre-image oracle in Cannon is not thread-safe.
	PreimageKey    common.Hash `json:"preimageKey"`
	PreimageOffset uint32      `json:"preimageOffset"` // note that the offset includes the 8-byte length prefix

	Heap uint32 `json:"heap"` // to handle mmap growth

	ExitCode uint8 `json:"exit"`
	Exited   bool  `json:"exited"`

	Step uint64 `json:"step"`

	// Addr last requested to have futex-blocked threads woken up
	Wakeup uint32 `json:"wakeup"`

	CurrentThread uint32 `json:"currentThread"`
	// merkleized like a SSZ List[2**32]
	Threads []ThreadContext `json:"threads"`

	// TODO: include thread-context merkle branch in witness
	// TODO: include branch of last thread context, to append with
	// TODO: if we spawn lots of threads we may need to GC the old thread contexts

	// LastHint is optional metadata, and not part of the VM state itself.
	// It is used to remember the last pre-image hint,
	// so a VM can start from any state without fetching prior pre-images,
	// and instead just repeat the last hint on setup,
	// to make sure pre-image requests can be served.
	// The first 4 bytes are a uin32 length prefix.
	// Warning: the hint MAY NOT BE COMPLETE. I.e. this is buffered,
	// and should only be read when len(LastHint) > 4 && uint32(LastHint[:4]) >= len(LastHint[4:])
	LastHint hexutil.Bytes `json:"lastHint,omitempty"`
}

func (s *State) GetStep() uint64 { return s.Step }

func (s *State) Thread() *ThreadContext {
	if s.CurrentThread >= uint32(len(s.Threads)) {
		return nil
	}
	return &s.Threads[s.CurrentThread]
}

type ThreadContext struct {
	ThreadID uint32 `json:"threadID"`
	ExitCode uint8  `json:"exitCode"`
	Exited   bool   `json:"exited"`

	State *ThreadState `json:"state"` // null if thread has exited and entered zombie-state
}

type ThreadState struct {
	// futex(int32 *uaddr, int32 op, int32 val, struct timespec *timeout, int32 *uaddr2, int32 val3);
	//
	// The Go runtime only uses two futex ops:
	//
	//  FUTEX_WAIT_PRIVATE: futexsleep(addr *uint32, val uint32, ns int64):
	// 		wait for val, with relative timeout. -1 is forever.
	//
	//  FUTEX_WAKE_PRIVATE: futexwakeup(addr *uint32, cnt uint32):
	// 		wake up at most cnt threads, in practice caller always sets cnt to 1.
	//      Only a signal, note that the returned woken-up count is ignored,
	//      so we do not support that, and claim it is 0, which is still valid because there are no guarantees anyway.
	//
	// uaddr2 and val3 are not used by the Go runtime.

	// addr to inspect for futex waking.
	FutexAddr        uint32 `json:"futexAddr"` // -1 if not waiting
	FutexVal         uint32 // value to wait for
	FutexTimeoutStep uint64 // step counter value when futex times out
	// TODO timer functionality?

	PC     uint32 `json:"pc"`
	NextPC uint32 `json:"nextPC"`
	LO     uint32 `json:"lo"`
	HI     uint32 `json:"hi"`

	Registers [32]uint32 `json:"registers"`
}

func (s *State) VMStatus() uint8 {
	return vmStatus(s.Exited, s.ExitCode)
}

func (s *State) EncodeWitness() StateWitness {
	out := make([]byte, 0)
	memRoot := s.Memory.MerkleRoot()
	out = append(out, memRoot[:]...)
	out = append(out, s.PreimageKey[:]...)
	out = binary.BigEndian.AppendUint32(out, s.PreimageOffset)
	out = binary.BigEndian.AppendUint32(out, s.Heap)
	out = append(out, s.ExitCode)
	if s.Exited {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	out = binary.BigEndian.AppendUint64(out, s.Step)

	/* TODO: encode thread context
	out = binary.BigEndian.AppendUint32(out, s.PC)
	out = binary.BigEndian.AppendUint32(out, s.NextPC)
	out = binary.BigEndian.AppendUint32(out, s.LO)
	out = binary.BigEndian.AppendUint32(out, s.HI)
	for _, r := range s.Registers {
		out = binary.BigEndian.AppendUint32(out, r)
	}
	*/
	return out
}

type StateWitness []byte

const (
	VMStatusValid      = 0
	VMStatusInvalid    = 1
	VMStatusPanic      = 2
	VMStatusUnfinished = 3
)

func (sw StateWitness) StateHash() (common.Hash, error) {
	if len(sw) != 226 {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected 226", len(sw))
	}

	hash := crypto.Keccak256Hash(sw)
	offset := 32*2 + 4*6
	exitCode := sw[offset]
	exited := sw[offset+1]
	status := vmStatus(exited == 1, exitCode)
	hash[0] = status
	return hash, nil
}

func vmStatus(exited bool, exitCode uint8) uint8 {
	if !exited {
		return VMStatusUnfinished
	}

	switch exitCode {
	case 0:
		return VMStatusValid
	case 1:
		return VMStatusInvalid
	default:
		return VMStatusPanic
	}
}
