package contracts

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	methodProvenWithdrawals         = "provenWithdrawals"
	methodDeleteProvenWithdrawal    = "deleteProvenWithdrawal"
	eventWithdrawalProvenExtension1 = "WithdrawalProvenExtension1"
)

// WithdrawalProof identifies a single proof of a withdrawal.
// The portal records one entry per (withdrawal, submitter) pair.
type WithdrawalProof struct {
	WithdrawalHash common.Hash
	ProofSubmitter common.Address
}

// ProvenWithdrawal is the portal's record for a WithdrawalProof.
// Timestamp is zero when no record exists, including after the proof has been deleted.
type ProvenWithdrawal struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}

type OptimismPortal2Contract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	abi         *abi.ABI
	contract    *batching.BoundContract
}

func NewOptimismPortal2Contract(metrics metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *OptimismPortal2Contract {
	portalAbi := snapshots.LoadOptimismPortal2ABI()
	return &OptimismPortal2Contract{
		metrics:     metrics,
		multiCaller: caller,
		abi:         portalAbi,
		contract:    batching.NewBoundContract(portalAbi, addr),
	}
}

func (p *OptimismPortal2Contract) Addr() common.Address {
	return p.contract.Addr()
}

// WithdrawalProvenExtension1Topic is the event topic to filter L1 logs on to find withdrawal proofs.
func (p *OptimismPortal2Contract) WithdrawalProvenExtension1Topic() common.Hash {
	return p.abi.Events[eventWithdrawalProvenExtension1].ID
}

func (p *OptimismPortal2Contract) DecodeWithdrawalProvenExtension1(log *ethTypes.Log) (WithdrawalProof, error) {
	name, result, err := p.contract.DecodeEvent(log)
	if err != nil {
		return WithdrawalProof{}, err
	}
	if name != eventWithdrawalProvenExtension1 {
		return WithdrawalProof{}, fmt.Errorf("%w: %v", batching.ErrUnknownEvent, name)
	}
	return WithdrawalProof{WithdrawalHash: result.GetHash(0), ProofSubmitter: result.GetAddress(1)}, nil
}

// GetProvenWithdrawals reads the portal's record for each proof, in the order supplied.
func (p *OptimismPortal2Contract) GetProvenWithdrawals(ctx context.Context, block rpcblock.Block, proofs []WithdrawalProof) ([]ProvenWithdrawal, error) {
	defer p.metrics.StartContractRequest("GetProvenWithdrawals")()
	calls := make([]batching.Call, len(proofs))
	for i, proof := range proofs {
		calls[i] = p.contract.Call(methodProvenWithdrawals, proof.WithdrawalHash, proof.ProofSubmitter)
	}
	results, err := p.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve proven withdrawals: %w", err)
	}
	withdrawals := make([]ProvenWithdrawal, len(results))
	for i, result := range results {
		withdrawals[i] = ProvenWithdrawal{
			DisputeGameProxy: result.GetAddress(0),
			Timestamp:        result.GetUint64(1),
		}
	}
	return withdrawals, nil
}

func (p *OptimismPortal2Contract) DeleteProvenWithdrawalTx(proof WithdrawalProof) (txmgr.TxCandidate, error) {
	return p.contract.Call(methodDeleteProvenWithdrawal, proof.WithdrawalHash, proof.ProofSubmitter).ToTxCandidate()
}
