# Build Cache Validation Package

This package provides build cache validation functionality for the OP Stack acceptance tests. It helps avoid unnecessary Solidity contract recompilation by checking if contract artifacts are up-to-date relative to source files.

## Features

- **Cross-platform compatibility**: Uses Go standard library (`os`, `filepath`, `time`) for file system operations
- **Timestamp comparison**: Compares source files and artifacts to determine cache validity
- **Comprehensive validation**: Checks source files, `foundry.toml`, and artifact existence
- **Detailed status reporting**: Provides detailed information about cache state
- **Logging support**: Configurable logging for debugging and monitoring

## Usage

### Basic Usage

```go
import (
    "context"
    "log"
    "os"

    "github.com/ethereum-optimism/optimism/op-acceptance-tests/buildcache"
)

// Create validator
logger := log.New(os.Stdout, "[BUILD-CACHE] ", log.LstdFlags)
contractsDir := "/path/to/packages/contracts-bedrock"
validator := buildcache.NewBuildCacheValidator(contractsDir, logger)

// Check cache validity
ctx := context.Background()
isValid, err := validator.IsCacheValid(ctx)
if err != nil {
    // Handle error
}

if isValid {
    // Skip rebuild
} else {
    // Perform rebuild
}
```

### Detailed Status Information

```go
status, err := validator.GetCacheStatus(ctx)
if err != nil {
    // Handle error
}

fmt.Printf("Cache valid: %v\n", status.IsValid)
fmt.Printf("Reason: %s\n", status.Reason)
fmt.Printf("Missing artifacts: %v\n", status.MissingArtifacts)
fmt.Printf("Newest source: %v\n", status.NewestSource)
fmt.Printf("Oldest artifact: %v\n", status.OldestArtifact)
```

## Cache Validation Logic

The validator checks the following conditions to determine cache validity:

1. **Artifacts directory exists**: `forge-artifacts/` directory must exist
2. **Artifacts present**: Directory must contain JSON files (indicating successful build)
3. **Source file timestamps**: All `.sol` files in `src/` must be older than artifacts
4. **Configuration timestamps**: `foundry.toml` must be older than artifacts

If any condition fails, the cache is considered invalid and a rebuild is needed.

## File Structure Expected

```
contracts-directory/
├── src/                    # Solidity source files (.sol)
├── forge-artifacts/        # Compiled artifacts (JSON files)
├── foundry.toml           # Foundry configuration
└── ...
```

## API Reference

### Types

#### `BuildCacheValidator`
Main validator struct that performs cache validation.

#### `CacheStatus`
Detailed information about cache state:
- `IsValid bool`: Whether cache is valid
- `Reason string`: Human-readable reason for cache state
- `NewestSource time.Time`: Timestamp of newest source file
- `OldestArtifact time.Time`: Timestamp of oldest artifact
- `MissingArtifacts bool`: Whether artifacts are missing

### Functions

#### `NewBuildCacheValidator(contractsDir string, logger *log.Logger) *BuildCacheValidator`
Creates a new validator instance. Logger is optional (uses default if nil).

#### `IsCacheValid(ctx context.Context) (bool, error)`
Returns true if build cache is valid, false if rebuild is needed.

#### `GetCacheStatus(ctx context.Context) (*CacheStatus, error)`
Returns detailed information about cache state.

## Testing

Run tests with:
```bash
go test -v ./op-acceptance-tests/buildcache/
```

The test suite covers:
- Missing artifacts directory
- Empty artifacts directory
- Stale artifacts (source files newer)
- Stale artifacts (foundry.toml newer)
- Valid cache scenarios
- Edge cases and error conditions
