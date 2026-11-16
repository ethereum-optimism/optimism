package types

type Tx struct{}

func NewBlock(header int) int { return header }

func Gas(limit string) int { return len(limit) }

type ChainConfig struct{}

func Apply() (int, int, error) { return 0, 0, nil }

type Budget struct{ Gas uint64 }

func Call(gas uint64) Budget { return Budget{gas} }

type Payload struct{ AccessList, Withdrawals []byte }

func Gas2(gas uint64) Budget { return Budget{gas} }

type Hash [32]byte

type Header struct{ Root *Hash }

type Block struct{ GasLimit, GasUsed uint64 }

func Limit() uint64 { return 0 }

func EmptyRoot() Hash { return Hash{} }
