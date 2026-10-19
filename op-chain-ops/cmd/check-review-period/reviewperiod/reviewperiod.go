// Package reviewperiod reads a chain's review period delays from L1. It is shared by the
// check-review-period tool and the acceptance tests.
package reviewperiod

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/lmittmann/w3"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

var (
	optimismPortalFn      = w3.MustNewFunc("optimismPortal()", "address")
	delayedWETHFn         = w3.MustNewFunc("delayedWETH()", "address")
	anchorStateRegistryFn = w3.MustNewFunc("anchorStateRegistry()", "address")
	proofMaturityDelayFn  = w3.MustNewFunc("proofMaturityDelaySeconds()", "uint256")
	disputeGameFinalityFn = w3.MustNewFunc("disputeGameFinalityDelaySeconds()", "uint256")
	withdrawalDelayFn     = w3.MustNewFunc("delay()", "uint256")
	l2ChainIDFn           = w3.MustNewFunc("l2ChainId()", "uint256")
	lastUsedOPCMFn        = w3.MustNewFunc("lastUsedOPCM()", "address")
	lastUsedOPCMVersionFn = w3.MustNewFunc("lastUsedOPCMVersion()", "string")
)

// Caller makes read-only calls against L1.
type Caller interface {
	Call(ctx context.Context, msg ethereum.CallMsg, blockNumber rpc.BlockNumber) ([]byte, error)
}

// ChainDelays holds the review period delays of one chain and the contracts they were read from.
type ChainDelays struct {
	SystemConfig                    common.Address
	L2ChainID                       uint64
	OptimismPortal                  common.Address
	AnchorStateRegistry             common.Address
	DelayedWETH                     common.Address
	ProofMaturityDelaySeconds       uint64
	DisputeGameFinalityDelaySeconds uint64
	WithdrawalDelaySeconds          uint64
	// OPCMTracked is false when the SystemConfig predates lastUsedOPCM(), so the two fields below are unset.
	OPCMTracked         bool
	LastUsedOPCM        common.Address
	LastUsedOPCMVersion string
}

// ReadChainDelays resolves the chain's contracts from its SystemConfig and reads the three delays at block.
func ReadChainDelays(ctx context.Context, l1 Caller, systemConfig common.Address, block rpc.BlockNumber) (ChainDelays, error) {
	out := ChainDelays{SystemConfig: systemConfig}
	var err error
	if out.L2ChainID, err = callUint64(ctx, l1, systemConfig, l2ChainIDFn, block); err != nil {
		return out, fmt.Errorf("SystemConfig %s: l2ChainId(): %w", systemConfig, err)
	}
	// lastUsedOPCM() only exists on SystemConfigs from #18547 onwards, so a failed read means it is not tracked.
	if err := call(ctx, l1, systemConfig, lastUsedOPCMFn, block, &out.LastUsedOPCM); err == nil {
		if err := call(ctx, l1, systemConfig, lastUsedOPCMVersionFn, block, &out.LastUsedOPCMVersion); err != nil {
			return out, fmt.Errorf("SystemConfig %s: lastUsedOPCMVersion(): %w", systemConfig, err)
		}
		out.OPCMTracked = true
	}
	if out.OptimismPortal, err = callAddress(ctx, l1, systemConfig, optimismPortalFn, block); err != nil {
		return out, fmt.Errorf("SystemConfig %s: optimismPortal(): %w", systemConfig, err)
	}
	if out.DelayedWETH, err = callAddress(ctx, l1, systemConfig, delayedWETHFn, block); err != nil {
		return out, fmt.Errorf("SystemConfig %s: delayedWETH(): %w", systemConfig, err)
	}
	if out.AnchorStateRegistry, err = callAddress(ctx, l1, out.OptimismPortal, anchorStateRegistryFn, block); err != nil {
		return out, fmt.Errorf("OptimismPortal %s: anchorStateRegistry(): %w", out.OptimismPortal, err)
	}
	if out.ProofMaturityDelaySeconds, err = callUint64(ctx, l1, out.OptimismPortal, proofMaturityDelayFn, block); err != nil {
		return out, fmt.Errorf("OptimismPortal %s: proofMaturityDelaySeconds(): %w", out.OptimismPortal, err)
	}
	if out.DisputeGameFinalityDelaySeconds, err = callUint64(ctx, l1, out.AnchorStateRegistry, disputeGameFinalityFn, block); err != nil {
		return out, fmt.Errorf("AnchorStateRegistry %s: disputeGameFinalityDelaySeconds(): %w", out.AnchorStateRegistry, err)
	}
	if out.WithdrawalDelaySeconds, err = callUint64(ctx, l1, out.DelayedWETH, withdrawalDelayFn, block); err != nil {
		return out, fmt.Errorf("DelayedWETH %s: delay(): %w", out.DelayedWETH, err)
	}
	return out, nil
}

func call(ctx context.Context, l1 Caller, to common.Address, fn *w3.Func, block rpc.BlockNumber, result any) error {
	data, err := fn.EncodeArgs()
	if err != nil {
		return err
	}
	ret, err := l1.Call(ctx, ethereum.CallMsg{To: &to, Data: data}, block)
	if err != nil {
		return err
	}
	return fn.DecodeReturns(ret, result)
}

// callAddress fails on a zero address, so an unset contract surfaces as an error instead of a bogus read.
func callAddress(ctx context.Context, l1 Caller, to common.Address, fn *w3.Func, block rpc.BlockNumber) (common.Address, error) {
	var addr common.Address
	if err := call(ctx, l1, to, fn, block, &addr); err != nil {
		return common.Address{}, err
	}
	if addr == (common.Address{}) {
		return common.Address{}, errors.New("returned the zero address")
	}
	return addr, nil
}

func callUint64(ctx context.Context, l1 Caller, to common.Address, fn *w3.Func, block rpc.BlockNumber) (uint64, error) {
	var v *big.Int
	if err := call(ctx, l1, to, fn, block, &v); err != nil {
		return 0, err
	}
	if !v.IsUint64() {
		return 0, fmt.Errorf("value %s does not fit in uint64", v)
	}
	return bigs.Uint64Strict(v), nil
}
