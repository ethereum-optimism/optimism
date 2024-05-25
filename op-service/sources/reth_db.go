//go:build rethdb

package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

/*
#cgo LDFLAGS: -L../rethdb-reader/target/release -lrethdbreader
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    char* data;
    size_t data_len;
    bool error;
} ReceiptsResult;

extern ReceiptsResult rdb_read_receipts(const uint8_t* block_hash, size_t block_hash_len, const char* db_path);
extern void rdb_free_string(char* string);
*/
import "C"

// FetchRethReceipts fetches the receipts for the given block hash directly from the Reth Database
// and populates the given results slice pointer with the receipts that were found.
func FetchRethReceipts(dbPath string, blockHash *common.Hash) (types.Receipts, error) {
	if blockHash == nil {
		return nil, fmt.Errorf("Must provide a block hash to fetch receipts for.")
	}

	// Convert the block hash to a C byte array and defer its deallocation
	cBlockHash := C.CBytes(blockHash[:])
	defer C.free(cBlockHash)

	// Convert the db path to a C string and defer its deallocation
	cDbPath := C.CString(dbPath)
	defer C.free(unsafe.Pointer(cDbPath))

	// Call the C function to fetch the receipts from the Reth Database
	receiptsResult := C.rdb_read_receipts((*C.uint8_t)(cBlockHash), C.size_t(len(blockHash)), cDbPath)

	if receiptsResult.error {
		return nil, fmt.Errorf("Error fetching receipts from Reth Database.")
	}

	// Free the memory allocated by the C code
	defer C.rdb_free_string(receiptsResult.data)

	// Convert the returned JSON string to Go string and parse it
	receiptsJSON := C.GoStringN(receiptsResult.data, C.int(receiptsResult.data_len))
	var receipts types.Receipts
	if err := json.Unmarshal([]byte(receiptsJSON), &receipts); err != nil {
		return nil, err
	}

	return receipts, nil
}

type RethDBReceiptsFetcher struct {
	dbPath string
	// TODO(8225): Now that we have reading from a Reth DB encapsulated here,
	//   We could store a reference to the RethDB here instead of just a db path,
	//   which would be more optimal.
	//   We could move the opening of the RethDB and creation of the db reference
	//   into NewRethDBReceiptsFetcher.
}

func NewRethDBReceiptsFetcher(dbPath string) *RethDBReceiptsFetcher {
	return &RethDBReceiptsFetcher{
		dbPath: dbPath,
	}
}

func (f *RethDBReceiptsFetcher) FetchReceipts(ctx context.Context, block eth.BlockID, txHashes []common.Hash) (types.Receipts, error) {
	return FetchRethReceipts(f.dbPath, &block.Hash)
}

func NewCachingRethDBReceiptsFetcher(dbPath string, m caching.Metrics, cacheSize int) *CachingReceiptsProvider {
	return NewCachingReceiptsProvider(NewRethDBReceiptsFetcher(dbPath), m, cacheSize)
}

const buildRethdb = true

func newRecProviderFromConfig(client client.RPC, log log.Logger, metrics caching.Metrics, config *EthClientConfig) *CachingReceiptsProvider {
	if dbPath := config.RethDBPath; dbPath != "" {
		return NewCachingRethDBReceiptsFetcher(dbPath, metrics, config.ReceiptsCacheSize)
	}
	return newRPCRecProviderFromConfig(client, log, metrics, config)
}
