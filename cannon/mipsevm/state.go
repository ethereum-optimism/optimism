package mipsevm

type CpuScalars struct {
	PC     uint32 `json:"pc"`
	NextPC uint32 `json:"nextPC"`
	LO     uint32 `json:"lo"`
	HI     uint32 `json:"hi"`
}

const (
	VMStatusValid      = 0
	VMStatusInvalid    = 1
	VMStatusPanic      = 2
	VMStatusUnfinished = 3
)

func VmStatus(exited bool, exitCode uint8) uint8 {
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
