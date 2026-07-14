package opcm

import (
	"github.com/ethereum/go-ethereum/common"
)

type SetDisputeGameImplInput struct {
	Factory             common.Address
	Impl                common.Address
	AnchorStateRegistry common.Address
	GameType            uint32
	GameArgs            []byte
}
