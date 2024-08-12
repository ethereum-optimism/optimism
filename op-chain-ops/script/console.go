package script

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ConsolePrecompile struct {
	logger log.Logger
	sender func() common.Address
}

func (c *ConsolePrecompile) log(ctx ...any) {
	sender := c.sender()

	// Log the sender, since the self-address is always equal to the ConsoleAddr
	c.logger.With("sender", sender).Info("console", ctx...)
}

//go:generate go run ./consolegen --abi-txt=console2.txt --out=console2_gen.go
