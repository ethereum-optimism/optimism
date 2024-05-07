package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"
)

// RLPBlockRange is a range of blocks in RLP format
type RLPBlockRange struct {
	start    uint64
	hashes   [][]byte
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	tds      [][]byte
}

func migrateAncientsDb(oldDBPath, newDBPath string, batchSize uint64) (uint64, error) {
	oldFreezer, err := rawdb.NewChainFreezer(filepath.Join(oldDBPath, "ancient"), "", true)
	if err != nil {
		return 0, fmt.Errorf("failed to open old freezer: %v", err)
	}
	defer oldFreezer.Close()

	newFreezer, err := rawdb.NewChainFreezer(filepath.Join(newDBPath, "ancient"), "", false)
	if err != nil {
		return 0, fmt.Errorf("failed to open new freezer: %v", err)
	}
	defer newFreezer.Close()

	numAncientsOld, err := oldFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in old freezer: %v", err)
	}

	numAncientsNew, err := newFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in new freezer: %v", err)
	}

	log.Info("Migration Started", "process", "ancients migration", "startBlock", numAncientsNew, "endBlock", numAncientsOld, "count", numAncientsOld-numAncientsNew+1)
	g, ctx := errgroup.WithContext(context.Background())
	readChan := make(chan RLPBlockRange, 10)
	transformChan := make(chan RLPBlockRange, 10)

	log.Info("Migrating data", "start", numAncientsNew, "end", numAncientsOld, "step", batchSize)

	g.Go(func() error {
		return readAncientBlocks(ctx, oldFreezer, numAncientsNew, numAncientsOld, batchSize, readChan)
	})
	g.Go(func() error { return transformBlocks(ctx, readChan, transformChan) })
	g.Go(func() error { return writeAncientBlocks(ctx, newFreezer, transformChan) })

	if err = g.Wait(); err != nil {
		return 0, fmt.Errorf("failed to migrate ancients: %v", err)
	}

	numAncientsNew, err = newFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in new freezer: %v", err)
	}

	log.Info("Migration End", "process", "ancients migration", "totalBlocks", numAncientsNew)
	return numAncientsNew, nil
}

func readAncientBlocks(ctx context.Context, freezer *rawdb.Freezer, startBlock, endBlock, batchSize uint64, out chan<- RLPBlockRange) error {
	defer close(out)

	for i := startBlock; i < endBlock; i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			count := min(batchSize, endBlock-i+1)
			start := i

			blockRange := RLPBlockRange{
				start:    start,
				hashes:   make([][]byte, count),
				headers:  make([][]byte, count),
				bodies:   make([][]byte, count),
				receipts: make([][]byte, count),
				tds:      make([][]byte, count),
			}
			var err error

			blockRange.hashes, err = freezer.AncientRange(rawdb.ChainFreezerHashTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read hashes from old freezer: %v", err)
			}
			blockRange.headers, err = freezer.AncientRange(rawdb.ChainFreezerHeaderTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read headers from old freezer: %v", err)
			}
			blockRange.bodies, err = freezer.AncientRange(rawdb.ChainFreezerBodiesTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read bodies from old freezer: %v", err)
			}
			blockRange.receipts, err = freezer.AncientRange(rawdb.ChainFreezerReceiptTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read receipts from old freezer: %v", err)
			}
			blockRange.tds, err = freezer.AncientRange(rawdb.ChainFreezerDifficultyTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read tds from old freezer: %v", err)
			}

			out <- blockRange
		}
	}
	return nil
}

func transformBlocks(ctx context.Context, in <-chan RLPBlockRange, out chan<- RLPBlockRange) error {
	// Transform blocks from the in channel and send them to the out channel
	defer close(out)
	for blockRange := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			for i := range blockRange.hashes {
				blockNumber := blockRange.start + uint64(i)

				newHeader, err := transformHeader(blockRange.headers[i])
				if err != nil {
					return fmt.Errorf("can't transform header: %v", err)
				}
				newBody, err := transformBlockBody(blockRange.bodies[i])
				if err != nil {
					return fmt.Errorf("can't transform body: %v", err)
				}

				if yes, newHash := hasSameHash(newHeader, blockRange.hashes[i]); !yes {
					log.Error("Hash mismatch", "block", blockNumber, "oldHash", common.BytesToHash(blockRange.hashes[i]), "newHash", newHash)
					return fmt.Errorf("hash mismatch at block %d", blockNumber)
				}

				blockRange.headers[i] = newHeader
				blockRange.bodies[i] = newBody
			}
			out <- blockRange
		}
	}
	return nil
}

func writeAncientBlocks(ctx context.Context, freezer *rawdb.Freezer, in <-chan RLPBlockRange) error {
	// Write blocks from the in channel to the newDb
	for blockRange := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := freezer.ModifyAncients(func(aWriter ethdb.AncientWriteOp) error {
				for i := range blockRange.hashes {
					blockNumber := blockRange.start + uint64(i)
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHashTable, blockNumber, blockRange.hashes[i]); err != nil {
						return fmt.Errorf("can't write hash to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHeaderTable, blockNumber, blockRange.headers[i]); err != nil {
						return fmt.Errorf("can't write header to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerBodiesTable, blockNumber, blockRange.bodies[i]); err != nil {
						return fmt.Errorf("can't write body to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerReceiptTable, blockNumber, blockRange.receipts[i]); err != nil {
						return fmt.Errorf("can't write receipts to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerDifficultyTable, blockNumber, blockRange.tds[i]); err != nil {
						return fmt.Errorf("can't write td to Freezer: %v", err)
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to write block range: %v", err)
			}
			log.Info("Wrote ancient blocks", "start", blockRange.start, "end", blockRange.start+uint64(len(blockRange.hashes)-1), "count", len(blockRange.hashes))
		}
	}
	return nil
}
