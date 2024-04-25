package contracts

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

// The maximum number of children that will be processed during a call to `resolveClaim`
var maxChildChecks = big.NewInt(512)

var (
	methodMaxClockDuration    = "maxClockDuration"
	methodMaxGameDepth        = "maxGameDepth"
	methodAbsolutePrestate    = "absolutePrestate"
	methodStatus              = "status"
	methodRootClaim           = "rootClaim"
	methodClaimCount          = "claimDataLen"
	methodClaim               = "claimData"
	methodL1Head              = "l1Head"
	methodResolvedSubgames    = "resolvedSubgames"
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

var ErrSimulationFailed = errors.New("tx simulation failed")

type FaultDisputeGameContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

type Proposal struct {
	L2BlockNumber *big.Int
	OutputRoot    common.Hash
}

func NewFaultDisputeGameContract(metrics metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *FaultDisputeGameContract {
	contractAbi := snapshots.LoadFaultDisputeGameABI()

	return &FaultDisputeGameContract{
		metrics:     metrics,
		multiCaller: caller,
		contract:    batching.NewBoundContract(contractAbi, addr),
	}
}

// GetBalance returns the total amount of ETH controlled by this contract.
// Note that the ETH is actually held by the DelayedWETH contract which may be shared by multiple games.
// Returns the balance and the address of the contract that actually holds the balance.
func (f *FaultDisputeGameContract) GetBalance(ctx context.Context, block rpcblock.Block) (*big.Int, common.Address, error) {
	defer f.metrics.StartContractRequest("GetBalance")()
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
	defer f.metrics.StartContractRequest("GetBlockRange")()
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

// GetGameMetadata returns the game's L1 head, L2 block number, root claim, status, and max clock duration.
func (f *FaultDisputeGameContract) GetGameMetadata(ctx context.Context, block rpcblock.Block) (common.Hash, uint64, common.Hash, gameTypes.GameStatus, uint64, error) {
	defer f.metrics.StartContractRequest("GetGameMetadata")()
	results, err := f.multiCaller.Call(ctx, block,
		f.contract.Call(methodL1Head),
		f.contract.Call(methodL2BlockNumber),
		f.contract.Call(methodRootClaim),
		f.contract.Call(methodStatus),
		f.contract.Call(methodMaxClockDuration))
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
	defer f.metrics.StartContractRequest("GetStartingRootHash")()
	startingRootHash, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodStartingRootHash))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve genesis output root: %w", err)
	}
	return startingRootHash.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetSplitDepth(ctx context.Context) (types.Depth, error) {
	defer f.metrics.StartContractRequest("GetSplitDepth")()
	splitDepth, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodSplitDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	return types.Depth(splitDepth.GetBigInt(0).Uint64()), nil
}

func (f *FaultDisputeGameContract) GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error) {
	defer f.metrics.StartContractRequest("GetCredit")()
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

func (f *FaultDisputeGameContract) GetRequiredBonds(ctx context.Context, block rpcblock.Block, positions ...*big.Int) ([]*big.Int, error) {
	calls := make([]batching.Call, 0, len(positions))
	for _, position := range positions {
		calls = append(calls, f.contract.Call(methodRequiredBond, position))
	}
	results, err := f.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve required bonds: %w", err)
	}
	requiredBonds := make([]*big.Int, 0, len(positions))
	for _, result := range results {
		requiredBonds = append(requiredBonds, result.GetBigInt(0))
	}
	return requiredBonds, nil
}

func (f *FaultDisputeGameContract) GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	defer f.metrics.StartContractRequest("GetCredits")()
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

func (f *FaultDisputeGameContract) ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error) {
	defer f.metrics.StartContractRequest("ClaimCredit")()
	call := f.contract.Call(methodClaimCredit, recipient)
	_, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("%w: %v", ErrSimulationFailed, err.Error())
	}
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) GetRequiredBond(ctx context.Context, position types.Position) (*big.Int, error) {
	defer f.metrics.StartContractRequest("GetRequiredBond")()
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

func (f *FaultDisputeGameContract) GetWithdrawals(ctx context.Context, block rpcblock.Block, gameAddr common.Address, recipients ...common.Address) ([]*WithdrawalRequest, error) {
	defer f.metrics.StartContractRequest("GetWithdrawals")()
	delayedWETH, err := f.getDelayedWETH(ctx)
	if err != nil {
		return nil, err
	}
	return delayedWETH.GetWithdrawals(ctx, block, gameAddr, recipients...)
}

func (f *FaultDisputeGameContract) getDelayedWETH(ctx context.Context) (*DelayedWETHContract, error) {
	defer f.metrics.StartContractRequest("GetDelayedWETH")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodWETH))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WETH addr: %w", err)
	}
	return NewDelayedWETHContract(f.metrics, result.GetAddress(0), f.multiCaller), nil
}

