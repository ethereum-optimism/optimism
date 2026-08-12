package contracts

import (
	"context"
	"fmt"
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

type ProposalStatus uint8

const (
	ProposalStatusUnchallenged ProposalStatus = iota
	ProposalStatusChallenged
	ProposalStatusUnchallengedAndValidProofProvided
	ProposalStatusChallengedAndValidProofProvided
	ProposalStatusResolved
)

// ProposalStatusFromUint8 converts a contract status value to a checked ProposalStatus.
func ProposalStatusFromUint8(value uint8) (ProposalStatus, error) {
	status := ProposalStatus(value)
	if status > ProposalStatusResolved {
		return 0, fmt.Errorf("invalid proposal status: %d", value)
	}
	return status, nil
}

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
	methodGameCreator    = "gameCreator"
	methodTotalBonds     = "totalBonds"
)

type claimData struct {
	ParentIndex uint32
	Status      ProposalStatus
	Challenger  common.Address
	Prover      common.Address
	Deadline    uint64
	Claim       common.Hash
}

type ZKDisputeGameContract interface {
	DisputeGameContract
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
	GetProposal(ctx context.Context) (common.Hash, uint64, error)
	GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (ChallengerMetadata, error)
	GetAnchorStateRegistry(ctx context.Context, block rpcblock.Block) (common.Address, error)
	GetBondMetadata(ctx context.Context, block rpcblock.Block) (ZKBondMetadata, error)
	GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error)
	GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*WithdrawalRequest, error)
	GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
	IsClosed(ctx context.Context) (bool, error)
	GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error)
	ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error)
	GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (types.BondDistributionMode, error)
	CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error)
}

type ZKDisputeGameContractLatest struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func (g *ZKDisputeGameContractLatest) IsClosed(ctx context.Context) (bool, error) {
	mode, err := g.GetBondDistributionMode(ctx, rpcblock.Latest)
	if err != nil {
		return false, err
	}
	return mode != types.UndecidedDistributionMode, nil
}

func (g *ZKDisputeGameContractLatest) GetCredit(ctx context.Context, recipient common.Address) (*big.Int, gameTypes.GameStatus, error) {
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

func (g *ZKDisputeGameContractLatest) ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error) {
	defer g.metrics.StartContractRequest("ClaimCredit")()
	call := g.contract.Call(methodClaimCredit, recipient)
	_, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("%w: %w", ErrSimulationFailed, err)
	}
	return call.ToTxCandidate()
}

func (g *ZKDisputeGameContractLatest) GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (types.BondDistributionMode, error) {
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodBondDistributionMode))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch bond mode: %w", err)
	}
	return types.BondDistributionMode(result.GetUint8(0)), nil
}

func (g *ZKDisputeGameContractLatest) CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error) {
	defer g.metrics.StartContractRequest("CloseGame")()
	call := g.contract.Call(methodCloseGame)
	_, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("%w: %w", ErrSimulationFailed, err)
	}
	return call.ToTxCandidate()
}

var _ ZKDisputeGameContract = (*ZKDisputeGameContractLatest)(nil)

func NewZKDisputeGameContract(
	m metrics.ContractMetricer,
	addr common.Address,
	caller *batching.MultiCaller,
) (*ZKDisputeGameContractLatest, error) {
	abi := snapshots.LoadZKDisputeGameABI()
	return &ZKDisputeGameContractLatest{
		metrics:     m,
		multiCaller: caller,
		contract:    batching.NewBoundContract(abi, addr),
	}, nil
}

func (g *ZKDisputeGameContractLatest) Addr() common.Address {
	return g.contract.Addr()
}

// GetMetadata returns the basic game metadata
func (g *ZKDisputeGameContractLatest) GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error) {
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
	l2SequenceNumber := getBlockNumber(results[1], 0)
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

func (g *ZKDisputeGameContractLatest) GetL1Head(ctx context.Context) (common.Hash, error) {
	defer g.metrics.StartContractRequest("GetL1Head")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (g *ZKDisputeGameContractLatest) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("GetStatus")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

// GetGameRange returns super-root timestamps, not L2 block numbers, for the super-root ZK game; they
// feed status display and the (Noop) sync validator.
func (g *ZKDisputeGameContractLatest) GetGameRange(ctx context.Context) (prestateSeqNr uint64, poststateSeqNr uint64, retErr error) {
	defer g.metrics.StartContractRequest("GetGameRange")()
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest,
		g.contract.Call(methodStartingSequenceNumber),
		g.contract.Call(methodL2SequenceNumber))
	if err != nil {
		retErr = fmt.Errorf("failed to retrieve game block range: %w", err)
		return
	}
	if len(results) != 2 {
		retErr = fmt.Errorf("expected 2 results but got %v", len(results))
		return
	}
	prestateSeqNr = getBlockNumber(results[0], 0)
	poststateSeqNr = getBlockNumber(results[1], 0)
	return
}

type ChallengerMetadata struct {
	ParentIndex      uint32
	ProposalStatus   ProposalStatus
	Challenger       common.Address
	Prover           common.Address
	ProposedRoot     common.Hash
	L2SequenceNumber uint64
	Deadline         time.Time
}

// ZKBondMetadata contains the pinned values needed to account for ZK game bonds.
type ZKBondMetadata struct {
	GameCreator    common.Address
	TotalBonds     *big.Int
	ChallengerBond *big.Int
}

func (g *ZKDisputeGameContractLatest) GetBondMetadata(ctx context.Context, block rpcblock.Block) (ZKBondMetadata, error) {
	defer g.metrics.StartContractRequest("GetBondMetadata")()
	results, err := g.multiCaller.Call(ctx, block,
		g.contract.Call(methodGameCreator),
		g.contract.Call(methodTotalBonds),
		g.contract.Call(methodChallengerBond),
	)
	if err != nil {
		return ZKBondMetadata{}, fmt.Errorf("failed to retrieve ZK bond metadata: %w", err)
	}
	if err := validateResultCount(3, len(results)); err != nil {
		return ZKBondMetadata{}, err
	}
	return ZKBondMetadata{
		GameCreator:    results[0].GetAddress(0),
		TotalBonds:     results[1].GetBigInt(0),
		ChallengerBond: results[2].GetBigInt(0),
	}, nil
}

func (g *ZKDisputeGameContractLatest) GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	defer g.metrics.StartContractRequest("GetCredits")()
	if len(recipients) == 0 {
		return []*big.Int{}, nil
	}
	calls := make([]batching.Call, 0, len(recipients))
	for _, recipient := range recipients {
		calls = append(calls, g.contract.Call(methodCredit, recipient))
	}
	results, err := g.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ZK credit: %w", err)
	}
	if err := validateResultCount(len(recipients), len(results)); err != nil {
		return nil, err
	}
	credits := make([]*big.Int, len(results))
	for i, result := range results {
		credits[i] = result.GetBigInt(0)
	}
	return credits, nil
}

