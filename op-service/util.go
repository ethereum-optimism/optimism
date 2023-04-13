package op_service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func PrefixEnvVar(prefix, suffix string) string {
	return prefix + "_" + suffix
}

// CloseAction runs the function in the background, until it finishes or until it is closed by the user with an interrupt.
func CloseAction(fn func(ctx context.Context, shutdown <-chan struct{}) error) error {
	stopped := make(chan error, 1)
	shutdown := make(chan struct{}, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stopped <- fn(ctx, shutdown)
	}()

	doneCh := make(chan os.Signal, 1)
	signal.Notify(doneCh, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)

	select {
	case <-doneCh:
		cancel()
		shutdown <- struct{}{}

		select {
		case err := <-stopped:
			return err
		case <-time.After(time.Second * 10):
			return errors.New("command action is unresponsive for more than 10 seconds... shutting down")
		}
	case err := <-stopped:
		cancel()
		return err
	}
}

// ParseHttpHeader takes a slice of strings of the form "K=V" and returns a http.Header
func ParseHttpHeader(slice []string) http.Header {
	header := make(http.Header)
	for _, s := range slice {
		split := strings.SplitN(s, "=", 2)
		val := ""
		if len(split) >= 2 {
			val = split[1]
		}
		header.Add(split[0], val)
	}
	return header
}
