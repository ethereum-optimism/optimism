package types

type Tx struct{}

func NewBlock(header int) int { return header }

func Gas(limit string) int { return len(limit) }

type ChainConfig struct{}
