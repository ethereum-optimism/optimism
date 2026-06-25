package contracts

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodDisputeGameFactory  = "disputeGameFactory"
	methodOptimismPortal      = "optimismPortal"
	methodAnchorStateRegistry = "anchorStateRegistry"
)

type SystemConfig struct {
	caller         *batching.MultiCaller
	contract       *batching.BoundContract
	networkTimeout time.Duration
}

func NewSystemConfig(addr common.Address, caller *batching.MultiCaller, networkTimeout time.Duration) *SystemConfig {
	return &SystemConfig{
		caller:         caller,
		contract:       batching.NewBoundContract(snapshots.LoadSystemConfigABI(), addr),
		networkTimeout: networkTimeout,
	}
}

func (s *SystemConfig) DisputeGameFactory(ctx context.Context) (common.Address, error) {
	cCtx, cancel := context.WithTimeout(ctx, s.networkTimeout)
	defer cancel()
	result, err := s.caller.SingleCall(cCtx, rpcblock.Latest, s.contract.Call(methodDisputeGameFactory))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get dispute game factory from system config: %w", err)
	}
	return result.GetAddress(0), nil
}

func (s *SystemConfig) AnchorStateRegistry(ctx context.Context) (common.Address, error) {
	cCtx, cancel := context.WithTimeout(ctx, s.networkTimeout)
	defer cancel()
	result, err := s.caller.SingleCall(cCtx, rpcblock.Latest, s.contract.Call(methodOptimismPortal))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get optimism portal from system config: %w", err)
	}
	portal := result.GetAddress(0)
	portalContract := batching.NewBoundContract(snapshots.LoadOptimismPortal2ABI(), portal)

	cCtx, cancel = context.WithTimeout(ctx, s.networkTimeout)
	defer cancel()
	result, err = s.caller.SingleCall(cCtx, rpcblock.Latest, portalContract.Call(methodAnchorStateRegistry))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get anchor state registry from optimism portal %v: %w", portal, err)
	}
	return result.GetAddress(0), nil
}

func (s *SystemConfig) ProposerAddresses(ctx context.Context) (common.Address, common.Address, error) {
	dgf, err := s.DisputeGameFactory(ctx)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	asr, err := s.AnchorStateRegistry(ctx)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	return dgf, asr, nil
}
