package test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// TODO: CLI flag to specify endpoint

// TODO: typed client bindings for resources

var Endpoint = ""

const maxDialTime = 20 * time.Second
const maxDialAttempts = 5
const dialWait = 3 * time.Second

// We pool client connections to the op-test server,
// so they can be re-used between tests of the same package,
// and to make parallel interactions with the server easy.
var clientPool = sync.Pool{}

func getRPCClient(ctx context.Context) *rpc.Client {
	cl := clientPool.Get()
	if cl != nil {
		return cl.(*rpc.Client)
	}
	res, err := retry.Do(ctx, maxDialAttempts, retry.Fixed(dialWait), func() (*rpc.Client, error) {
		ctx, cancel := context.WithTimeout(context.Background(), maxDialTime)
		defer cancel()
		return rpc.DialContext(ctx, Endpoint)
	})
	if err != nil {
		panic(fmt.Errorf("failed to create RPC client to op-test server: %w", err))
	}
	// Try to be nice, and close the connection, as soon as GC gets ownership over it
	runtime.SetFinalizer(res, func() {
		res.Close()
	})
	return res
}

// Cleanup cleans up the op-test run.
// It empties the pool of temporary RPC clients, and closes all retrieved RPC clients.
// This can be called by TestMain(m *testing.M) after m.Run() to clean up open connections nicely before test shutdown.
func Cleanup() {
	for cl := clientPool.Get(); cl != nil; cl = clientPool.Get() {
		cl.(*rpc.Client).Close()
	}
}

// WriteToServer writes an RPC message to the global op-test server, using a pool of RPC clients.
func WriteToServer(ctx context.Context, result any, method string, args ...any) error {
	// We don't use sync.Pool.New(), so we can time-out/cancel the creation of the RPC client with a context.
	cl := getRPCClient(ctx)
	// When we are done with the client, put it in the pool, so we can re-use it for next write calls.
	// It may be GC'ed if collected by
	defer clientPool.Put(cl)

	return cl.CallContext(ctx, result, method, args...)
}

// Resource represents a unique instance of a test component on the op-test server.
type Resource string

// Do performs an action with the resource on the server.
func (res Resource) Do(t Executor, result any, args ...any) {
	// optest_do has three params: the test name, the resource ID, and a command to do as that resource.
	err := WriteToServer(t.Ctx(), result, "optest_do", t.Name(), string(res), args)
	require.NoError(t, err, "op-test server must do action with resource %s", string(res))
}

// RequestResource requests the server to provide us with a handle to a resource that matches the provided settings.
func RequestResource(t Executor, settings any) Resource {
	var res string
	// server is trusted, don't have to further validate the resource ID we get
	err := WriteToServer(t.Ctx(), &res, "optest_request", t.Name(), settings)
	require.NoError(t, err, "op-test server must provide resource")
	return Resource(res)
}
