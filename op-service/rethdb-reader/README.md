# `rethdb-reader`

A dylib to be accessed via FFI in `op-service`'s `sources` package for reading information
directly from the `reth` database.

### C Header

```c
#include <cstdarg>
#include <cstdint>
#include <cstdlib>
#include <ostream>
#include <new>

struct ReceiptsResult {
  uint32_t *data;
  uintptr_t data_len;
  bool error;
};

extern "C" {

/// Read the receipts for a blockhash from the RETH database directly.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a
///   failing [ReceiptsResult] if any are found.
ReceiptsResult read_receipts(const uint8_t *block_hash,
                             uintptr_t block_hash_len,
                             const char *db_path);

/// Free a string that was allocated in Rust and passed to C.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
void free_string(char *string);

}
```
