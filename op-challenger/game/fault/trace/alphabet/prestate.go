package alphabet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var absolutePrestate = common.FromHex("0000000000000000000000000000000000000000000000000000000000000060")
var absolutePrestateInt = new(big.Int).SetBytes(absolutePrestate)

var _ types.PrestateProvider = (*alphabetPrestateProvider)(nil)

// PrestateProvider provides the alphabet VM prestate
var PrestateProvider = &alphabetPrestateProvider{}

// alphabetPrestateProvider is a stateless [PrestateProvider] that
// uses a pre-determined, fixed pre-state hash.
type alphabetPrestateProvider struct{}

func (ap *alphabetPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	hash := common.BytesToHash(crypto.Keccak256(absolutePrestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}
