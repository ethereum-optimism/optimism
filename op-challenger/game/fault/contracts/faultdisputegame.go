package contracts

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var (
	methodGameDuration        = "gameDuration"
	methodMaxGameDepth        = "maxGameDepth"
	methodAbsolutePrestate    = "absolutePrestate"
	methodStatus              = "status"
	methodRootClaim           = "rootClaim"
	methodClaimCount          = "claimDataLen"
	methodClaim               = "claimData"
	methodL1Head              = "l1Head"
	methodResolve             = "resolve"
	methodResolveClaim        = "resolveClaim"
	methodAttack              = "attack"
	methodDefend              = "defend"
	methodStep                = "step"
	methodAddLocalData        = "addLocalData"
	methodVM                  = "vm"
	methodStartingBlockNumber = "startingBlockNumber"
	methodStartingRootHash    = "startingRootHash"
	methodSplitDepth          = "splitDepth"
	methodL2BlockNumber       = "l2BlockNumber"
	methodRequiredBond        = "getRequiredBond"
	methodClaimCredit         = "claimCredit"
	methodCredit              = "credit"
	methodWETH                = "weth"
)

type FaultDisputeGameContract struct {
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

type Proposal struct {
	L2BlockNumber *big.Int
	OutputRoot    common.Hash
}

func NewFaultDisputeGameContract(addr common.Address, caller *batching.MultiCaller) (*FaultDisputeGameContract, error) {
	contractAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load fault dispute game ABI: %w", err)
	}

	return &FaultDisputeGameContract{
		multiCaller: caller,
		contract:    batching.NewBoundContract(contractAbi, addr),
	}, nil
}

// GetBalance returns the total amount of ETH controlled by this contract.
// Note that the ETH is actually held by the DelayedWETH contract which may be shared by multiple games.
// Returns the balance and the address of the contract that actually holds the balance.
func (f *FaultDisputeGameContract) GetBalance(ctx context.Context, block rpcblock.Block) (*big.Int, common.Address, error) {
	result, err := f.multiCaller.SingleCall(ctx, block, f.contract.Call(methodWETH))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to load weth address: %w", err)
	}
	wethAddr := result.GetAddress(0)
	result, err = f.multiCaller.SingleCall(ctx, block, batching.NewBalanceCall(wethAddr))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to retrieve game balance: %w", err)
	}
	return result.GetBigInt(0), wethAddr, nil
}

// GetBlockRange returns the block numbers of the absolute pre-state block (typically genesis or the bedrock activation block)
// and the post-state block (that the proposed output root is for).
func (f *FaultDisputeGameContract) GetBlockRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error) {
	results, err := f.multiCaller.Call(ctx, rpcblock.Latest,
		f.contract.Call(methodStartingBlockNumber),
		f.contract.Call(methodL2BlockNumber))
	if err != nil {
		retErr = fmt.Errorf("failed to retrieve game block range: %w", err)
		return
	}
	if len(results) != 2 {
		retErr = fmt.Errorf("expected 2 results but got %v", len(results))
		return
	}
	prestateBlock = results[0].GetBigInt(0).Uint64()
	poststateBlock = results[1].GetBigInt(0).Uint64()
	return
}

// GetGameMetadata returns the game's L1 head, L2 block number, root claim, status, and game duration.
func (f *FaultDisputeGameContract) GetGameMetadata(ctx context.Context, block rpcblock.Block) (common.Hash, uint64, common.Hash, gameTypes.GameStatus, uint64, error) {
	results, err := f.multiCaller.Call(ctx, block,
		f.contract.Call(methodL1Head),
		f.contract.Call(methodL2BlockNumber),
		f.contract.Call(methodRootClaim),
		f.contract.Call(methodStatus),
		f.contract.Call(methodGameDuration))
	if err != nil {
		return common.Hash{}, 0, common.Hash{}, 0, 0, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 5 {
		return common.Hash{}, 0, common.Hash{}, 0, 0, fmt.Errorf("expected 3 results but got %v", len(results))
	}
	l1Head := results[0].GetHash(0)
	l2BlockNumber := results[1].GetBigInt(0).Uint64()
	rootClaim := results[2].GetHash(0)
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return common.Hash{}, 0, common.Hash{}, 0, 0, fmt.Errorf("failed to convert game status: %w", err)
	}
	duration := results[4].GetUint64(0)
	return l1Head, l2BlockNumber, rootClaim, status, duration, nil
}

