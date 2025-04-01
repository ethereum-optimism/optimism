package stack

import (
	"github.com/ethereum/go-ethereum/log"
)

type Common interface {
	Logger() log.Logger
}
