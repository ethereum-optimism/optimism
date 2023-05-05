package op_service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func PrefixEnvVar(prefix, suffix string) string {
	return prefix + "_" + suffix
}

// ParseAddress parses an ETH address from a hex string. This method will fail if
// the address is not a valid hexadecimal address.
func ParseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
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
