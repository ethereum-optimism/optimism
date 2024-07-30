package multithreaded

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
)

// SERIALIZED_THREAD_SIZE is the size of a serialized ThreadState object
const SERIALIZED_THREAD_SIZE = 166

// THREAD_WITNESS_SIZE is the size of a thread witness encoded in bytes.
//
//	It consists of the active thread serialized and concatenated with the
//	32 byte hash onion of the active thread stack without the active thread
const THREAD_WITNESS_SIZE = SERIALIZED_THREAD_SIZE + 32

// The empty thread root - keccak256(bytes32(0) ++ bytes32(0))
var EmptyThreadsRoot common.Hash = common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")

type ThreadState struct {
	ThreadId         uint32             `json:"threadId"`
	ExitCode         uint8              `json:"exit"`
	Exited           bool               `json:"exited"`
	FutexAddr        uint32             `json:"futexAddr"`
	FutexVal         uint32             `json:"futexVal"`
	FutexTimeoutStep uint64             `json:"futexTimeoutStep"`
	Cpu              mipsevm.CpuScalars `json:"cpu"`
	Registers        [32]uint32         `json:"registers"`
}

func CreateEmptyThread() *ThreadState {
	initThreadId := uint32(0)
	return &ThreadState{
		ThreadId: initThreadId,
		ExitCode: 0,
		Exited:   false,
		Cpu: mipsevm.CpuScalars{
			PC:     0,
			NextPC: 4,
			LO:     0,
			HI:     0,
		},
		FutexAddr:        exec.FutexEmptyAddr,
		FutexVal:         0,
		FutexTimeoutStep: 0,
		Registers:        [32]uint32{},
	}
}

func (t *ThreadState) serializeThread() []byte {
	out := make([]byte, 0, SERIALIZED_THREAD_SIZE)

	out = binary.BigEndian.AppendUint32(out, t.ThreadId)
	out = append(out, t.ExitCode)
	out = mipsevm.AppendBoolToWitness(out, t.Exited)
	out = binary.BigEndian.AppendUint32(out, t.FutexAddr)
	out = binary.BigEndian.AppendUint32(out, t.FutexVal)
	out = binary.BigEndian.AppendUint64(out, t.FutexTimeoutStep)

	out = binary.BigEndian.AppendUint32(out, t.Cpu.PC)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.NextPC)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.LO)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.HI)

	for _, r := range t.Registers {
		out = binary.BigEndian.AppendUint32(out, r)
	}

	return out
}

func computeThreadRoot(prevStackRoot common.Hash, threadToPush *ThreadState) common.Hash {
	hashedThread := crypto.Keccak256Hash(threadToPush.serializeThread())

	var hashData []byte
	hashData = append(hashData, prevStackRoot[:]...)
	hashData = append(hashData, hashedThread[:]...)

	return crypto.Keccak256Hash(hashData)
}
