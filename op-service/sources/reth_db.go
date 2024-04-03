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
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct {
    char* data;
    size_t data_len;
    bool error;
} ReceiptsResult;

typedef struct OpenDBResult {
  const void *data;
  bool error;
} OpenDBResult;

extern ReceiptsResult rdb_read_receipts(const uint8_t* block_hash, size_t block_hash_len, const void *db_instance);
extern void rdb_free_string(char* string);
extern OpenDBResult open_db_read_only(const char *db_path);
*/
import "C"

// FetchRethReceipts fetches the receipts for the given block hash directly from the Reth Database
// and populates the given results slice pointer with the receipts that were found.
func FetchRethReceipts(db unsafe.Pointer, blockHash *common.Hash) (types.Receipts, error) {
	if blockHash == nil {
		return nil, fmt.Errorf("Must provide a block hash to fetch receipts for.")
	}

	// Convert the block hash to a C byte array and defer its deallocation
	cBlockHash := C.CBytes(blockHash[:])
	defer C.free(cBlockHash)

	// Call the C function to fetch the receipts from the Reth Database
	receiptsResult := C.rdb_read_receipts((*C.uint8_t)(cBlockHash), C.size_t(len(blockHash)), db)

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

func OpenDBReadOnly(dbPath string) (db unsafe.Pointer, err error) {
	// Convert the db path to a C string and defer its deallocation
	cDbPath := C.CString(dbPath)
	defer C.free(unsafe.Pointer(cDbPath))

	// Call the C function to fetch the receipts from the Reth Database
	openDBResult := C.open_db_read_only(cDbPath)

	if openDBResult.error {
		return nil, fmt.Errorf("failed to open RethDB")
	}

	return openDBResult.data, nil
}

type RethDBReceiptsFetcher struct {
	dbInstance unsafe.Pointer
}

var _ ReceiptsProvider = (*RethDBReceiptsFetcher)(nil)

// NewRethDBReceiptsFetcher opens a RethDB for reading receipts. It returns nil if it was unable to open the database
func NewRethDBReceiptsFetcher(dbPath string) *RethDBReceiptsFetcher {
	db, err := OpenDBReadOnly(dbPath)
	if err != nil {
		return nil
	}
	return &RethDBReceiptsFetcher{
		dbInstance: db,
	}
}

func (f *RethDBReceiptsFetcher) FetchReceipts(ctx context.Context, block eth.BlockInfo, txHashes []common.Hash) (types.Receipts, error) {
	if f.dbInstance == nil {
		return nil, fmt.Errorf("Reth dbInstance is nil")
	}
	hash := block.Hash()
	return FetchRethReceipts(f.dbInstance, &hash)
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
