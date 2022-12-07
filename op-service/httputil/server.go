package httputil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func ListenAndServeContext(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
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

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())

		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}
