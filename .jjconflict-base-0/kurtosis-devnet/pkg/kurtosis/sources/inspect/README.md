# Kurtosis Inspect Tool

A command-line tool for inspecting Kurtosis enclaves and extracting conductor configurations and environment data from running Optimism devnets.

## Overview

The Kurtosis Inspect Tool provides a clean interface to:

- 🔍 **Inspect running Kurtosis enclaves** - Extract service information and file artifacts
- 🎛️ **Generate conductor configurations** - Create TOML configs for `op-conductor-ops`
- 📊 **Export environment data** - Save complete devnet information as JSON
- 🔧 **Fix Traefik issues** - Repair missing network labels on containers

## Installation

### Build from Source

```bash
cd optimism/kurtosis-devnet
go build -o kurtosis-inspect pkg/kurtosis/sources/inspect/cmd/main.go
```

### Run Directly

```bash
go run pkg/kurtosis/sources/inspect/cmd/main.go [options] <enclave-id>
```

## Usage

### Basic Inspection

Inspect a running enclave and display results:

```bash
./kurtosis-inspect my-devnet-enclave
```

### Extract Conductor Configuration

Generate a conductor configuration file for use with `op-conductor-ops`:

```bash
./kurtosis-inspect --conductor-config conductor.toml my-devnet-enclave
```

### Export Complete Environment

Save the complete environment data as JSON:

```bash
./kurtosis-inspect --environment environment.json my-devnet-enclave
```

### Combined Export

Extract both conductor config and environment data:

```bash
./kurtosis-inspect \
  --conductor-config conductor.toml \
  --environment environment.json \
  my-devnet-enclave
```

### Fix Traefik Network Issues

Repair missing Traefik labels on containers:

```bash
./kurtosis-inspect --fix-traefik my-devnet-enclave
```

## Configuration Options

### CLI Flags

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--conductor-config` | `KURTOSIS_INSPECT_CONDUCTOR_CONFIG` | Path to write conductor configuration TOML file |
| `--environment` | `KURTOSIS_INSPECT_ENVIRONMENT` | Path to write environment JSON file |
| `--fix-traefik` | `KURTOSIS_INSPECT_FIX_TRAEFIK` | Fix missing Traefik labels on containers |
| `--log.level` | `KURTOSIS_INSPECT_LOG_LEVEL` | Logging level (DEBUG, INFO, WARN, ERROR) |
| `--log.format` | `KURTOSIS_INSPECT_LOG_FORMAT` | Log format (text, json, logfmt) |

### Environment Variables

All flags can be set via environment variables with the `KURTOSIS_INSPECT_` prefix:

```bash
export KURTOSIS_INSPECT_CONDUCTOR_CONFIG="/tmp/conductor.toml"
export KURTOSIS_INSPECT_ENVIRONMENT="/tmp/environment.json"
export KURTOSIS_INSPECT_LOG_LEVEL="DEBUG"

./kurtosis-inspect my-devnet-enclave
```

## Output Formats

### Conductor Configuration (TOML)

The conductor configuration file is compatible with `op-conductor-ops`:

```toml
[networks]
  [networks.2151908-chain0-kona]
    sequencers = ["op-conductor-2151908-chain0-kona-sequencer"]
  [networks.2151908-chain0-optimism]
    sequencers = ["op-conductor-2151908-chain0-optimism-sequencer"]

[sequencers]
  [sequencers.op-conductor-2151908-chain0-kona-sequencer]
    raft_addr = "127.0.0.1:60135"
    conductor_rpc_url = "http://127.0.0.1:60134"
    node_rpc_url = "http://127.0.0.1:60048"
    voting = true
  [sequencers.op-conductor-2151908-chain0-optimism-sequencer]
    raft_addr = "127.0.0.1:60176"
    conductor_rpc_url = "http://127.0.0.1:60177"
    node_rpc_url = "http://127.0.0.1:60062"
    voting = true
```

### Environment Data (JSON)

Complete environment data including services and file artifacts:

```json
{
  "FileArtifacts": [
    "genesis-l1.json",
    "genesis-l2-chain0.json",
    "jwt.txt",
    "rollup-l2-chain0.json"
  ],
  "UserServices": {
    "op-node-chain0-sequencer": {
      "Labels": {
        "app": "op-node",
        "chain": "chain0",
        "role": "sequencer"
      },
      "Ports": {
        "rpc": {
          "Host": "127.0.0.1",
          "Port": 9545
        },
        "p2p": {
          "Host": "127.0.0.1",
          "Port": 9222
        }
      }
    }
  }
}
```

## Integration with op-conductor-ops

### 1. Generate Conductor Configuration

```bash
# Extract conductor config from running devnet
./kurtosis-inspect --conductor-config conductor.toml my-devnet

