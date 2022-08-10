package httputil

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func ListenAndServeContext(ctx context.Context, server *http.Server) error {
	errCh := make(chan error)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// verify that the server comes up
	tick := time.NewTimer(10 * time.Millisecond)
	select {
	case err := <-errCh:
		return fmt.Errorf("http server failed: %w", err)
	case <-tick.C:
		break
	}

	<-ctx.Done()
	return ctx.Err()
}
