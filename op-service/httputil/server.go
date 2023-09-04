package httputil

import (
	"context"
	"errors"
	"net/http"
)

func ListenAndServeContext(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

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
