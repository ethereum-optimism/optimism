package filter

import (
	"net/http"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// newJWTProtectedAdminHandler creates an http.Handler for the admin API
// that requires JWT authentication. It uses go-ethereum's built-in
// HTTPHandlerStack which handles JWT validation.
func newJWTProtectedAdminHandler(backend *Backend, jwtSecret []byte) http.Handler {
	srv := rpc.NewServer()
	if err := srv.RegisterName("admin", &AdminFrontend{backend: backend}); err != nil {
		panic(err) // Programming error - should never happen
	}

	// Wrap with JWT authentication using go-ethereum's handler stack.
	// This is the same mechanism used by geth and op-service internally.
	return node.NewHTTPHandlerStack(srv, []string{"*"}, []string{"*"}, jwtSecret)
}
