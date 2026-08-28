package sequencing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestFetchCatchupScheduleWaitsUntilReleaseArmed(t *testing.T) {
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(time.Unix(2_000, 0).UnixNano()),
	}
	require.NoError(t, release.Seal())

	var requests atomic.Uint64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) < 3 {
			http.Error(w, "release is not armed", http.StatusTooEarly)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(release))
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	schedule, err := FetchCatchupSchedule(ctx, server.URL)
	require.NoError(t, err)
	require.NotNil(t, schedule)
	require.Equal(t, uint64(3), requests.Load())
}

func TestFetchCatchupScheduleRejectsOversizedResponseTail(t *testing.T) {
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(time.Unix(2_000, 0).UnixNano()),
	}
	require.NoError(t, release.Seal())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(writer).Encode(release))
		_, err := writer.Write([]byte(strings.Repeat(" ", 64<<10)))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	_, err := FetchCatchupSchedule(t.Context(), server.URL)
	require.ErrorContains(t, err, "exceeds 64 KiB limit")
}

func TestCatchupReleaseGoldenWireContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "catchup-v1", "release.json"))
	require.NoError(t, err)
	var release CatchupRelease
	require.NoError(t, json.Unmarshal(data, &release))
	_, err = NewCatchupSchedule(release)
	require.NoError(t, err)
}

func TestCatchupDeadlinePreservesPayloadTimestamp(t *testing.T) {
	releaseTime := time.Unix(2_000_000_000, 0)
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(releaseTime.UnixNano()),
	}
	require.NoError(t, release.Seal())
	schedule, err := NewCatchupSchedule(release)
	require.NoError(t, err)

	parent := eth.L2BlockRef{Number: 500, Hash: common.HexToHash("0x500"), Time: 1_000}
	deadline, payloadTimestamp, err := schedule.BuildDeadline(parent, 50*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, uint64(1_002), payloadTimestamp)
	require.Equal(t, releaseTime, deadline)
}

func TestCatchupDueTimeTransitionsToCanonicalWallClock(t *testing.T) {
	releaseTime := time.Unix(2_000, 0)
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(releaseTime.UnixNano()),
	}
	require.NoError(t, release.Seal())
	schedule, err := NewCatchupSchedule(release)
	require.NoError(t, err)

	due, err := schedule.PayloadDueTime(2_000)
	require.NoError(t, err)
	require.Equal(t, releaseTime.Add(100*time.Second), due)
	due, err = schedule.PayloadDueTime(2_200)
	require.NoError(t, err)
	require.Equal(t, time.Unix(2_200, 0), due)
}

func TestCatchupScheduleRejectsPreCheckpointPayload(t *testing.T) {
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(time.Unix(2_000, 0).UnixNano()),
	}
	require.NoError(t, release.Seal())
	schedule, err := NewCatchupSchedule(release)
	require.NoError(t, err)

	_, _, err = schedule.BuildDeadline(eth.L2BlockRef{Number: 499, Time: 998}, 50*time.Millisecond)
	require.ErrorContains(t, err, "precedes catch-up checkpoint")
	_, err = schedule.SealDeadline(eth.L2BlockRef{Number: 499, Time: 998}, 50*time.Millisecond)
	require.ErrorContains(t, err, "precedes catch-up checkpoint")
}

func TestCatchupInitialBringupUsesCheckpointTimestampsNotHostTime(t *testing.T) {
	releaseTime := time.Unix(2_000_000_000, 0)
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(releaseTime.UnixNano()),
	}
	require.NoError(t, release.Seal())
	schedule, err := NewCatchupSchedule(release)
	require.NoError(t, err)

	parent := eth.L2BlockRef{Number: 1_399, Time: 2_798}
	deadline, payloadTimestamp, err := schedule.BuildDeadline(parent, 50*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, uint64(2_800), payloadTimestamp)
	require.Equal(t, releaseTime.Add(179*time.Second+800*time.Millisecond), deadline)
	require.NotEqual(t, uint64(deadline.Unix()), payloadTimestamp)
}

func TestCatchupScheduleReservesOriginHeadroom(t *testing.T) {
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: 12, L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(time.Unix(2_000, 0).UnixNano()),
	}
	require.NoError(t, release.Seal())
	schedule, err := NewCatchupSchedule(release)
	require.NoError(t, err)

	require.False(t, schedule.ShouldParkForOrigin(2_000, 261, 1_800))
	require.True(t, schedule.ShouldParkForOrigin(2_000, 260, 1_800))
	require.Equal(t, 300*time.Millisecond, schedule.OriginRetryDelay())
}

func TestCatchupScheduleRejectsBlockTimeDurationOverflow(t *testing.T) {
	release := CatchupRelease{
		CheckpointID: "sha256:checkpoint", DeploymentID: "shadow", ResetGeneration: "1",
		CheckpointNumber: 500, CheckpointHash: common.HexToHash("0x500"), CheckpointTimestamp: 1_000,
		OriginNumber: 100, OriginTimestamp: 900, L1BlockTime: ^uint64(0), L2BlockTime: 2,
		TargetLead: 60, HardCap: 1_200, CatchupMultiplier: 10,
		ReleaseUnixNano: uint64(time.Unix(2_000, 0).UnixNano()),
	}
	require.NoError(t, release.Seal())
	_, err := NewCatchupSchedule(release)
	require.ErrorContains(t, err, "block time exceeds supported duration")

	release.L1BlockTime = 12
	release.L2BlockTime = ^uint64(0)
	require.NoError(t, release.Seal())
	_, err = NewCatchupSchedule(release)
	require.ErrorContains(t, err, "block time exceeds supported duration")
}
