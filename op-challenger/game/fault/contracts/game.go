package contracts

import (
	"context"
	"fmt"
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
	L1Head(opts *bind.CallOpts) ([32]byte, error)
	Proposals(opts *bind.CallOpts) (struct {
		Starting bindings.IFaultDisputeGameOutputProposal
		Disputed bindings.IFaultDisputeGameOutputProposal
	}, error)
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
	VM(opts *bind.CallOpts) (common.Address, error)
}

type FaultDisputeGame struct {
	caller     bind.ContractCaller
	gameCaller MinimalFaultDisputeGameCaller
}

func NewFaultDisputeGame(addr common.Address, caller bind.ContractCaller) (*FaultDisputeGame, error) {
	gameCaller, err := bindings.NewFaultDisputeGameCaller(addr, caller)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault dispute game contract: %w", err)
	}

	return &FaultDisputeGame{
		caller:     caller,
		gameCaller: gameCaller,
	}, nil
}

func (f *FaultDisputeGame) L1Head(ctx context.Context) (common.Hash, error) {
	return f.gameCaller.L1Head(&bind.CallOpts{Context: ctx})
}

func (f *FaultDisputeGame) Proposals(ctx context.Context) (
	agreedOutputRoot common.Hash,
	agreedBlockNumber *big.Int,
	disputedOutputRoot common.Hash,
	disputedBlockNumber *big.Int,
	err error) {
	proposals, err := f.gameCaller.Proposals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return common.Hash{}, nil, common.Hash{}, nil, fmt.Errorf("failed to retrieve proposals: %w", err)
	}
	return proposals.Starting.OutputRoot, proposals.Starting.L2BlockNumber, proposals.Disputed.OutputRoot, proposals.Disputed.L2BlockNumber, nil
}

func (f *FaultDisputeGame) PreimageOracleAddr(ctx context.Context) (common.Address, error) {
	opts := &bind.CallOpts{Context: ctx}
	vm, err := f.gameCaller.VM(opts)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to load VM address: %w", err)
	}
	mipsCaller, err := bindings.NewMIPSCaller(vm, f.caller)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create MIPS caller for address %v: %w", vm, err)
	}
	oracleAddr, err := mipsCaller.Oracle(opts)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to load pre-image oracle address: %w", err)
	}
	return oracleAddr, nil
}

// GetGameStatus returns the current game status.
func (f *FaultDisputeGame) GetGameStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	status, err := f.gameCaller.Status(&bind.CallOpts{Context: ctx})
	return gameTypes.GameStatus(status), err
}

// GetClaimCount returns the number of claims in the game.
func (f *FaultDisputeGame) GetClaimCount(ctx context.Context) (uint64, error) {
	count, err := f.gameCaller.ClaimDataLen(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return count.Uint64(), nil
}

// FetchGameDepth fetches the game depth from the fault dispute game.
func (f *FaultDisputeGame) FetchGameDepth(ctx context.Context) (uint64, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	gameDepth, err := f.gameCaller.MAXGAMEDEPTH(&callOpts)
	if err != nil {
		return 0, err
	}

	return gameDepth.Uint64(), nil
}

// fetchClaim fetches a single [Claim] with a hydrated parent.
func (f *FaultDisputeGame) fetchClaim(ctx context.Context, arrIndex uint64) (types.Claim, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	fetchedClaim, err := f.gameCaller.ClaimData(&callOpts, new(big.Int).SetUint64(arrIndex))
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

	if !claim.IsRootPosition() {
		parentIndex := uint64(fetchedClaim.ParentIndex)
		parentClaim, err := f.gameCaller.ClaimData(&callOpts, new(big.Int).SetUint64(parentIndex))
		if err != nil {
			return types.Claim{}, err
		}
		claim.Parent = types.ClaimData{
			Value:    parentClaim.Claim,
			Position: types.NewPositionFromGIndex(parentClaim.Position),
		}
	}

	return claim, nil
}

// FetchClaims fetches all claims from the fault dispute game.
func (f *FaultDisputeGame) FetchClaims(ctx context.Context) ([]types.Claim, error) {
	// Get the current claim count.
	claimCount, err := f.gameCaller.ClaimDataLen(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	// Fetch each claim and build a list.
	claimList := make([]types.Claim, claimCount.Uint64())
	for i := uint64(0); i < claimCount.Uint64(); i++ {
		claim, err := f.fetchClaim(ctx, i)
		if err != nil {
			return nil, err
		}
		claimList[i] = claim
	}

	return claimList, nil
}

// FetchAbsolutePrestateHash fetches the hashed absolute prestate from the fault dispute game.
func (f *FaultDisputeGame) FetchAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	callOpts := bind.CallOpts{
		Context: ctx,
	}

	absolutePrestate, err := f.gameCaller.ABSOLUTEPRESTATE(&callOpts)
	if err != nil {
		return common.Hash{}, err
	}

	return absolutePrestate, nil
}
