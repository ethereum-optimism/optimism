package fault

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// ClaimFetcher is a minimal interface around [bindings.FaultDisputeGameCaller].
// This needs to be updated if the [bindings.FaultDisputeGameCaller] interface changes.
type ClaimFetcher interface {
	ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	}, error)
	ClaimDataLen(opts *bind.CallOpts) (*big.Int, error)
}

// Loader is a minimal interface for loading onchain [Claim] data.
type Loader interface {
	FetchClaims(ctx context.Context) ([]Claim, error)
}

// loader pulls in fault dispute game claim data periodically and over subscriptions.
type loader struct {
	claimFetcher ClaimFetcher
}

// NewLoader creates a new [loader].
func NewLoader(claimFetcher ClaimFetcher) *loader {
	return &loader{
		claimFetcher: claimFetcher,
	}
}

// fetchClaim fetches a single [Claim] with a hydrated parent.
func (l *loader) fetchClaim(ctx context.Context, arrIndex uint64) (Claim, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	fetchedClaim, err := l.claimFetcher.ClaimData(&callOpts, new(big.Int).SetUint64(arrIndex))
	if err != nil {
		return Claim{}, err
	}

	claim := Claim{
		ClaimData: ClaimData{
			Value:    fetchedClaim.Claim,
			Position: NewPositionFromGIndex(fetchedClaim.Position.Uint64()),
		},
		Countered:           fetchedClaim.Countered,
		Clock:               fetchedClaim.Clock.Uint64(),
		ContractIndex:       int(arrIndex),
		ParentContractIndex: int(fetchedClaim.ParentIndex),
	}

	if !claim.IsRootPosition() {
		parentIndex := uint64(fetchedClaim.ParentIndex)
		parentClaim, err := l.claimFetcher.ClaimData(&callOpts, new(big.Int).SetUint64(parentIndex))
		if err != nil {
			return Claim{}, err
		}
		claim.Parent = ClaimData{
			Value:    parentClaim.Claim,
			Position: NewPositionFromGIndex(parentClaim.Position.Uint64()),
		}
	}

	return claim, nil
}

// FetchClaims fetches all claims from the fault dispute game.
func (l *loader) FetchClaims(ctx context.Context) ([]Claim, error) {
	// Get the current claim count.
	claimCount, err := l.claimFetcher.ClaimDataLen(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	// Fetch each claim and build a list.
	claimList := make([]Claim, claimCount.Uint64())
	for i := uint64(0); i < claimCount.Uint64(); i++ {
		claim, err := l.fetchClaim(ctx, i)
		if err != nil {
			return nil, err
		}
		claimList[i] = claim
	}

	return claimList, nil
}
