package types

type Tx struct{ Mint int }

func (Tx) IsDeposit() bool { return false }

func NewIsthmusSigner() int { return 0 }

func NewBlock(header, config int) int { return header + config }

func Gas(limit int) int { return limit }

type ChainConfig struct{}

func (*ChainConfig) IsCanyon() bool { return true }

func Apply() (int, error) { return 0, nil }

func Call(gas uint64) uint64 { return gas }

type Payload struct{ AccessList, Withdrawals *[]byte }

func Gas2(gas uint64) uint64 { return gas }

type Hash [32]byte

type Header struct{ Root Hash }

type Block struct{ GasLimit, GasUsed uint64 }

func Limit() uint64 { return 0 }

func EmptyRoot() Hash { return Hash{} }
