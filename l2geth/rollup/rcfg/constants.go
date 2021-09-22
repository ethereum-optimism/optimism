package rcfg

import (
	"github.com/ethereum/go-ethereum/common"
)

// DefaultL1MessageSender is the default L1MessageSender value attached to a transaction that is
// not an L1 to L2 message.
var DefaultL1MessageSender = common.HexToAddress("0x00000000000000000000000000000000000beef")