func (f *FaultDisputeGameContract) GetStartingRootHash(ctx context.Context) (common.Hash, error) {
	startingRootHash, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodStartingRootHash))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve genesis output root: %w", err)
	}
	return startingRootHash.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetSplitDepth(ctx context.Context) (types.Depth, error) {
	splitDepth, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodSplitDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	return types.Depth(splitDepth.GetBigInt(0).Uint64()), nil
}

func (f *FaultDisputeGameContract) GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error) {
	results, err := f.multiCaller.Call(ctx, rpcblock.Latest,
		f.contract.Call(methodCredit, recipient),
		f.contract.Call(methodStatus))
	if err != nil {
		return nil, gameTypes.GameStatusInProgress, err
	}
	if len(results) != 2 {
		return nil, gameTypes.GameStatusInProgress, fmt.Errorf("expected 2 results but got %v", len(results))
	}
	credit := results[0].GetBigInt(0)
	status, err := gameTypes.GameStatusFromUint8(results[1].GetUint8(0))
	if err != nil {
		return nil, gameTypes.GameStatusInProgress, fmt.Errorf("invalid game status %v: %w", status, err)
	}
	return credit, status, nil
}

func (f *FaultDisputeGameContract) GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	calls := make([]batching.Call, 0, len(recipients))
	for _, recipient := range recipients {
		calls = append(calls, f.contract.Call(methodCredit, recipient))
	}
	results, err := f.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credit: %w", err)
	}
	credits := make([]*big.Int, 0, len(recipients))
	for _, result := range results {
		credits = append(credits, result.GetBigInt(0))
	}
	return credits, nil
}

func (f *FaultDisputeGameContract) ClaimCredit(recipient common.Address) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodClaimCredit, recipient)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) GetRequiredBond(ctx context.Context, position types.Position) (*big.Int, error) {
	bond, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodRequiredBond, position.ToGIndex()))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve required bond: %w", err)
	}
	return bond.GetBigInt(0), nil
}

func (f *FaultDisputeGameContract) UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	if data.IsLocal {
		return f.addLocalDataTx(claimIdx, data)
	}
	return f.addGlobalDataTx(ctx, data)
}

func (f *FaultDisputeGameContract) addLocalDataTx(claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	call := f.contract.Call(
		methodAddLocalData,
		data.GetIdent(),
		new(big.Int).SetUint64(claimIdx),
		new(big.Int).SetUint64(uint64(data.OracleOffset)),
	)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) addGlobalDataTx(ctx context.Context, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	oracle, err := f.GetOracle(ctx)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	return oracle.AddGlobalDataTx(data)
}

func (f *FaultDisputeGameContract) GetOracle(ctx context.Context) (*PreimageOracleContract, error) {
	vm, err := f.vm(ctx)
	if err != nil {
		return nil, err
	}
	return vm.Oracle(ctx)
}

func (f *FaultDisputeGameContract) GetGameDuration(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodGameDuration))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch game duration: %w", err)
	}
	return result.GetUint64(0), nil
}

func (f *FaultDisputeGameContract) GetMaxGameDepth(ctx context.Context) (types.Depth, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodMaxGameDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max game depth: %w", err)
	}
	return types.Depth(result.GetBigInt(0).Uint64()), nil
}

