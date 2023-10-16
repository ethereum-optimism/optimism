package fault

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// MinimalFaultDisputeGameCaller is a minimal interface around [bindings.FaultDisputeGameCaller].
// This needs to be updated if the [bindings.FaultDisputeGameCaller] interface changes.
type MinimalFaultDisputeGameCaller interface {
	ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	}, error)
	Status(opts *bind.CallOpts) (uint8, error)
	ClaimDataLen(opts *bind.CallOpts) (*big.Int, error)
	MAXGAMEDEPTH(opts *bind.CallOpts) (*big.Int, error)
	ABSOLUTEPRESTATE(opts *bind.CallOpts) ([32]byte, error)
}

// loader pulls in fault dispute game claim data periodically and over subscriptions.
type loader struct {
	caller MinimalFaultDisputeGameCaller
}

// NewLoader creates a new [loader].
func NewLoader(caller MinimalFaultDisputeGameCaller) *loader {
	return &loader{
		caller: caller,
	}
}

// NewLoaderFromBindings creates a new [loader] from a [bindings.FaultDisputeGameCaller].
func NewLoaderFromBindings(fdgAddr common.Address, client bind.ContractCaller) (*loader, error) {
	caller, err := bindings.NewFaultDisputeGameCaller(fdgAddr, client)
	if err != nil {
		return nil, err
	}
	return NewLoader(caller), nil
}

// GetGameStatus returns the current game status.
func (l *loader) GetGameStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	status, err := l.caller.Status(&bind.CallOpts{Context: ctx})
	return gameTypes.GameStatus(status), err
}

// GetClaimCount returns the number of claims in the game.
func (l *loader) GetClaimCount(ctx context.Context) (uint64, error) {
	count, err := l.caller.ClaimDataLen(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return count.Uint64(), nil
}

// FetchGameDepth fetches the game depth from the fault dispute game.
func (l *loader) FetchGameDepth(ctx context.Context) (uint64, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	gameDepth, err := l.caller.MAXGAMEDEPTH(&callOpts)
	if err != nil {
		return 0, err
	}

	return gameDepth.Uint64(), nil
}

// fetchClaim fetches a single [Claim] with a hydrated parent.
func (l *loader) fetchClaim(ctx context.Context, arrIndex uint64) (types.Claim, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	fetchedClaim, err := l.caller.ClaimData(&callOpts, new(big.Int).SetUint64(arrIndex))
	if err != nil {
		return types.Claim{}, err
	}

	claim := types.Claim{
		ClaimData: types.ClaimData{
			Value:    fetchedClaim.Claim,
			Position: types.NewPositionFromGIndex(fetchedClaim.Position),
		},
		Countered:           fetchedClaim.Countered,
		Clock:               fetchedClaim.Clock.Uint64(),
		ContractIndex:       int(arrIndex),
		ParentContractIndex: int(fetchedClaim.ParentIndex),
	}

	return claim, nil
}

// FetchClaims fetches all claims from the fault dispute game.
func (l *loader) FetchClaims(ctx context.Context) ([]types.Claim, error) {
	// Get the current claim count.
	claimCount, err := l.caller.ClaimDataLen(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	// Fetch each claim and build a list.
	claimList := make([]types.Claim, claimCount.Uint64())
	for i := uint64(0); i < claimCount.Uint64(); i++ {
		claim, err := l.fetchClaim(ctx, i)
		if err != nil {
			return nil, err
		}
		claimList[i] = claim
	}

	return claimList, nil
}

// FetchAbsolutePrestateHash fetches the hashed absolute prestate from the fault dispute game.
func (l *loader) FetchAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	absolutePrestate, err := l.caller.ABSOLUTEPRESTATE(&callOpts)
	if err != nil {
		return common.Hash{}, err
	}

	return absolutePrestate, nil
}
