package cmd

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

func oracleForTest(t *testing.T, timeout time.Duration) (*ProcessPreimageOracle, preimage.FileChannel, preimage.FileChannel) {
	t.Helper()
	pClient, pHost, err := preimage.CreateBidirectionalChannel()
	require.NoError(t, err)
	hClient, hHost, err := preimage.CreateBidirectionalChannel()
	require.NoError(t, err)
	ctx, cancel := context.WithCancelCause(t.Context())
	t.Cleanup(func() {
		cancel(context.Canceled)
		_ = pClient.Close()
		_ = pHost.Close()
		_ = hClient.Close()
		_ = hHost.Close()
	})
	return &ProcessPreimageOracle{pIO: pClient, hIO: hClient, ioCtx: ctx, timeout: timeout, cancelIO: cancel}, pHost, hHost
}

func guardedOracleCall(fn func()) error {
	_, err := Guard(nil, func(bool) (*mipsevm.StepWitness, error) {
		fn()
		return nil, nil
	})(false)
	return err
}

func awaitOracleResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(20 * time.Second):
		t.Fatal("oracle request did not complete")
		return nil
	}
}

func TestProcessPreimageOracleTimeout(t *testing.T) {
	for _, partialPayload := range []bool{false, true} {
		name := "missing length"
		if partialPayload {
			name = "partial payload"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			oracle, host, _ := oracleForTest(t, 5*time.Second)
			key := [32]byte{1, 2, 3}
			if partialPayload {
				// Queue the partial response before starting the request, so the
				// deadline exercises the payload read rather than host scheduling.
				require.NoError(t, binary.Write(host, binary.BigEndian, uint64(4)))
				_, err := host.Write([]byte{1, 2})
				require.NoError(t, err)
			}
			result := make(chan error, 1)
			go func() {
				result <- guardedOracleCall(func() { oracle.GetPreimage(key) })
			}()
			var received [32]byte
			require.NoError(t, host.Reader().SetReadDeadline(time.Now().Add(5*time.Second)))
			_, err := io.ReadFull(host, received[:])
			require.NoError(t, err)
			require.Equal(t, key, received)
			// Keep the peer open and alive, as when a host only logs a failed request.
			err = awaitOracleResult(t, result)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			if partialPayload {
				require.ErrorContains(t, err, "failed to read pre-image payload")
			} else {
				require.ErrorContains(t, err, "failed to read pre-image length")
			}
		})
	}
}

func TestProcessPreimageOracleHintTimeout(t *testing.T) {
	t.Parallel()
	oracle, _, host := oracleForTest(t, 5*time.Second)
	result := make(chan error, 1)
	go func() { result <- guardedOracleCall(func() { oracle.Hint([]byte("hint")) }) }()
	var message [8]byte
	require.NoError(t, host.Reader().SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err := io.ReadFull(host, message[:])
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 4, 'h', 'i', 'n', 't'}, message[:])
	require.ErrorIs(t, awaitOracleResult(t, result), context.DeadlineExceeded)
}

func TestProcessPreimageOracleRepeatedRequests(t *testing.T) {
	t.Parallel()
	oracle, preimageHost, hintHost := oracleForTest(t, 10*time.Second)
	key := [32]byte{42}
	payload := []byte{1, 2, 3, 4}
	result := make(chan error, 1)
	go func() {
		for range 2 {
			if err := preimage.NewOracleServer(preimageHost).NextPreimageRequest(func(received [32]byte) ([]byte, error) {
				if received != key {
					return nil, errors.New("unexpected preimage key")
				}
				return payload, nil
			}); err != nil {
				result <- err
				return
			}
			if err := preimage.NewHintReader(hintHost).NextHint(func(hint string) error {
				if hint != "hint" {
					return errors.New("unexpected hint")
				}
				return nil
			}); err != nil {
				result <- err
				return
			}
		}
		result <- nil
	}()
	for range 2 {
		require.Equal(t, payload, oracle.GetPreimage(key))
		oracle.Hint([]byte("hint"))
	}
	require.NoError(t, awaitOracleResult(t, result))
}

func TestProcessPreimageOracleHostCancellation(t *testing.T) {
	t.Parallel()
	oracle, _, _ := oracleForTest(t, defaultOracleTimeout)
	oracle.cancelIO(errors.New("host exited"))
	require.ErrorIs(t, guardedOracleCall(func() { oracle.GetPreimage([32]byte{}) }), context.Canceled)
	require.ErrorIs(t, guardedOracleCall(func() { oracle.Hint([]byte("hint")) }), context.Canceled)
}

func TestProcessPreimageOracleNoHost(t *testing.T) {
	t.Parallel()
	oracle, err := NewProcessPreimageOracle(nil, "", nil, nil, nil)
	require.NoError(t, err)
	require.NotPanics(t, func() { oracle.Hint(nil) })
	require.PanicsWithValue(t, "no pre-image retriever available", func() { oracle.GetPreimage([32]byte{}) })
	require.NoError(t, oracle.Start())
	require.NoError(t, oracle.Close())
}

func TestOracleTimeoutFlag(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want time.Duration
	}{
		{name: "default", want: 10 * time.Minute},
		{name: "override", args: []string{"--oracle-timeout", "30m"}, want: 30 * time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{CreateRunCommand(func(ctx *cli.Context) error {
				require.Equal(t, test.want, ctx.Duration(RunOracleTimeoutFlag.Name))
				return nil
			})}}
			args := append([]string{"cannon", "run", "--input", "unused"}, test.args...)
			require.NoError(t, app.Run(args))
		})
	}
	for _, value := range []string{"0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{CreateRunCommand(Run)}}
			err := app.Run([]string{"cannon", "run", "--input", "unused", "--oracle-timeout", value})
			require.EqualError(t, err, "oracle timeout must be positive")
		})
	}
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := newProcessPreimageOracle(nil, "", nil, nil, nil, timeout)
		require.EqualError(t, err, "oracle timeout must be positive")
	}
}

func TestGuardWithoutProcessState(t *testing.T) {
	t.Parallel()
	t.Run("panic error", func(t *testing.T) {
		require.ErrorIs(t, guardedOracleCall(func() { panic(context.DeadlineExceeded) }), context.DeadlineExceeded)
	})
	t.Run("panic string", func(t *testing.T) {
		require.ErrorContains(t, guardedOracleCall(func() { panic("oracle failure") }), "oracle failure")
	})
	t.Run("returned error", func(t *testing.T) {
		_, err := Guard(nil, func(bool) (*mipsevm.StepWitness, error) {
			return nil, context.DeadlineExceeded
		})(false)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
