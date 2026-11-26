package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

var httpRegex = regexp.MustCompile("^http(s)?://")

type RPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error)
}

type rpcConfig struct {
	gethRPCOptions   []rpc.ClientOption
	recorder         jsonrpc.Recorder
	httpPollInterval time.Duration
	backoffAttempts  int
	limit            float64
	burst            int
	lazy             bool
	callTimeout      time.Duration
	batchCallTimeout time.Duration
	fixedDialBackoff time.Duration
	connectTimeout   time.Duration
	maxResponseSize  int64
}

type RPCOption func(cfg *rpcConfig)

func WithConnectTimeout(d time.Duration) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.connectTimeout = d
	}
}

func WithCallTimeout(d time.Duration) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.callTimeout = d
	}
}

func WithBatchCallTimeout(d time.Duration) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.batchCallTimeout = d
	}
}

// WithDialAttempts configures the number of attempts for the initial dial to the RPC,
// attempts are executed with an exponential backoff strategy by default.
func WithDialAttempts(attempts int) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.backoffAttempts = attempts
	}
}

// WithFixedDialBackoff makes the RPC client use a fixed delay between dial attempts of 2 seconds instead of exponential
func WithFixedDialBackoff(d time.Duration) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.fixedDialBackoff = d
	}
}

// WithMaxResponseSize sets the largest JSON-RPC response, in bytes, that the client reads over
// HTTP or websocket. A call whose response is larger fails with ErrResponseTooLarge over HTTP,
// and with a websocket read limit error over websocket. The default, also for a size of 0 or
// less, is DefaultMaxResponseSize.
func WithMaxResponseSize(size int64) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.maxResponseSize = size
	}
}

// WithHttpPollInterval configures the RPC to poll at the given rate, in case RPC subscriptions are not available.
func WithHttpPollInterval(duration time.Duration) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.httpPollInterval = duration
	}
}

// WithGethRPCOptions passes the list of go-ethereum RPC options to the internal RPC instance.
func WithGethRPCOptions(gethRPCOptions ...rpc.ClientOption) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.gethRPCOptions = append(cfg.gethRPCOptions, gethRPCOptions...)
	}
}

// WithRateLimit configures the RPC to target the given rate limit (in requests / second).
// See NewRateLimitingClient for more details.
func WithRateLimit(rateLimit float64, burst int) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.limit = rateLimit
		cfg.burst = burst
	}
}

// WithLazyDial makes the RPC client initialization defer the initial connection attempt,
// and defer to later RPC requests upon subsequent dial errors.
// Any dial-backoff option will be ignored if this option is used.
func WithLazyDial() RPCOption {
	return func(cfg *rpcConfig) {
		cfg.lazy = true
	}
}

// WithRPCRecorder makes the RPC client report its requests and their responses to the given
// recorder, if it is not nil. Subscriptions are not recorded.
// Warning: this overwrites any previous recorder choice.
func WithRPCRecorder(recorder jsonrpc.Recorder) RPCOption {
	return func(cfg *rpcConfig) {
		cfg.recorder = recorder
	}
}

// NewRPC returns the correct client.RPC instance for a given RPC url.
func NewRPC(ctx context.Context, lgr log.Logger, addr string, opts ...RPCOption) (RPC, error) {
	cfg := applyOptions(opts)

	var wrapped RPC
	if cfg.lazy {
		wrapped = newLazyRPC(addr, cfg)
	} else {
		underlying, err := dialRPCClientWithBackoff(ctx, lgr, addr, cfg)
		if err != nil {
			return nil, err
		}
		wrapped = wrapClient(underlying, cfg)
	}

	return NewRPCWithClient(ctx, lgr, addr, wrapped, cfg.httpPollInterval)
}

func applyOptions(opts []RPCOption) rpcConfig {
	var cfg rpcConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.connectTimeout == 0 {
		cfg.connectTimeout = 10 * time.Second
	}
	if cfg.backoffAttempts < 1 { // default to at least 1 attempt, or it always fails to dial.
		cfg.backoffAttempts = 1
	}
	if cfg.callTimeout == 0 {
		cfg.callTimeout = 10 * time.Second
	}
	if cfg.batchCallTimeout == 0 {
		cfg.batchCallTimeout = 20 * time.Second
	}
	if cfg.maxResponseSize <= 0 {
		cfg.maxResponseSize = DefaultMaxResponseSize
	}
	return cfg
}

// NewRPCWithClient builds a new polling client with the given underlying RPC client.
func NewRPCWithClient(ctx context.Context, lgr log.Logger, addr string, underlying RPC, pollInterval time.Duration) (RPC, error) {
	if httpRegex.MatchString(addr) {
		underlying = NewPollingClient(ctx, lgr, underlying, WithPollRate(pollInterval))
	}
	return underlying, nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, cfg rpcConfig) (*rpc.Client, error) {
	bOff := retry.Exponential()
	if cfg.fixedDialBackoff != 0 {
		bOff = retry.Fixed(cfg.fixedDialBackoff)
	}
	return retry.Do(ctx, cfg.backoffAttempts, bOff, func() (*rpc.Client, error) {
		return checkAndDial(ctx, log, addr, cfg.connectTimeout, cfg.maxResponseSize, cfg.gethRPCOptions...)
	})
}

