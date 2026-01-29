package backend

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet/frontend"
	"github.com/ethereum-optimism/optimism/op-faucet/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type APIRouter interface {
	AddRPC(route string) error
	AddAPIToRPC(route string, api rpc.API) error
}

type Backend struct {
	log      log.Logger
	m        metrics.Metricer
	faucets  locks.RWMap[ftypes.FaucetID, *Faucet]
	defaults locks.RWMap[eth.ChainID, ftypes.FaucetID]
}

func FromConfig(log log.Logger, m metrics.Metricer, cfg *config.Config, router APIRouter) (*Backend, error) {
	b := &Backend{
		log: log,
		m:   m,
	}

	var faucetIDs []ftypes.FaucetID
	for fID, fCfg := range cfg.Faucets {
		f, err := FaucetFromConfig(log, m, fID, fCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to setup faucet %q: %w", fID, err)
		}
		b.faucets.Set(fID, f)
		faucetIDs = append(faucetIDs, fID)
	}

	for chID, fID := range cfg.Defaults {
		if !b.faucets.Has(fID) {
			return nil, fmt.Errorf("unknown faucet, cannot set as default for chain %s: %q", chID, fID)
		}
		b.defaults.Set(chID, fID)
	}

	// Infer defaults for chains that were not explicitly mentioned.
	// Always use the lowest faucet ID, so map-iteration doesn't affect defaults.
	sort.Slice(faucetIDs, func(i, j int) bool {
		return faucetIDs[i] < faucetIDs[j]
	})
	for _, fID := range faucetIDs {
		f, _ := b.faucets.Get(fID)
		b.defaults.SetIfMissing(f.chainID, fID)
	}

	// Set up the faucet routes
	var faucetErr error
	b.faucets.Range(func(id ftypes.FaucetID, f *Faucet) bool {
		if err := router.AddRPC("/faucet/" + id.String()); err != nil {
			faucetErr = errors.Join(fmt.Errorf("failed to setup faucet route for %q: %w", id, err))
			return true
		}
		if err := router.AddAPIToRPC("/faucet/"+id.String(), rpc.API{
			Namespace: "faucet",
			Service:   frontend.NewFaucetFrontend(f),
		}); err != nil {
			faucetErr = errors.Join(faucetErr,
				fmt.Errorf("failed to setup faucet RPC for %q: %w", id, err))
		}
		return true
	})
	if faucetErr != nil {
		return nil, fmt.Errorf("failed to set up faucet route(s): %w", faucetErr)
	}

	// Set up the faucet aliases per chain
	var defaultSetupErr error
	b.defaults.Range(func(chain eth.ChainID, id ftypes.FaucetID) bool {
		f, _ := b.faucets.Get(id)
		if err := router.AddRPC("/chain/" + chain.String()); err != nil {
			defaultSetupErr = errors.Join(fmt.Errorf("failed to setup chain route for %q: %w", id, err))
			return true
		}
		if err := router.AddAPIToRPC("/chain/"+chain.String(), rpc.API{
			Namespace: "faucet",
			Service:   frontend.NewFaucetFrontend(f),
		}); err != nil {
			defaultSetupErr = errors.Join(defaultSetupErr,
				fmt.Errorf("failed to setup alias for %q: %w", chain, err))
		}
		return true
	})
	if defaultSetupErr != nil {
		return nil, fmt.Errorf("failed to set up chain alias(es): %w", defaultSetupErr)
	}

	return b, nil
}

// FaucetByChain gets the default faucet for the given chain.
// This returns nil if there is no such faucet configured (unknown chain or missing faucet).
func (b *Backend) FaucetByChain(id eth.ChainID) *Faucet {
	fID, ok := b.defaults.Get(id)
	if !ok {
		return nil
	}
	out, _ := b.faucets.Get(fID)
	return out
}

// FaucetByID gets the faucet by its identifier.
// This returns nil if there is no such faucet configured.
func (b *Backend) FaucetByID(id ftypes.FaucetID) *Faucet {
	out, _ := b.faucets.Get(id)
	return out
}

func (b *Backend) EnableFaucet(id ftypes.FaucetID) {
	f, ok := b.faucets.Get(id)
	if ok {
		f.Enable()
	}
}

func (b *Backend) DisableFaucet(id ftypes.FaucetID) {
	f, ok := b.faucets.Get(id)
	if ok {
		f.Disable()
	}
}

func (b *Backend) Stop(ctx context.Context) error {
	// We have support for ctx/error here,
	// for future improvements like awaiting txs to complete and/or storing rate-limit data to disk.

	b.faucets.Range(func(key ftypes.FaucetID, value *Faucet) bool {
		value.Close()
		return true
	})
	return nil
}

func (b *Backend) Faucets() (out map[ftypes.FaucetID]eth.ChainID) {
	out = make(map[ftypes.FaucetID]eth.ChainID)
	b.faucets.Range(func(key ftypes.FaucetID, value *Faucet) bool {
		out[key] = value.chainID
		return true
	})
	return out
}

func (b *Backend) Defaults() (out map[eth.ChainID]ftypes.FaucetID) {
	out = make(map[eth.ChainID]ftypes.FaucetID)
	b.defaults.Range(func(key eth.ChainID, value ftypes.FaucetID) bool {
		out[key] = value
		return true
	})
	return out
}
