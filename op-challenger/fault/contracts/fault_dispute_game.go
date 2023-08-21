package contracts

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// MinimalFaultDisputeGameCaller is a minimal interface around [bindings.FaultDisputeGameCaller].
// This needs to be updated if the [bindings.FaultDisputeGameCaller] interface changes.
type MinimalFaultDisputeGameCaller interface {
	Status(opts *bind.CallOpts) (uint8, error)
	ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	}, error)
	ClaimDataLen(opts *bind.CallOpts) (*big.Int, error)
	MAXGAMEDEPTH(opts *bind.CallOpts) (*big.Int, error)
	ABSOLUTEPRESTATE(opts *bind.CallOpts) ([32]byte, error)
}

// FaultDisputeGame provides an anti-corruption layer between the on-chain contracts and op-challenger bounded contexts.
// It improves the decoupling between these two contexts to make it easier to adapt to any future API changes.
// Additionally, it provides a simpler API for the rest of op-challenger to use with better typing of values.
type FaultDisputeGame struct {
	caller MinimalFaultDisputeGameCaller
}

func NewFaultDisputeGame(caller MinimalFaultDisputeGameCaller) *FaultDisputeGame {
	return &FaultDisputeGame{caller: caller}
}

func NewFaultDisputeGameAtAddress(addr common.Address, client bind.ContractCaller) (*FaultDisputeGame, error) {
	caller, err := bindings.NewFaultDisputeGameCaller(addr, client)
	if err != nil {
		return nil, err
	}
	return NewFaultDisputeGame(caller), nil
}

// FetchGameDepth fetches the game depth from the fault dispute game.
func (g *FaultDisputeGame) FetchGameDepth(ctx context.Context) (uint64, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	gameDepth, err := g.caller.MAXGAMEDEPTH(&callOpts)
	if err != nil {
		return 0, err
	}

	return gameDepth.Uint64(), nil
}

// fetchClaim fetches a single [Claim] with a hydrated parent.
func (g *FaultDisputeGame) fetchClaim(ctx context.Context, arrIndex uint64) (types.Claim, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	fetchedClaim, err := g.caller.ClaimData(&callOpts, new(big.Int).SetUint64(arrIndex))
	if err != nil {
		return types.Claim{}, err
	}

	claim := types.Claim{
		ClaimData: types.ClaimData{
			Value:    fetchedClaim.Claim,
			Position: types.NewPositionFromGIndex(fetchedClaim.Position.Uint64()),
		},
		Countered:           fetchedClaim.Countered,
		Clock:               fetchedClaim.Clock.Uint64(),
		ContractIndex:       int(arrIndex),
		ParentContractIndex: int(fetchedClaim.ParentIndex),
	}

	if !claim.IsRootPosition() {
		parentIndex := uint64(fetchedClaim.ParentIndex)
		parentClaim, err := g.caller.ClaimData(&callOpts, new(big.Int).SetUint64(parentIndex))
		if err != nil {
			return types.Claim{}, err
		}
		claim.Parent = types.ClaimData{
			Value:    parentClaim.Claim,
			Position: types.NewPositionFromGIndex(parentClaim.Position.Uint64()),
		}
	}

	return claim, nil
}

// FetchClaims fetches all claims from the fault dispute game.
func (g *FaultDisputeGame) FetchClaims(ctx context.Context) ([]types.Claim, error) {
	// Get the current claim count.
	claimCount, err := g.caller.ClaimDataLen(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	// Fetch each claim and build a list.
	claimList := make([]types.Claim, claimCount.Uint64())
	for i := uint64(0); i < claimCount.Uint64(); i++ {
		claim, err := g.fetchClaim(ctx, i)
		if err != nil {
			return nil, err
		}
		claimList[i] = claim
	}

	return claimList, nil
}

// FetchAbsolutePrestateHash fetches the hashed absolute prestate from the fault dispute game.
func (g *FaultDisputeGame) FetchAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	absolutePrestate, err := g.caller.ABSOLUTEPRESTATE(&callOpts)
	if err != nil {
		return common.Hash{}, err
	}
	return absolutePrestate, nil
}

// GetGameStatus returns the current game status.
func (g *FaultDisputeGame) GetGameStatus(ctx context.Context) (types.GameStatus, error) {
	status, err := g.caller.Status(&bind.CallOpts{Context: ctx})
	return types.GameStatus(status), err
}

// GetClaimCount returns the number of claims in the game.
func (g *FaultDisputeGame) GetClaimCount(ctx context.Context) (uint64, error) {
	count, err := g.caller.ClaimDataLen(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return count.Uint64(), nil
}
