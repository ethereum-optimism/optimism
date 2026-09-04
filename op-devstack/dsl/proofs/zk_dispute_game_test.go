package proofs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type claimReadResult struct {
	claim ZKClaimData
	err   error
}

func challengedClaim(prover common.Address) ZKClaimData {
	return ZKClaimData{Status: uint8(ZKProposalChallenged), Prover: prover}
}

func newTestZKGame(t *testing.T, read func(context.Context) (ZKClaimData, error)) *ZKGame {
	return newTestZKGameWithT(devtest.SerialT(t), read)
}

func newTestZKGameWithT(t devtest.T, read func(context.Context) (ZKClaimData, error)) *ZKGame {
	return &ZKGame{
		t:             t,
		Address:       common.HexToAddress("0x1234"),
		claimDataRead: read,
	}
}

func sequenceClaimReader(results ...claimReadResult) func(context.Context) (ZKClaimData, error) {
	next := 0
	return func(context.Context) (ZKClaimData, error) {
		index := min(next, len(results)-1)
		next++
		return results[index].claim, results[index].err
	}
}

func TestVerifyUnproven(t *testing.T) {
	t.Run("challenged without prover", func(t *testing.T) {
		game := newTestZKGame(t, sequenceClaimReader(claimReadResult{claim: challengedClaim(common.Address{})}))

		claim, err := game.verifyUnproven(context.Background())

		require.NoError(t, err)
		require.Equal(t, challengedClaim(common.Address{}), claim)
	})

	t.Run("status changed", func(t *testing.T) {
		claim := challengedClaim(common.Address{})
		claim.Status = uint8(ZKProposalChallengedAndValidProofProvided)
		game := newTestZKGame(t, sequenceClaimReader(claimReadResult{claim: claim}))

		_, err := game.verifyUnproven(context.Background())

		require.ErrorContains(t, err, "expected proposal status")
		require.ErrorContains(t, err, "observed")
	})

	t.Run("prover appeared", func(t *testing.T) {
		prover := common.HexToAddress("0xbeef")
		game := newTestZKGame(t, sequenceClaimReader(claimReadResult{claim: challengedClaim(prover)}))

		_, err := game.verifyUnproven(context.Background())

		require.ErrorContains(t, err, prover.Hex())
	})

	t.Run("read failed", func(t *testing.T) {
		readErr := errors.New("RPC unavailable")
		game := newTestZKGame(t, sequenceClaimReader(claimReadResult{err: readErr}))

		_, err := game.verifyUnproven(context.Background())

		require.ErrorIs(t, err, readErr)
	})
}

func TestReadClaimDataBoundsIndividualRead(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		game := newTestZKGameWithT(devT, func(ctx context.Context) (ZKClaimData, error) {
			<-ctx.Done()
			return ZKClaimData{}, ctx.Err()
		})
		start := time.Now()

		_, err := game.readClaimData(ctx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, time.Minute, time.Since(start))
	})
}

func TestVerifyUnprovenForRejectsNonPositiveWindow(t *testing.T) {
	reads := 0
	game := newTestZKGame(t, func(context.Context) (ZKClaimData, error) {
		reads++
		return challengedClaim(common.Address{}), nil
	})

	for _, window := range []time.Duration{0, -time.Second} {
		err := game.verifyUnprovenFor(context.Background(), window)
		require.ErrorContains(t, err, "must be positive")
	}
	require.Zero(t, reads)
}

func TestVerifyUnprovenForObservesAtOrAfterBoundary(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var observedAt []time.Time
		game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
			observedAt = append(observedAt, time.Now())
			return challengedClaim(common.Address{}), nil
		})

		err := game.verifyUnprovenFor(context.Background(), 2500*time.Millisecond)

		require.NoError(t, err)
		require.Len(t, observedAt, 4)
		for i, expectedOffset := range []time.Duration{0, time.Second, 2 * time.Second, 2500 * time.Millisecond} {
			require.Equal(t, start.Add(expectedOffset), observedAt[i])
		}
		require.False(t, observedAt[len(observedAt)-1].Before(start.Add(2500*time.Millisecond)))
	})
}

func TestVerifyUnprovenForDoesNotCountReadStartedBeforeBoundary(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		reads := 0
		game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
			reads++
			if reads == 2 {
				timer := time.NewTimer(time.Second)
				<-timer.C
				return challengedClaim(common.Address{}), nil
			}
			if reads == 3 {
				claim := challengedClaim(common.Address{})
				claim.Status = uint8(ZKProposalChallengedAndValidProofProvided)
				return claim, nil
			}
			return challengedClaim(common.Address{}), nil
		})

		err := game.verifyUnprovenFor(context.Background(), 1500*time.Millisecond)

		require.ErrorContains(t, err, "changed before")
		require.Equal(t, 3, reads)
	})
}