// CheckAndDial dials addr, after checking that it accepts TCP connections, with responses
// limited to DefaultMaxResponseSize.
func CheckAndDial(ctx context.Context, log log.Logger, addr string, connectTimeout time.Duration, options ...rpc.ClientOption) (*rpc.Client, error) {
	return checkAndDial(ctx, log, addr, connectTimeout, DefaultMaxResponseSize, options...)
}

func checkAndDial(ctx context.Context, log log.Logger, addr string, connectTimeout time.Duration, maxResponseSize int64, options ...rpc.ClientOption) (*rpc.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if !IsURLAvailable(ctx, addr, connectTimeout) {
		log.Warn("failed to dial address, but may connect later", "addr", addr)
		return nil, fmt.Errorf("address unavailable (%s)", addr)
	}
	client, err := dialRPCWithOptions(ctx, addr, maxResponseSize, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial address (%s): %w", addr, err)
	}
	return client, nil
}

// DefaultMaxResponseSize is the default limit on the size of a single JSON-RPC response (a batch
// response counts as one). It is above the largest L2 block that derivation accepts: a channel
// holds at most 100 MB of batch data, about 200 MB as hex in JSON. Clients that fetch block
// receipts need MaxReceiptsResponseSize.
const DefaultMaxResponseSize = 256 << 20

// MaxReceiptsResponseSize is the response size limit for clients that fetch block receipts. A
// receipts response can be about one byte of JSON per gas: an empty LOG0 costs 375 gas and adds
// a log object of about 370 bytes. That makes about 500 MB for an L2 block at 500M gas, the
// largest gas limit that SystemConfig accepts, and leaves room for L1 gas limits up to about 1G.
const MaxReceiptsResponseSize = 1 << 30

// dialRPCWithOptions dials addr with a limit of maxResponseSize bytes for each response, over HTTP
// and over websocket. Over HTTP it counts the body after net/http inflates it; websocket
// messages aren't compressed. options must not replace the HTTP client, or the limit is lost
// over HTTP.
func dialRPCWithOptions(ctx context.Context, addr string, maxResponseSize int64, options ...rpc.ClientOption) (*rpc.Client, error) {
	httpClient := &http.Client{Transport: &limitedTransport{base: http.DefaultTransport, limit: maxResponseSize}}
	options = append([]rpc.ClientOption{
		rpc.WithHTTPClient(httpClient),
		rpc.WithWebsocketMessageSizeLimit(maxResponseSize),
	}, options...)
	return rpc.DialOptions(ctx, addr, options...)
}

func IsURLAvailable(ctx context.Context, address string, timeout time.Duration) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	addr := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "http", "ws":
			addr += ":80"
		case "https", "wss":
			addr += ":443"
		default:
			// Fail open if we can't figure out what the port should be
			return true
		}
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// BaseRPCClient is a wrapper around a concrete *rpc.Client instance to make it compliant
// with the client.RPC interface.
// It sets a default timeout of 10s on CallContext & 20s on BatchCallContext made through it.
type BaseRPCClient struct {
	c                *rpc.Client
	batchCallTimeout time.Duration
	callTimeout      time.Duration
}

func NewBaseRPCClient(c *rpc.Client, opts ...RPCOption) RPC {
	cfg := applyOptions(opts)
	return wrapClient(c, cfg)
}

func wrapClient(c *rpc.Client, cfg rpcConfig) RPC {
	var wrapped RPC
	wrapped = &BaseRPCClient{c: c, callTimeout: cfg.callTimeout, batchCallTimeout: cfg.batchCallTimeout}

	if cfg.recorder != nil {
		wrapped = newRecordingRPC(wrapped, cfg.recorder)
	}
	if cfg.limit != 0 {
		wrapped = NewRateLimitingClient(wrapped, rate.Limit(cfg.limit), cfg.burst)
	}
	return wrapped
}

func (b *BaseRPCClient) Close() {
	b.c.Close()
}

type ErrorDataProvider interface {
	ErrorData() interface{}
}

func (b *BaseRPCClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	cCtx, cancel := context.WithTimeout(ctx, b.callTimeout)
	defer cancel()
	err := b.c.CallContext(cCtx, result, method, args...)
	if ed, ok := err.(ErrorDataProvider); ok && ed.ErrorData() != nil {
		err = fmt.Errorf("%w: %v", err, ed.ErrorData())
	}
	return err
}

func (b *BaseRPCClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	cCtx, cancel := context.WithTimeout(ctx, b.batchCallTimeout)
	defer cancel()
	err := b.c.BatchCallContext(cCtx, batch)
	if ed, ok := err.(ErrorDataProvider); ok && ed.ErrorData() != nil {
		err = fmt.Errorf("%w: %v", err, ed.ErrorData())
	}
	return err
}

func (b *BaseRPCClient) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	return b.c.Subscribe(ctx, namespace, channel, args...)
}