func (g *ZKDisputeGameContractLatest) GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*WithdrawalRequest, error) {
	defer g.metrics.StartContractRequest("GetWithdrawals")()
	if len(recipients) == 0 {
		return []*WithdrawalRequest{}, nil
	}
	delayedWETH, err := g.getDelayedWETH(ctx, block)
	if err != nil {
		return nil, err
	}
	return delayedWETH.GetWithdrawals(ctx, block, g.contract.Addr(), recipients...)
}

func (g *ZKDisputeGameContractLatest) GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	defer g.metrics.StartContractRequest("GetBalanceAndDelay")()
	delayedWETH, err := g.getDelayedWETH(ctx, block)
	if err != nil {
		return nil, 0, common.Address{}, err
	}
	balance, delay, err := delayedWETH.GetBalanceAndDelay(ctx, block)
	if err != nil {
		return nil, 0, common.Address{}, err
	}
	return balance, delay, delayedWETH.Addr(), nil
}

func (g *ZKDisputeGameContractLatest) getDelayedWETH(ctx context.Context, block rpcblock.Block) (*DelayedWETHContract, error) {
	defer g.metrics.StartContractRequest("GetDelayedWETH")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodWETH))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZK WETH address: %w", err)
	}
	return NewDelayedWETHContract(g.metrics, result.GetAddress(0), g.multiCaller), nil
}

func (g *ZKDisputeGameContractLatest) GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (ChallengerMetadata, error) {
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
	l2SeqNum := getBlockNumber(results[1], 0)
	return ChallengerMetadata{
		ParentIndex:      data.ParentIndex,
		ProposalStatus:   data.Status,
		Challenger:       data.Challenger,
		Prover:           data.Prover,
		ProposedRoot:     data.Claim,
		L2SequenceNumber: l2SeqNum,
		Deadline:         time.Unix(int64(data.Deadline), 0),
	}, nil
}

func (g *ZKDisputeGameContractLatest) GetAnchorStateRegistry(ctx context.Context, block rpcblock.Block) (common.Address, error) {
	defer g.metrics.StartContractRequest("GetAnchorStateRegistry")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodAnchorStateRegistry))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to retrieve anchor state registry: %w", err)
	}
	return result.GetAddress(0), nil
}

func (g *ZKDisputeGameContractLatest) ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error) {
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

// GetProposal returns the root claim and its l2SequenceNumber. For the super-root ZK game the root
// claim is a super-root hash and l2SequenceNumber is a super-root timestamp, not an L2 block number.
func (g *ZKDisputeGameContractLatest) GetProposal(ctx context.Context) (common.Hash, uint64, error) {
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest, g.contract.Call(methodRootClaim), g.contract.Call(methodL2SequenceNumber))
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to retrieve proposal: %w", err)
	}
	if len(results) != 2 {
		return common.Hash{}, 0, fmt.Errorf("expected 2 results but got %v", len(results))
	}
	return results[0].GetHash(0), getBlockNumber(results[1], 0), nil
}

func (g *ZKDisputeGameContractLatest) GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error) {
	defer g.metrics.StartContractRequest("GetResolvedAt")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodResolvedAt))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve resolution time: %w", err)
	}
	resolvedAt := time.Unix(int64(result.GetUint64(0)), 0)
	return resolvedAt, nil
}

func (g *ZKDisputeGameContractLatest) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("CallResolve")()
	call := g.resolveCall()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to call resolve: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (g *ZKDisputeGameContractLatest) ResolveTx() (txmgr.TxCandidate, error) {
	call := g.resolveCall()
	return call.ToTxCandidate()
}

func (g *ZKDisputeGameContractLatest) resolveCall() *batching.ContractCall {
	return g.contract.Call(methodResolve)
}

func (g *ZKDisputeGameContractLatest) decodeClaimData(result *batching.CallResult) claimData {
	parentIndex := result.GetUint32(0)
	status := result.GetUint8(1)
	challenger := result.GetAddress(2)
	prover := result.GetAddress(3)
	deadline := result.GetUint64(4)
	claim := result.GetHash(5)
	return claimData{
		ParentIndex: parentIndex,
		Status:      ProposalStatus(status),
		Challenger:  challenger,
		Prover:      prover,
		Deadline:    deadline,
		Claim:       claim,
	}
}

var _ DisputeGameContract = (*ZKDisputeGameContractLatest)(nil)