func (f *FaultDisputeGameContract) GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodAbsolutePrestate))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch absolute prestate hash: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetL1Head(ctx context.Context) (common.Hash, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (f *FaultDisputeGameContract) GetClaimCount(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodClaimCount))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch claim count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetClaim(ctx context.Context, idx uint64) (types.Claim, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodClaim, new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.Claim{}, fmt.Errorf("failed to fetch claim %v: %w", idx, err)
	}
	return f.decodeClaim(result, int(idx)), nil
}

func (f *FaultDisputeGameContract) GetAllClaims(ctx context.Context, block rpcblock.Block) ([]types.Claim, error) {
	results, err := batching.ReadArray(ctx, f.multiCaller, block, f.contract.Call(methodClaimCount), func(i *big.Int) *batching.ContractCall {
		return f.contract.Call(methodClaim, i)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load claims: %w", err)
	}

	var claims []types.Claim
	for idx, result := range results {
		claims = append(claims, f.decodeClaim(result, idx))
	}
	return claims, nil
}

func (f *FaultDisputeGameContract) vm(ctx context.Context) (*VMContract, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodVM))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VM addr: %w", err)
	}
	vmAddr := result.GetAddress(0)
	return NewVMContract(vmAddr, f.multiCaller)
}

func (f *FaultDisputeGameContract) AttackTx(parentContractIndex uint64, pivot common.Hash) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodAttack, new(big.Int).SetUint64(parentContractIndex), pivot)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) DefendTx(parentContractIndex uint64, pivot common.Hash) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodDefend, new(big.Int).SetUint64(parentContractIndex), pivot)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) StepTx(claimIdx uint64, isAttack bool, stateData []byte, proof []byte) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodStep, new(big.Int).SetUint64(claimIdx), isAttack, stateData, proof)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	call := f.resolveClaimCall(claimIdx)
	_, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return fmt.Errorf("failed to call resolve claim: %w", err)
	}
	return nil
}

func (f *FaultDisputeGameContract) ResolveClaimTx(claimIdx uint64) (txmgr.TxCandidate, error) {
	call := f.resolveClaimCall(claimIdx)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) resolveClaimCall(claimIdx uint64) *batching.ContractCall {
	return f.contract.Call(methodResolveClaim, new(big.Int).SetUint64(claimIdx))
}

func (f *FaultDisputeGameContract) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	call := f.resolveCall()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to call resolve: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (f *FaultDisputeGameContract) ResolveTx() (txmgr.TxCandidate, error) {
	call := f.resolveCall()
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) resolveCall() *batching.ContractCall {
	return f.contract.Call(methodResolve)
}

// decodeClock decodes a uint128 into a Clock duration and timestamp.
func decodeClock(clock *big.Int) *types.Clock {
	maxUint64 := new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(1))
	remainder := new(big.Int)
	quotient, _ := new(big.Int).QuoRem(clock, maxUint64, remainder)
	return types.NewClock(quotient.Uint64(), remainder.Uint64())
}

// packClock packs the Clock duration and timestamp into a uint128.
func packClock(c *types.Clock) *big.Int {
	duration := new(big.Int).SetUint64(c.Duration)
	encoded := new(big.Int).Lsh(duration, 64)
	return new(big.Int).Or(encoded, new(big.Int).SetUint64(c.Timestamp))
}

func (f *FaultDisputeGameContract) decodeClaim(result *batching.CallResult, contractIndex int) types.Claim {
	parentIndex := result.GetUint32(0)
	counteredBy := result.GetAddress(1)
	claimant := result.GetAddress(2)
	bond := result.GetBigInt(3)
	claim := result.GetHash(4)
	position := result.GetBigInt(5)
	clock := result.GetBigInt(6)
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    claim,
			Position: types.NewPositionFromGIndex(position),
			Bond:     bond,
		},
		CounteredBy:         counteredBy,
		Claimant:            claimant,
		Clock:               decodeClock(clock),
		ContractIndex:       contractIndex,
		ParentContractIndex: int(parentIndex),
	}
}
