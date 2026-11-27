package interopsmoke

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// smokeInvalidSpam sends invalid executing messages back to back on both
// chains for the given duration, without waiting for them to be reorged out.
// Each direction reuses one initiating message and points every executing
// message at a log index that does not exist, so every block that includes one
// must be invalidated. It is a load generator for reorg stress tests; use
// invalid-message to verify that individual invalid txs are reorged out.
func smokeInvalidSpam(env *smokeEnv, duration, interval time.Duration) error {
	dirs, err := invalidDirections(env)
	if err != nil {
		return err
	}
	stderr := &lockedWriter{w: env.stderr}
	for _, d := range dirs {
		eventLogger, err := d.initUser.deployEventLogger(env.ctx)
		if err != nil {
			return fmt.Errorf("%s: deploy EventLogger: %w", d.name, err)
		}
		d.eventLogger = eventLogger
	}
	deadline := time.Now().Add(duration)
	var wg sync.WaitGroup
	for _, d := range dirs {
		wg.Add(1)
		go func(d *invalidDirection) {
			defer wg.Done()
			var sent, failed atomic.Int64
			var initMsg *initMessage
			lastReport := time.Now()
			for time.Now().Before(deadline) && env.ctx.Err() == nil {
				if initMsg == nil {
					m, err := d.initUser.sendRandomInitMessage(env.ctx, d.rng, d.eventLogger, 2, 10)
					if err != nil {
						failed.Add(1)
						fmt.Fprintf(stderr, "    [%s] init message failed, retrying: %v\n", d.name, err)
						time.Sleep(2 * time.Second)
						continue
					}
					initMsg = m
					fmt.Fprintf(stderr, "    [%s] init message on %s block %d\n", d.name, d.initUser.chain.name, initMsg.BlockNumber())
				}
				exec, err := d.execUser.sendInvalidExecMessage(env.ctx, initMsg)
				switch {
				case err != nil:
					failed.Add(1)
					fmt.Fprintf(stderr, "    [%s] invalid exec failed, retrying: %v\n", d.name, err)
					// The init message may have been reorged out with its block; make a fresh one.
					initMsg = nil
					time.Sleep(2 * time.Second)
				case exec.Receipt.Status != types.ReceiptStatusSuccessful:
					failed.Add(1)
				default:
					sent.Add(1)
				}
				if time.Since(lastReport) > 30*time.Second {
					fmt.Fprintf(stderr, "%s [%s] sent=%d failed=%d last=%s block %v\n", time.Now().UTC().Format("15:04:05"), d.name, sent.Load(), failed.Load(), d.execUser.chain.name, blockOf(exec))
					lastReport = time.Now()
				}
				if interval > 0 {
					time.Sleep(interval)
				}
			}
			fmt.Fprintf(stderr, "    [%s] done: sent=%d failed=%d\n", d.name, sent.Load(), failed.Load())
		}(d)
	}
	wg.Wait()
	return env.ctx.Err()
}

func blockOf(e *execMessage) any {
	if e == nil || e.Receipt == nil {
		return "-"
	}
	return e.BlockNumber()
}
