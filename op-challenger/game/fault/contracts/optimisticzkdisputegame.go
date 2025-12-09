package contracts

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

type ProposalStatus uint8

const (
	ProposalStatusUnchallenged ProposalStatus = iota
	ProposalStatusChallenged
	ProposalStatusUnchallengedAndValidProofProvided
	ProposalStatusChallengedAndValidProofProvided
	ProposalStatusResolved
)

func (p ProposalStatus) String() string {
	switch p {
	case ProposalStatusUnchallenged:
		return "Unchallenged"
	case ProposalStatusChallenged:
		return "Challenged"
	case ProposalStatusUnchallengedAndValidProofProvided:
		return "UnchallengedAndValidProofProvided"
	case ProposalStatusChallengedAndValidProofProvided:
		return "ChallengedAndValidProofProvided"
	case ProposalStatusResolved:
		return "Resolved"
	default:
		return fmt.Sprintf("ProposalStatus(%d)", uint8(p))
	}
}

var (
	methodChallenge      = "challenge"
	methodChallengerBond = "challengerBond"
	methodClaimData      = "claimData"
)

type claimData struct {
	ParentIndex uint32
	CounteredBy common.Address
	Prover      common.Address
	Claim       common.Hash
	Status      ProposalStatus
	Deadline    uint64
}

type OptimisticZKDisputeGameContract interface {
	DisputeGameContract
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
	GetProposal(ctx context.Context) (common.Hash, uint64, error)
	GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (ChallengerMetadata, error)
	GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error)
	ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error)
}

type OptimisticZKDisputeGameContractLatest struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func (g *OptimisticZKDisputeGameContractLatest) GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("GetCredit")()
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest,
		g.contract.Call(methodCredit, recipient),
		g.contract.Call(methodStatus))
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

func (g *OptimisticZKDisputeGameContractLatest) ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error) {
	defer g.metrics.StartContractRequest("ClaimCredit")()
	call := g.contract.Call(methodClaimCredit, recipient)
	_, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("%w: %w", ErrSimulationFailed, err)
	}
	return call.ToTxCandidate()
}

var _ OptimisticZKDisputeGameContract = (*OptimisticZKDisputeGameContractLatest)(nil)

func NewOptimisticZKDisputeGameContract(
	m metrics.ContractMetricer,
	addr common.Address,
	caller *batching.MultiCaller,
) (*OptimisticZKDisputeGameContractLatest, error) {
	abi := snapshots.LoadZKDisputeGameABI()
	return &OptimisticZKDisputeGameContractLatest{
		metrics:     m,
		multiCaller: caller,
		contract:    batching.NewBoundContract(abi, addr),
	}, nil
}

func (g *OptimisticZKDisputeGameContractLatest) Addr() common.Address {
	return g.contract.Addr()
}

// GetMetadata returns the basic game metadata
func (g *OptimisticZKDisputeGameContractLatest) GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error) {
	defer g.metrics.StartContractRequest("GetMetadata")()
	results, err := g.multiCaller.Call(ctx, block,
		g.contract.Call(methodL1Head),
		g.contract.Call(methodL2SequenceNumber),
		g.contract.Call(methodRootClaim),
		g.contract.Call(methodStatus),
	)
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 4 {
		return GenericGameMetadata{}, fmt.Errorf("expected 4 results but got %v", len(results))
	}
	l1Head := results[0].GetHash(0)
	l2SequenceNumber := results[1].GetBigInt(0).Uint64()
	rootClaim := results[2].GetHash(0)
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to convert game status: %w", err)
	}
	return GenericGameMetadata{
		L1Head:        l1Head,
		L2SequenceNum: l2SequenceNumber,
		ProposedRoot:  rootClaim,
		Status:        status,
	}, nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetL1Head(ctx context.Context) (common.Hash, error) {
	defer g.metrics.StartContractRequest("GetL1Head")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("GetStatus")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (g *OptimisticZKDisputeGameContractLatest) GetGameRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error) {
	defer g.metrics.StartContractRequest("GetGameRange")()
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest,
		g.contract.Call(methodStartingBlockNumber),
		g.contract.Call(methodL2SequenceNumber))
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

type ChallengerMetadata struct {
	ParentIndex      uint32
	ProposalStatus   ProposalStatus
	ProposedRoot     common.Hash
	L2SequenceNumber uint64
	Deadline         time.Time
}

func (g *OptimisticZKDisputeGameContractLatest) GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (ChallengerMetadata, error) {
	defer g.metrics.StartContractRequest("GetChallengerMetadata")()
	results, err := g.multiCaller.Call(ctx, block,
		g.contract.Call(methodClaimData),
		g.contract.Call(methodL2SequenceNumber))
	if err != nil {
		return ChallengerMetadata{}, fmt.Errorf("failed to retrieve challenger metadata: %w", err)
	}
	if len(results) != 2 {
		return ChallengerMetadata{}, fmt.Errorf("expected 2 results but got %v", len(results))
	}
	data := g.decodeClaimData(results[0])
	l2SeqNum := results[1].GetBigInt(0).Uint64()
	return ChallengerMetadata{
		ParentIndex:      data.ParentIndex,
		ProposalStatus:   data.Status,
		ProposedRoot:     data.Claim,
		L2SequenceNumber: l2SeqNum,
		Deadline:         time.Unix(int64(data.Deadline), 0),
	}, nil
}

func (g *OptimisticZKDisputeGameContractLatest) ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error) {
	tx, err := g.contract.Call(methodChallenge).ToTxCandidate()
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to create challenge tx: %w", err)
	}

	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodChallengerBond))
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to retrieve challenger bond: %w", err)
	}
	tx.Value = result.GetBigInt(0)
	return tx, nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetProposal(ctx context.Context) (common.Hash, uint64, error) {
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest, g.contract.Call(methodRootClaim), g.contract.Call(methodL2SequenceNumber))
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to retrieve proposal: %w", err)
	}
	if len(results) != 2 {
		return common.Hash{}, 0, fmt.Errorf("expected 2 results but got %v", len(results))
	}
	return results[0].GetHash(0), results[1].GetBigInt(0).Uint64(), nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error) {
	defer g.metrics.StartContractRequest("GetResolvedAt")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodResolvedAt))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve resolution time: %w", err)
	}
	resolvedAt := time.Unix(int64(result.GetUint64(0)), 0)
	return resolvedAt, nil
}

func (g *OptimisticZKDisputeGameContractLatest) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("CallResolve")()
	call := g.resolveCall()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to call resolve: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (g *OptimisticZKDisputeGameContractLatest) ResolveTx() (txmgr.TxCandidate, error) {
	call := g.resolveCall()
	return call.ToTxCandidate()
}

func (g *OptimisticZKDisputeGameContractLatest) resolveCall() *batching.ContractCall {
	return g.contract.Call(methodResolve)
}

func (g *OptimisticZKDisputeGameContractLatest) decodeClaimData(result *batching.CallResult) claimData {
	parentIndex := result.GetUint32(0)
	counteredBy := result.GetAddress(1)
	prover := result.GetAddress(2)
	claim := result.GetHash(3)
	status := result.GetUint8(4)
	deadline := result.GetUint64(5)
	return claimData{
		ParentIndex: parentIndex,
		CounteredBy: counteredBy,
		Prover:      prover,
		Claim:       claim,
		Status:      ProposalStatus(status),
		Deadline:    deadline,
	}
}

var _ DisputeGameContract = (*OptimisticZKDisputeGameContractLatest)(nil)
