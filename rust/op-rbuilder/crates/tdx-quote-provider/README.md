# TDX Quote Provider

This crate is a service intended to be run alongside the same TDX VM as op-rbuilder for flashtestations. This is a simple HTTP server that generates and returns an attestation quote by the requesting service.

TDX attestations uses configfs, which requires root access. Having a separate service allow applications like the op-rbuilder to request attestations without root access.

## Usage

You can run the server using [Cargo](https://doc.rust-lang.org/cargo/). To run the quote provider server:

```bash
cargo run -p tdx-quote-provider --bin tdx-quote-provider --
```

This will run a server that will generate and provide TDX attestation quotes on http:localhost:8181 by default. 

To run the server with a mock attestation quote for testing:

```bash
cargo run -p tdx-quote-provider --bin tdx-quote-provider \
  --mock \
  --mock-attestation-path /path/to/mock/quote.bin
```

### Command-line Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--service-host` | `SERVICE_HOST` | `127.0.0.1` | Host to run the HTTP server on |
| `--service-port` | `SERVICE_PORT` | `8181` | Port to run the HTTP server on |
| `--metrics` | `METRICS` | `false` | Enable Prometheus metrics |
| `--metrics-host` | `METRICS_HOST` | `127.0.0.1` | Host to run the metrics server on |
| `--metrics-port` | `METRICS_PORT` | `9090` | Port to run the metrics server on |
| `--mock` | `MOCK` | `false` | Use mock attestation for testing |
| `--mock-attestation-path` | `MOCK_ATTESTATION_PATH` | `""` | Path to the mock attestation file |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (trace, debug, info, warn, error) |
| `--log-format` | `LOG_FORMAT` | `text` | Log format (text, json) |

## Endpoints

### `GET /healthz`
Health check endpoint that returns `200 OK` if the server is running.

**Response:**
```
HTTP/1.1 200 OK
```

### `GET /attest/{appdata}`
Generates and returns a TDX attestation quote for the provided report data. In the case of op-rbuilder, this is the public key of the ethereum key pair generated during the bootstrapping step for flashtestations.

**Parameters:**
- `appdata` (path parameter): Hex-encoded 64-byte report data

**Response:**
- **Success (200 OK):** Binary attestation quote with `Content-Type: application/octet-stream`

**Example:**
```bash
curl -X GET "http://localhost:8181/attest/bbbbf586ac29a7b62fef9118e9d614179962d463419ebd905eb5ece84f2946dfccff93a66129a140ea49c49f7590c36143ad2aec7f8ed74aaa0ff479494a6493" \ # debug key for flashtestations
  -H "Accept: application/octet-stream" \
  --output attestation.bin
```

### Metrics Endpoint

When enabled with `--metrics`, Prometheus metrics are available at the configured metrics address (default: `http://localhost:9090`).

## Contributing

### Building & Testing

All tests and test data with a mock attestation quote are in the `/tests` folder.

```bash
# Build the project
cargo build -p tdx-quote-provider 

# Run all the tests 
cargo test -p tdx-quote-provider 
```

### Deployment

To build a docker image:

```bash
# Build from the workspace root
docker build -f crates/tdx-quote-provider/Dockerfile -t tdx-quote-provider .
```

Builds of the websocket proxy are provided on [Dockerhub](https://hub.docker.com/r/flashbots/tdx-quote-provider/tags).

You can see a full list of parameters by running:

`docker run flashbots/tdx-quote-provider:latest --help`

Example:

```bash
docker run flashbots/tdx-quote-provider:latest
```