# Use with op-conductor-ops
cd infra/op-conductor-ops
python op-conductor-ops.py --config ../../kurtosis-devnet/conductor.toml status
```

### 2. Leadership Transfer Example

```bash
# Generate config and perform leadership transfer
./kurtosis-inspect --conductor-config conductor.toml my-devnet
cd infra/op-conductor-ops
python op-conductor-ops.py --config ../../kurtosis-devnet/conductor.toml \
  transfer-leadership \
  --target-sequencer "op-conductor-2151908-chain0-optimism-sequencer"
```

## Examples

### Simple Devnet

```bash
# Deploy simple devnet
cd kurtosis-devnet
just devnet simple

# Inspect and extract configs
./kurtosis-inspect --conductor-config tests/simple-conductor.toml simple-devnet

# Check conductor status
cd ../infra/op-conductor-ops
python op-conductor-ops.py --config ../../kurtosis-devnet/tests/simple-conductor.toml status
```

### Multi-Chain Interop

```bash
# Deploy interop devnet
just devnet interop

# Extract complex conductor configuration
./kurtosis-inspect \
  --conductor-config tests/interop-conductor.toml \
  --environment tests/interop-environment.json \
  interop-devnet

# View conductor cluster status
cd ../infra/op-conductor-ops
python op-conductor-ops.py --config ../../kurtosis-devnet/tests/interop-conductor.toml status
```

### Debugging Network Issues

```bash
# Fix Traefik network issues
./kurtosis-inspect --fix-traefik my-devnet

# Inspect with debug logging
./kurtosis-inspect --log.level DEBUG --log.format json my-devnet
```

## Architecture

The tool follows a clean architecture pattern with clear separation of concerns:

```
pkg/kurtosis/sources/inspect/
├── cmd/main.go              # CLI setup and entry point
├── config.go                # Configuration parsing and validation
├── service.go               # Business logic and service layer
├── conductor.go             # Conductor configuration extraction
├── inspect.go               # Core inspection functionality
├── flags/
│   ├── flags.go            # CLI flag definitions
│   └── flags_test.go       # Flag testing
└── *_test.go               # Comprehensive test suite
```

### Key Components

- **Config**: Handles CLI argument parsing and validation
- **InspectService**: Main business logic for inspection operations
- **ConductorConfig**: Data structures for conductor configuration
- **Inspector**: Core enclave inspection functionality

## Testing

### Run All Tests

```bash
go test ./pkg/kurtosis/sources/inspect/... -v
```

### Test Coverage

```bash
go test ./pkg/kurtosis/sources/inspect/... -cover
```

### Test Categories

- **Unit Tests**: Individual component functionality
- **Integration Tests**: File I/O and configuration parsing
- **Real-World Tests**: Based on actual devnet configurations
- **Error Tests**: Permission and validation error handling

## Troubleshooting

### Common Issues

#### Kurtosis Engine Not Running

```
Error: failed to create Kurtosis context: The Kurtosis Engine Server is unavailable
```

**Solution:**
```bash
kurtosis engine start
```

#### Enclave Not Found

```
Error: failed to get enclave: enclave with identifier 'my-devnet' not found
```

**Solution:**
```bash
# List available enclaves
kurtosis enclave ls

# Use correct enclave name
./kurtosis-inspect <correct-enclave-name>
```

#### Permission Denied

```
Error: error creating conductor config file: permission denied
```

**Solution:**
```bash
# Ensure write permissions to output directory
chmod 755 /output/directory
```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
./kurtosis-inspect --log.level DEBUG --log.format json my-devnet
```

## Contributing

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
go test ./pkg/kurtosis/sources/inspect/... -v

# Build
go build -o kurtosis-inspect pkg/kurtosis/sources/inspect/cmd/main.go
```

### Adding New Features

1. Add functionality to appropriate service layer
2. Create comprehensive tests with real data
3. Update CLI flags if needed
4. Update this README with examples

## Related Tools

- **[op-conductor-ops](../../infra/op-conductor-ops/)**: Python CLI for managing conductor clusters
- **[Kurtosis](https://kurtosis.com/)**: Orchestration platform for development environments
- **[Optimism Devnet](../)**: Kurtosis package for Optimism development networks

## License

This tool is part of the Optimism monorepo and follows the same licensing terms. 