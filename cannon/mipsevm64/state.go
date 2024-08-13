package mipsevm64

type CpuScalars struct {
	PC     uint64 `json:"pc"`
	NextPC uint64 `json:"nextPC"`
	LO     uint64 `json:"lo"`
	HI     uint64 `json:"hi"`
}
