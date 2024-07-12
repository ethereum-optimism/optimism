package heads

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func AdvanceUnsafeHead(chainId types.ChainID, newHead entrydb.EntryIdx) Operation {
	return OperationFn(func(heads *Heads) error {
		chain := heads.Get(chainId)
		if chain.Unsafe > newHead {
			return fmt.Errorf("attempting to rewind unsafe head from %v to %v in advance operation for chain ID %v", chain.Unsafe, newHead, chainId)
		}
		chain.Unsafe = newHead
		heads.Put(chainId, chain)
		return nil
	})
}

func AdvanceCrossUnsafeHead(chainId types.ChainID, newHead entrydb.EntryIdx) Operation {
	return OperationFn(func(heads *Heads) error {
		chain := heads.Get(chainId)
		if chain.CrossUnsafe > newHead {
			return fmt.Errorf("attempting to rewind cross-unsafe head from %v to %v in advance operation for chain ID %v", chain.CrossUnsafe, newHead, chainId)
		}
		if chain.Unsafe < newHead {
			return fmt.Errorf("attempting to advance safe head to %v past unsafe head %v for chain ID %v", newHead, chain.Unsafe, chainId)
		}
		chain.CrossUnsafe = newHead
		heads.Put(chainId, chain)
		return nil
	})
}

func AdvanceLocalSafeHead(chainId types.ChainID, newHead entrydb.EntryIdx) Operation {
	return OperationFn(func(heads *Heads) error {
		chain := heads.Get(chainId)
		if chain.LocalSafe > newHead {
			return fmt.Errorf("attempting to rewind local safe head from %v to %v in advance operation for chain ID %v", chain.LocalSafe, newHead, chainId)
		}
		if chain.Unsafe < newHead {
			return fmt.Errorf("attempting to advance local safe head to %v past unsafe head %v for chain ID %v", newHead, chain.Unsafe, chainId)
		}
		chain.LocalSafe = newHead
		heads.Put(chainId, chain)
		return nil
	})
}

func AdvanceLocalFinalizedHead(chainId types.ChainID, newHead entrydb.EntryIdx) Operation {
	return OperationFn(func(heads *Heads) error {
		chain := heads.Get(chainId)
		if chain.LocalFinalized > newHead {
			return fmt.Errorf("attempting to rewind local finalized head from %v to %v in advance operation for chain ID %v", chain.LocalFinalized, newHead, chainId)
		}
		if chain.LocalSafe < newHead {
			return fmt.Errorf("attempting to advance local finalized head to %v past local safe head %v for chain ID %v", newHead, chain.LocalSafe, chainId)
		}
		chain.LocalFinalized = newHead
		heads.Put(chainId, chain)
		return nil
	})
}
