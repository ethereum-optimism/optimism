package types

type Tx struct{ Mint int }

func (Tx) IsDeposit() bool { return false }

func NewIsthmusSigner() int { return 0 }

func NewBlock(header, config int) int { return header + config }

func Gas(limit int) int { return limit }

type ChainConfig struct{}

func (*ChainConfig) IsCanyon() bool { return true }
