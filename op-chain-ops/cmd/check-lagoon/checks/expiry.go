package checks

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// DefaultExpiryPeriod is the L2ToL2CrossDomainMessenger's expiry period on production networks,
// Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD: the 7-day message expiry window plus a day of margin.
const DefaultExpiryPeriod = depset.MessageExpiryTimeSecondsInterop + 24*60*60

// ExpiryChain is one interop chain checked by CheckExpiry.
type ExpiryChain struct {
	L2 *sources.EthClient
	// L1Messenger is the chain's L1CrossDomainMessenger. The zero address skips the pause
	// comparison for this chain.
	L1Messenger common.Address
}

// CheckExpiryConfig holds the dependencies of CheckExpiry.
type CheckExpiryConfig struct {
	Log log.Logger
	// L1 is required when any chain sets L1Messenger.
	L1     *sources.EthClient
	Chains []ExpiryChain
	// ExpiryPeriod is the expiryPeriod() every chain must report, in seconds.
	ExpiryPeriod uint64
}

// expiryState is what CheckExpiry reads from one chain.
type expiryState struct {
	ChainID      eth.ChainID
	ExpiryPeriod uint64
	L2Paused     bool
	// L1ChainID and L1Paused are read from the chain's L1CrossDomainMessenger, if one is set.
	L1Read    bool
	L1ChainID eth.ChainID
	L1Paused  bool
}

// verify checks a chain's state against the expected expiry period.
func (s expiryState) verify(expiryPeriod uint64) error {
	var errs []error
	if s.ExpiryPeriod != expiryPeriod {
		errs = append(errs, fmt.Errorf("expiryPeriod() is %ds, want %ds", s.ExpiryPeriod, expiryPeriod))
	}
	if s.L1Read {
		if s.L1ChainID != s.ChainID {
			errs = append(errs, fmt.Errorf("L1CrossDomainMessenger belongs to chain %s, not %s", s.L1ChainID, s.ChainID))
		} else if s.L2Paused != s.L1Paused {
			errs = append(errs, fmt.Errorf("ETHLiquidity.paused() is %v but the L1 pause is %v, run L1CrossDomainMessenger.syncETHLiquidityPause()",
				s.L2Paused, s.L1Paused))
		}
	}
	return errors.Join(errs...)
}

// CheckExpiry checks on each chain that the L2ToL2CrossDomainMessenger's expiryPeriod() equals
// cfg.ExpiryPeriod and, where an L1CrossDomainMessenger is given, that ETHLiquidity.paused()
// equals the chain's L1 pause. It checks every chain and reports every mismatch.
func CheckExpiry(ctx context.Context, cfg *CheckExpiryConfig) error {
	if len(cfg.Chains) == 0 {
		return errors.New("expiry check requires at least one L2 chain")
	}
	var errs []error
	for i, chain := range cfg.Chains {
		state, err := readExpiryState(ctx, cfg.L1, chain)
		if err == nil {
			err = state.verify(cfg.ExpiryPeriod)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("chain %d (%s): %w", i, state.ChainID, err))
			continue
		}
		cfg.Log.Info("expiry check passed", "chain", state.ChainID, "expiryPeriod", state.ExpiryPeriod,
			"ethLiquidityPaused", state.L2Paused, "l1PauseCompared", state.L1Read)
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	cfg.Log.Info("message expiry check passed", "chains", len(cfg.Chains))
	return nil
}

func readExpiryState(ctx context.Context, l1 *sources.EthClient, chain ExpiryChain) (expiryState, error) {
	var state expiryState
	chainID, err := chain.L2.ChainID(ctx)
	if err != nil {
		return state, fmt.Errorf("fetch L2 chain id: %w", err)
	}
	state.ChainID = eth.ChainIDFromBig(chainID)

	messenger := bindings.NewBindings[bindings.L2ToL2CrossDomainMessenger](
		bindings.WithClient(chain.L2), bindings.WithTo(predeploys.L2toL2CrossDomainMessengerAddr))
	period, err := contractio.Read(messenger.ExpiryPeriod(), ctx)
	if err != nil {
		return state, fmt.Errorf("read expiryPeriod(): %w", err)
	}
	if !period.IsUint64() {
		return state, fmt.Errorf("expiryPeriod() %s overflows uint64", period)
	}
	state.ExpiryPeriod = period.Uint64()

	liquidity := bindings.NewBindings[bindings.ETHLiquidity](
		bindings.WithClient(chain.L2), bindings.WithTo(predeploys.ETHLiquidityAddr))
	if state.L2Paused, err = contractio.Read(liquidity.Paused(), ctx); err != nil {
		return state, fmt.Errorf("read ETHLiquidity.paused(): %w", err)
	}

	if chain.L1Messenger == (common.Address{}) {
		return state, nil
	}
	if l1 == nil {
		return state, errors.New("an L1CrossDomainMessenger is set but no L1 RPC is")
	}
	l1Messenger := bindings.NewBindings[bindings.L1CrossDomainMessenger](
		bindings.WithClient(l1), bindings.WithTo(chain.L1Messenger))
	sysCfgAddr, err := contractio.Read(l1Messenger.SystemConfig(), ctx)
	if err != nil {
		return state, fmt.Errorf("read L1CrossDomainMessenger.systemConfig(): %w", err)
	}
	sysCfg := bindings.NewBindings[bindings.SystemConfig](bindings.WithClient(l1), bindings.WithTo(sysCfgAddr))
	l1ChainID, err := contractio.Read(sysCfg.L2ChainID(), ctx)
	if err != nil {
		return state, fmt.Errorf("read SystemConfig.l2ChainId(): %w", err)
	}
	state.L1ChainID = eth.ChainIDFromBig(l1ChainID)
	if state.L1Paused, err = contractio.Read(l1Messenger.Paused(), ctx); err != nil {
		return state, fmt.Errorf("read L1CrossDomainMessenger.paused(): %w", err)
	}
	state.L1Read = true
	return state, nil
}
