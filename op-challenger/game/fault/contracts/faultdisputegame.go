package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var (
	methodGameDuration       = "gameDuration"
	methodMaxGameDepth       = "maxGameDepth"
	methodAbsolutePrestate   = "absolutePrestate"
	methodStatus             = "status"
	methodClaimCount         = "claimDataLen"
	methodClaim              = "claimData"
	methodL1Head             = "l1Head"
	methodResolve            = "resolve"
	methodResolveClaim       = "resolveClaim"
	methodAttack             = "attack"
	methodDefend             = "defend"
	methodStep               = "step"
	methodAddLocalData       = "addLocalData"
	methodVM                 = "vm"
	methodGenesisBlockNumber = "genesisBlockNumber"
	methodGenesisOutputRoot  = "genesisOutputRoot"
	methodSplitDepth         = "splitDepth"
	methodL2BlockNumber      = "l2BlockNumber"
	methodRequiredBond       = "getRequiredBond"
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

// GetBlockRange returns the block numbers of the absolute pre-state block (typically genesis or the bedrock activation block)
// and the post-state block (that the proposed output root is for).
func (c *FaultDisputeGameContract) GetBlockRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error) {
	results, err := c.multiCaller.Call(ctx, batching.BlockLatest,
		c.contract.Call(methodGenesisBlockNumber),
		c.contract.Call(methodL2BlockNumber))
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

func (c *FaultDisputeGameContract) GetGenesisOutputRoot(ctx context.Context) (common.Hash, error) {
	genesisOutputRoot, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodGenesisOutputRoot))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve genesis output root: %w", err)
	}
	return genesisOutputRoot.GetHash(0), nil
}

func (c *FaultDisputeGameContract) GetSplitDepth(ctx context.Context) (types.Depth, error) {
	splitDepth, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodSplitDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	return types.Depth(splitDepth.GetBigInt(0).Uint64()), nil
}

func (c *FaultDisputeGameContract) GetRequiredBond(ctx context.Context, position types.Position) (*big.Int, error) {
	bond, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodRequiredBond, position.ToGIndex()))
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
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodGameDuration))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch game duration: %w", err)
	}
	return result.GetUint64(0), nil
}

func (f *FaultDisputeGameContract) GetMaxGameDepth(ctx context.Context) (types.Depth, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodMaxGameDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max game depth: %w", err)
	}
	return types.Depth(result.GetBigInt(0).Uint64()), nil
}

func (f *FaultDisputeGameContract) GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodAbsolutePrestate))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch absolute prestate hash: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetL1Head(ctx context.Context) (common.Hash, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (f *FaultDisputeGameContract) GetClaimCount(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodClaimCount))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch claim count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetClaim(ctx context.Context, idx uint64) (types.Claim, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodClaim, new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.Claim{}, fmt.Errorf("failed to fetch claim %v: %w", idx, err)
	}
	return f.decodeClaim(result, int(idx)), nil
}

func (f *FaultDisputeGameContract) GetAllClaims(ctx context.Context) ([]types.Claim, error) {
	results, err := batching.ReadArray(ctx, f.multiCaller, batching.BlockLatest, f.contract.Call(methodClaimCount), func(i *big.Int) *batching.ContractCall {
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
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodVM))
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
	_, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, call)
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
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, call)
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
		Clock:               clock.Uint64(),
		ContractIndex:       contractIndex,
		ParentContractIndex: int(parentIndex),
	}
}