func (f *FaultDisputeGameContract) GetOracle(ctx context.Context) (*PreimageOracleContract, error) {
	defer f.metrics.StartContractRequest("GetOracle")()
	vm, err := f.vm(ctx)
	if err != nil {
		return nil, err
	}
	return vm.Oracle(ctx)
}

func (f *FaultDisputeGameContract) GetMaxClockDuration(ctx context.Context) (time.Duration, error) {
	defer f.metrics.StartContractRequest("GetMaxClockDuration")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodMaxClockDuration))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max clock duration: %w", err)
	}
	return time.Duration(result.GetUint64(0)) * time.Second, nil
}

func (f *FaultDisputeGameContract) GetMaxGameDepth(ctx context.Context) (types.Depth, error) {
	defer f.metrics.StartContractRequest("GetMaxGameDepth")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodMaxGameDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max game depth: %w", err)
	}
	return types.Depth(result.GetBigInt(0).Uint64()), nil
}

func (f *FaultDisputeGameContract) GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	defer f.metrics.StartContractRequest("GetAbsolutePrestateHash")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodAbsolutePrestate))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch absolute prestate hash: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetL1Head(ctx context.Context) (common.Hash, error) {
	defer f.metrics.StartContractRequest("GetL1Head")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	defer f.metrics.StartContractRequest("GetStatus")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (f *FaultDisputeGameContract) GetClaimCount(ctx context.Context) (uint64, error) {
	defer f.metrics.StartContractRequest("GetClaimCount")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodClaimCount))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch claim count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetClaim(ctx context.Context, idx uint64) (types.Claim, error) {
	defer f.metrics.StartContractRequest("GetClaim")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodClaim, new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.Claim{}, fmt.Errorf("failed to fetch claim %v: %w", idx, err)
	}
	return f.decodeClaim(result, int(idx)), nil
}

func (f *FaultDisputeGameContract) GetAllClaims(ctx context.Context, block rpcblock.Block) ([]types.Claim, error) {
	defer f.metrics.StartContractRequest("GetAllClaims")()
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

func (f *FaultDisputeGameContract) IsResolved(ctx context.Context, block rpcblock.Block, claims ...types.Claim) ([]bool, error) {
	defer f.metrics.StartContractRequest("IsResolved")()
	calls := make([]batching.Call, 0, len(claims))
	for _, claim := range claims {
		calls = append(calls, f.contract.Call(methodResolvedSubgames, big.NewInt(int64(claim.ContractIndex))))
	}
	results, err := f.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resolved subgames: %w", err)
	}
	resolved := make([]bool, 0, len(claims))
	for _, result := range results {
		resolved = append(resolved, result.GetBool(0))
	}
	return resolved, nil
}

func (f *FaultDisputeGameContract) vm(ctx context.Context) (*VMContract, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodVM))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VM addr: %w", err)
	}
	vmAddr := result.GetAddress(0)
	return NewVMContract(vmAddr, f.multiCaller), nil
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
	defer f.metrics.StartContractRequest("CallResolveClaim")()
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
	return f.contract.Call(methodResolveClaim, new(big.Int).SetUint64(claimIdx), maxChildChecks)
}

func (f *FaultDisputeGameContract) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	defer f.metrics.StartContractRequest("CallResolve")()
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
func decodeClock(clock *big.Int) types.Clock {
	maxUint64 := new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(1))
	remainder := new(big.Int)
	quotient, _ := new(big.Int).QuoRem(clock, maxUint64, remainder)
	return types.NewClock(time.Duration(quotient.Int64())*time.Second, time.Unix(remainder.Int64(), 0))
}

// packClock packs the Clock duration and timestamp into a uint128.
func packClock(c types.Clock) *big.Int {
	duration := big.NewInt(int64(c.Duration.Seconds()))
	encoded := new(big.Int).Lsh(duration, 64)
	return new(big.Int).Or(encoded, big.NewInt(c.Timestamp.Unix()))
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
