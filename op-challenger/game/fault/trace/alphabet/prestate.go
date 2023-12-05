package alphabet

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	absolutePrestate = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")
)

type AlphabetPrestateProvider struct{}

func NewPrestateProvider() *AlphabetPrestateProvider {
	return &AlphabetPrestateProvider{}
}

func (o *AlphabetPrestateProvider) GenesisOutputRoot(ctx context.Context) (hash common.Hash, err error) {
	return common.Hash{}, fmt.Errorf("alphabet does not have a genesis output root")
}

func (ap *AlphabetPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	hash := common.BytesToHash(crypto.Keccak256(absolutePrestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}
