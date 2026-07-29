# websocket-proxy

## Command-line Options

### Connection Configuration

- `--listen-addr <ADDRESS>`: The address and port to listen on for incoming connections (default: 0.0.0.0:8545)
- `--upstream-ws <URI>`: WebSocket URI of the upstream server to connect to (required, can specify multiple with comma separation)

### Rate Limiting

- `--instance-connection-limit <LIMIT>`: Maximum number of concurrently connected clients per instance (default: 100)
- `--per-ip-connection-limit <LIMIT>`: Maximum number of concurrently connected clients per IP (default: 10)
- `--redis-url <URL>`: Redis URL for distributed rate limiting (e.g., redis://localhost:6379). If not provided, in-memory rate limiting will be used
- `--redis-key-prefix <PREFIX>`: Prefix for Redis keys (default: flashblocks)

### Message Handling

- `--message-buffer-size <SIZE>`: Number of messages to buffer for lagging clients (default: 20)
- `--enable-compression`: Enable Brotli compression on messages to downstream clients (default: false)
- `--ip-addr-http-header <HEADER>`: Header to use to determine the client's origin IP (default: X-Forwarded-For)

### Authentication

- `--api-keys <KEYS>`: API keys to allow in format `<app1>:<apiKey1>,<app2>:<apiKey2>`. If not provided, the endpoint will be unauthenticated

### Logging

- `--log-level <LEVEL>`: Log level (default: info)
- `--log-format <FORMAT>`: Format for logs, can be json or text (default: text)

### Metrics

- `--metrics`: Enable Prometheus metrics (default: true)
- `--metrics-addr <ADDRESS>`: Address to run the metrics server on (default: 0.0.0.0:9000)
- `--metrics-global-labels <LABELS>`: Tags to add to every metrics emitted in format `label1=value1,label2=value2` (default: "")
- `--metrics-host-label`: Add the hostname as a label to all Prometheus metrics (default: false)

### Upstream Connection Management

- `--subscriber-max-interval-ms <MS>`: Maximum backoff allowed for upstream connections in milliseconds (default: 20000)
- `--subscriber-ping-interval-ms <MS>`: Interval in milliseconds between ping messages sent to upstream servers to detect unresponsive connections (default: 2000)
- `--subscriber-pong-timeout-ms <MS>`: Timeout in milliseconds to wait for pong responses from upstream servers before considering the connection dead (default: 4000)

### Client Health Checks

- `--client-ping-enabled`: Enable ping/pong client health checks (default: false)
- `--client-ping-interval-ms <MS>`: Interval in milliseconds to send ping messages to clients (default: 15000)
- `--client-pong-timeout-ms <MS>`: Timeout in milliseconds to wait for pong response from clients (default: 30000)
