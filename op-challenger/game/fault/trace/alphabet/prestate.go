package alphabet

import (
	"context"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var absolutePrestate = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")

var _ types.PrestateProvider = (*AlphabetPrestateProvider)(nil)

// AlphabetPrestateProvider is a stateless [PrestateProvider] that
// uses a pre-determined, fixed pre-state hash.
type AlphabetPrestateProvider struct{}

func (ap *AlphabetPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	hash := common.BytesToHash(crypto.Keccak256(absolutePrestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}
