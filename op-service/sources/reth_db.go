package sources

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

/*
#cgo LDFLAGS: -L../rethdb-reader/target/release -lrethdbreader
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    uint8_t* data;
    size_t len;
} ByteArray;

typedef struct {
    ByteArray* data;
    size_t len;
} ByteArrays;

// Define ReceiptsResult with a bool for error
typedef struct {
    ByteArrays receipts;
    bool error;
} ReceiptsResult;

extern ReceiptsResult read_receipts(const uint8_t* block_hash, size_t block_hash_len, const char* db_path);
extern void free_byte_arrays(ByteArrays arrays);
*/
import "C"
import "unsafe"

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
	byteArrayStruct := C.read_receipts((*C.uint8_t)(cBlockHash), C.size_t(len(blockHash)), cDbPath)

	if byteArrayStruct.error {
		return nil, fmt.Errorf("Error fetching receipts from Reth Database.")
	}

	// Convert the returned receipt RLP byte arrays to decoded Receipts.
	data := make(types.Receipts, byteArrayStruct.receipts.len)
	byteArraySlice := (*[1 << 30]C.ByteArray)(unsafe.Pointer(byteArrayStruct.receipts.data))[:byteArrayStruct.receipts.len:byteArrayStruct.receipts.len]
	for i, byteArray := range byteArraySlice {
		receipt := types.Receipt{}
		receipt.UnmarshalBinary(C.GoBytes(unsafe.Pointer(byteArray.data), C.int(byteArray.len)))
		data[i] = &receipt
	}

	// Free the memory allocated by the C code
	C.free_byte_arrays(byteArrayStruct.receipts)

	return data, nil
}
