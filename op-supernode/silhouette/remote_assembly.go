package silhouette

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// RemoteAssembly is the entire supernode-side integration for a standalone op-silhouette-el. It
// owns no proof verifier, L1 reader, fact table, or execution shim: those all live behind rpc.
type RemoteAssembly struct {
	ChainID eth.ChainID
	Source  RemoteInteropSource

	log  log.Logger
	rpc  client.RPC
	sink *LogSink
}

// RemoteInteropSource is the standalone EL surface the supernode mirrors. Keeping this as the RPC
// contract, rather than a FactStore, makes the process boundary explicit and lets the reconciliation
// algorithm be tested independently of a socket.
type RemoteInteropSource interface {
	InteropSource
	Status(ctx context.Context) (*Status, error)
}

func NewRemoteAssembly(logger log.Logger, chainID eth.ChainID, rpc client.RPC) *RemoteAssembly {
	return &RemoteAssembly{
		ChainID: chainID,
		Source:  NewRPCInteropSource(rpc),
		log:     logger,
		rpc:     rpc,
	}
}

func (a *RemoteAssembly) AttachLogStore(store LogStore) error {
	if store == nil {
		return fmt.Errorf("chain %s: no log store", a.ChainID)
	}
	if a.sink != nil {
		return fmt.Errorf("chain %s: log store already attached", a.ChainID)
	}
	a.sink = NewLogSink(a.log.New("component", "remote-log-ingester"), store)
	return nil
}

func (a *RemoteAssembly) LogSink() *LogSink { return a.sink }

// Run mirrors the standalone EL's proof-derived public message view into the supernode LogsDB.
// Cross-safety already retries a block whose log seal has not arrived, so this RPC hand-off is a
// normal lagging-source condition rather than shared-memory coordination.
func (a *RemoteAssembly) Run(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := a.sync(ctx); err != nil && ctx.Err() == nil {
			a.log.Warn("could not sync silhouette interop facts from standalone EL", "chain", a.ChainID, "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *RemoteAssembly) sync(ctx context.Context) error {
	if a.sink == nil {
		return nil
	}
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	status, err := a.Source.Status(callCtx)
	if err != nil {
		return err
	}
	if status.HeadFact == nil || status.OldestFact == nil {
		// The standalone EL may have rewound ahead of the stock interop reorg pipeline. The stock
		// pipeline remains the sole owner of LogsDB rewinds, including the order in which accepted,
		// safe, and finalized heads move. Wait until it has removed the stale seals itself.
		return nil
	}
	head := uint64(*status.HeadFact)
	oldest := uint64(*status.OldestFact)

	// Mirroring is append-only. Rewinding here would race stock interop's timestamp-by-timestamp
	// reorg plan and replace seals that it still needs to identify the old accepted branch. If the
	// standalone EL is shorter or divergent, wait for stock interop to rewind LogsDB to the common
	// ancestor. On a later tick the seal at latest will match and replacement facts can be appended.
	if latest, ok := a.sink.store.LatestSealedBlock(); ok {
		if latest.Number > head || latest.Number < oldest {
			return nil
		}
		remote, err := a.Source.InteropBlock(callCtx, latest.Number)
		if err != nil {
			return err
		}
		if remote == nil || remote.Hash != latest.Hash {
			return nil
		}
	}

	next := oldest
	if latest, ok := a.sink.store.LatestSealedBlock(); ok {
		next = latest.Number + 1
	}
	for next <= head {
		block, err := a.Source.InteropBlock(callCtx, next)
		if err != nil {
			return err
		}
		if block == nil {
			return fmt.Errorf("remote EL reports head %d but has no interop facts for block %d", head, next)
		}
		export := proofbatch.BlockExport{
			Number:    uint64(block.Number),
			Timestamp: uint64(block.Timestamp),
			Hash:      block.Hash,
			Logs:      block.Logs,
		}
		if err := a.sink.Accept([]proofbatch.BlockExport{export}, block.ParentHash); err != nil {
			return err
		}
		next++
	}
	return nil
}

func (a *RemoteAssembly) Close() {
	if a.rpc != nil {
		a.rpc.Close()
	}
}
