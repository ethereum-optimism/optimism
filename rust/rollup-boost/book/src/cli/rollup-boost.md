# rollup-boost

### Command-line Options

- `--l2-jwt-token <TOKEN>`: JWT token for L2 authentication (required)
- `--l2-jwt-path <PATH>`: Path to the L2 JWT secret file (required if `--l2-jwt-token` is not provided)
- `--l2-url <URL>`: URL of the local L2 execution engine (required)
- `--builder-url <URL>`: URL of the builder execution engine (required)
- `--builder-jwt-token <TOKEN>`: JWT token for builder authentication (required)
- `--builder-jwt-path <PATH>`: Path to the builder JWT secret file (required if `--builder-jwt-token` is not provided)
- `--rpc-host <HOST>`: Host to run the server on (default: 127.0.0.1)
- `--rpc-port <PORT>`: Port to run the server on (default: 8081)
- `--tracing`: Enable tracing (default: false)
- `--log-level <LEVEL>`: Log level (default: info)
- `--log-format <FORMAT>`: Log format (default: text)
- `--metrics`: Enable metrics (default: false)
- `--metrics-host <METRICS_HOST>`: Host to run the metrics server on (default: 127.0.0.1)
- `--debug-host <HOST>`: Host to run the server on (default: 127.0.0.1)
- `--debug-server-port <PORT>`: Port to run the debug server on (default: 5555)