func TestVerifyUnprovenForFailsImmediatelyWhenGameChanges(t *testing.T) {
	tests := []struct {
		name      string
		claim     ZKClaimData
		wantError string
	}{
		{
			name: "status changed",
			claim: ZKClaimData{
				Status: uint8(ZKProposalChallengedAndValidProofProvided),
			},
			wantError: "expected proposal status",
		},
		{
			name:      "prover appeared",
			claim:     challengedClaim(common.HexToAddress("0xbeef")),
			wantError: "expected no prover",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			devT := devtest.SerialT(t)
			synctest.Test(t, func(t *testing.T) {
				reads := 0
				game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
					reads++
					if reads == 1 {
						return challengedClaim(common.Address{}), nil
					}
					return test.claim, nil
				})

				err := game.verifyUnprovenFor(context.Background(), time.Hour)

				require.ErrorContains(t, err, "changed before")
				require.ErrorContains(t, err, test.wantError)
				require.Equal(t, 2, reads)
			})
		})
	}
}

func TestVerifyUnprovenForRetriesTransientReads(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		readErr := errors.New("transient RPC failure")
		reads := 0
		game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
			reads++
			if reads == 2 {
				return ZKClaimData{}, readErr
			}
			return challengedClaim(common.Address{}), nil
		})

		err := game.verifyUnprovenFor(context.Background(), 2*time.Second)

		require.NoError(t, err)
		require.GreaterOrEqual(t, reads, 3)
	})
}

func TestVerifyUnprovenForReportsPersistentReadFailure(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		readErr := errors.New("persistent RPC failure")
		reads := 0
		game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
			reads++
			if reads == 1 {
				return challengedClaim(common.Address{}), nil
			}
			return ZKClaimData{}, readErr
		})

		err := game.verifyUnprovenFor(context.Background(), time.Second)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, readErr)
		require.ErrorContains(t, err, "remained unavailable after the verification boundary")
		require.ErrorContains(t, err, "last observation: status=")
		require.ErrorContains(t, err, readErr.Error())
	})
}

func TestVerifyUnprovenForReportsParentCancellationAndLastObservation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	game := newTestZKGame(t, func(context.Context) (ZKClaimData, error) {
		cancel()
		return challengedClaim(common.Address{}), nil
	})

	err := game.verifyUnprovenFor(ctx, time.Hour)

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "parent context ended")
	require.ErrorContains(t, err, "status=")
}

func TestVerifyUnprovenForDoesNotMaskCancellationAtBoundary(t *testing.T) {
	devT := devtest.SerialT(t)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		reads := 0
		game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
			reads++
			if reads == 2 {
				cancel()
			}
			return challengedClaim(common.Address{}), nil
		})

		err := game.verifyUnprovenFor(ctx, time.Second)

		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "parent context ended")
		require.Equal(t, 2, reads)
	})
}

func TestWaitForProvenBy(t *testing.T) {
	prover := common.HexToAddress("0xbeef")

	t.Run("retries transient reads and waits for prover", func(t *testing.T) {
		devT := devtest.SerialT(t)
		synctest.Test(t, func(t *testing.T) {
			readErr := errors.New("transient RPC failure")
			game := newTestZKGameWithT(devT, sequenceClaimReader(
				claimReadResult{err: readErr},
				claimReadResult{claim: challengedClaim(common.Address{})},
				claimReadResult{claim: challengedClaim(prover)},
			))
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			err := game.waitForProvenBy(ctx, prover)

			require.NoError(t, err)
		})
	})

	t.Run("fails immediately for wrong persistent prover", func(t *testing.T) {
		wrongProver := common.HexToAddress("0xdead")
		reads := 0
		game := newTestZKGame(t, func(context.Context) (ZKClaimData, error) {
			reads++
			return challengedClaim(wrongProver), nil
		})

		err := game.waitForProvenBy(context.Background(), prover)

		require.ErrorContains(t, err, wrongProver.Hex())
		require.ErrorContains(t, err, prover.Hex())
		require.Equal(t, 1, reads)
	})

	t.Run("does not mask context cancellation with expected prover", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		game := newTestZKGame(t, func(context.Context) (ZKClaimData, error) {
			cancel()
			return challengedClaim(prover), nil
		})

		err := game.waitForProvenBy(ctx, prover)

		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "context ended while waiting for expected prover")
	})

	t.Run("reports persistent reads when context ends", func(t *testing.T) {
		devT := devtest.SerialT(t)
		synctest.Test(t, func(t *testing.T) {
			readErr := errors.New("persistent RPC failure")
			game := newTestZKGameWithT(devT, func(context.Context) (ZKClaimData, error) {
				return ZKClaimData{}, readErr
			})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := game.waitForProvenBy(ctx, prover)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.ErrorContains(t, err, "last observation: none")
			require.ErrorContains(t, err, readErr.Error())
		})
	})

	t.Run("rejects zero expected prover", func(t *testing.T) {
		game := newTestZKGame(t, sequenceClaimReader(claimReadResult{claim: challengedClaim(common.Address{})}))

		err := game.waitForProvenBy(context.Background(), common.Address{})

		require.ErrorContains(t, err, "must not be the zero address")
	})
}
