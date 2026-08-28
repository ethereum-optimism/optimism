package sequencing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const catchupReleaseSchemaVersion = 1

const catchupReleaseRetryInterval = 100 * time.Millisecond

type CatchupRelease struct {
	SchemaVersion       uint64      `json:"schema_version"`
	ID                  string      `json:"id"`
	CheckpointID        string      `json:"checkpoint_id"`
	DeploymentID        string      `json:"deployment_id"`
	ResetGeneration     string      `json:"reset_generation"`
	CheckpointNumber    uint64      `json:"checkpoint_number"`
	CheckpointHash      common.Hash `json:"checkpoint_hash"`
	CheckpointTimestamp uint64      `json:"checkpoint_timestamp"`
	OriginNumber        uint64      `json:"origin_number"`
	OriginTimestamp     uint64      `json:"origin_timestamp"`
	L1BlockTime         uint64      `json:"l1_block_time"`
	L2BlockTime         uint64      `json:"l2_block_time"`
	TargetLead          uint64      `json:"target_lead"`
	HardCap             uint64      `json:"hard_cap"`
	CatchupMultiplier   uint64      `json:"catchup_multiplier"`
	ReleaseUnixNano     uint64      `json:"release_unix_nano"`
}

type CatchupSchedule struct {
	release CatchupRelease
}

func FetchCatchupSchedule(ctx context.Context, url string) (*CatchupSchedule, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create catch-up release request: %w", err)
		}
		response, err := client.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("fetch catch-up release: %w", ctx.Err())
			}
			if err := waitCatchupReleaseRetry(ctx); err != nil {
				return nil, err
			}
			continue
		}
		if response.StatusCode == http.StatusOK {
			data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
			response.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("read catch-up release: %w", err)
			}
			if len(data) > 64<<10 {
				return nil, fmt.Errorf("catch-up release response exceeds 64 KiB limit")
			}
			var release CatchupRelease
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&release); err != nil {
				return nil, fmt.Errorf("decode catch-up release: %w", err)
			}
			if err := decoder.Decode(&struct{}{}); err != io.EOF {
				return nil, fmt.Errorf("decode catch-up release: trailing JSON content")
			}
			return NewCatchupSchedule(release)
		}
		status := response.Status
		response.Body.Close()
		if !isCatchupReleasePending(response.StatusCode) {
			return nil, fmt.Errorf("fetch catch-up release: HTTP %s", status)
		}
		if err := waitCatchupReleaseRetry(ctx); err != nil {
			return nil, err
		}
	}
}

func isCatchupReleasePending(status int) bool {
	switch status {
	case http.StatusTooEarly, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func waitCatchupReleaseRetry(ctx context.Context) error {
	timer := time.NewTimer(catchupReleaseRetryInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("fetch catch-up release: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func NewCatchupSchedule(release CatchupRelease) (*CatchupSchedule, error) {
	if release.SchemaVersion != catchupReleaseSchemaVersion {
		return nil, fmt.Errorf("catch-up release schema_version must be %d", catchupReleaseSchemaVersion)
	}
	if release.CheckpointID == "" || release.DeploymentID == "" || release.ResetGeneration == "" || release.CheckpointNumber == 0 || release.CheckpointHash == (common.Hash{}) || release.CheckpointTimestamp == 0 || release.OriginNumber == 0 || release.OriginTimestamp == 0 {
		return nil, fmt.Errorf("complete catch-up checkpoint identity is required")
	}
	maxDurationSeconds := uint64(math.MaxInt64 / int64(time.Second))
	if release.L1BlockTime > maxDurationSeconds || release.L2BlockTime > maxDurationSeconds {
		return nil, fmt.Errorf("catch-up block time exceeds supported duration")
	}
	if release.L1BlockTime == 0 || release.L2BlockTime == 0 || release.TargetLead == 0 || release.HardCap <= release.TargetLead || release.ReleaseUnixNano > math.MaxInt64 {
		return nil, fmt.Errorf("valid catch-up timing is required")
	}
	switch release.CatchupMultiplier {
	case 1, 2, 5, 10:
	default:
		return nil, fmt.Errorf("catch-up multiplier must be one of 1, 2, 5, or 10")
	}
	clone := release
	if err := clone.Seal(); err != nil {
		return nil, err
	}
	if release.ID != clone.ID {
		return nil, fmt.Errorf("catch-up release identity mismatch: got %s, expected %s", release.ID, clone.ID)
	}
	return &CatchupSchedule{release: release}, nil
}

func (r *CatchupRelease) Seal() error {
	r.SchemaVersion = catchupReleaseSchemaVersion
	clone := *r
	clone.ID = ""
	data, err := json.Marshal(clone)
	if err != nil {
		return fmt.Errorf("encode catch-up release identity: %w", err)
	}
	sum := sha256.Sum256(data)
	r.ID = "sha256:" + hex.EncodeToString(sum[:])
	return nil
}

func (s *CatchupSchedule) ValidateCheckpoint(number uint64, hash common.Hash, timestamp, blockTime uint64) error {
	if number != s.release.CheckpointNumber || hash != s.release.CheckpointHash || timestamp != s.release.CheckpointTimestamp || blockTime != s.release.L2BlockTime {
		return fmt.Errorf("catch-up release does not match rollup checkpoint")
	}
	return nil
}

func (s *CatchupSchedule) PayloadDueTime(payloadTimestamp uint64) (time.Time, error) {
	if payloadTimestamp < s.release.CheckpointTimestamp || payloadTimestamp > math.MaxInt64 {
		return time.Time{}, fmt.Errorf("payload timestamp is outside the catch-up schedule")
	}
	delta := payloadTimestamp - s.release.CheckpointTimestamp
	wholeSeconds := delta / s.release.CatchupMultiplier
	remainder := delta % s.release.CatchupMultiplier
	if wholeSeconds > uint64(math.MaxInt64/int64(time.Second)) {
		return time.Time{}, fmt.Errorf("accelerated payload deadline overflows time.Duration")
	}
	scaled := time.Duration(wholeSeconds) * time.Second
	fraction := time.Duration(remainder) * time.Second / time.Duration(s.release.CatchupMultiplier)
	if fraction > time.Duration(math.MaxInt64)-scaled {
		return time.Time{}, fmt.Errorf("accelerated payload deadline overflows time.Duration")
	}
	accelerated := time.Unix(0, int64(s.release.ReleaseUnixNano)).Add(scaled + fraction)
	canonical := time.Unix(int64(payloadTimestamp), 0)
	if canonical.After(accelerated) {
		return canonical, nil
	}
	return accelerated, nil
}

// ShouldParkForOrigin reserves the configured L1 runway before sequencer drift is exhausted.
func (s *CatchupSchedule) ShouldParkForOrigin(payloadTimestamp, originTimestamp, maxDrift uint64) bool {
	if payloadTimestamp <= originTimestamp {
		return false
	}
	if maxDrift <= s.release.TargetLead {
		return true
	}
	return payloadTimestamp-originTimestamp >= maxDrift-s.release.TargetLead
}

// OriginRetryDelay polls four times per accelerated L1 slot while sequencing is parked.
func (s *CatchupSchedule) OriginRetryDelay() time.Duration {
	scaledL1Slot := time.Duration(s.release.L1BlockTime) * time.Second / time.Duration(s.release.CatchupMultiplier)
	retry := scaledL1Slot / 4
	if retry < catchupReleaseRetryInterval {
		return catchupReleaseRetryInterval
	}
	return retry
}

func (s *CatchupSchedule) BuildDeadline(parent eth.L2BlockRef, _ time.Duration) (time.Time, uint64, error) {
	payloadTimestamp, err := s.nextPayloadTimestamp(parent)
	if err != nil {
		return time.Time{}, 0, err
	}
	due, err := s.PayloadDueTime(payloadTimestamp)
	if err != nil {
		return time.Time{}, 0, err
	}
	scaledBlockTime := time.Duration(s.release.L2BlockTime) * time.Second / time.Duration(s.release.CatchupMultiplier)
	return due.Add(-scaledBlockTime), payloadTimestamp, nil
}

func (s *CatchupSchedule) nextPayloadTimestamp(parent eth.L2BlockRef) (uint64, error) {
	if parent.Number < s.release.CheckpointNumber || parent.Time < s.release.CheckpointTimestamp {
		return 0, fmt.Errorf("sequencer parent precedes catch-up checkpoint")
	}
	if parent.Number == math.MaxUint64 || parent.Time > math.MaxUint64-s.release.L2BlockTime {
		return 0, fmt.Errorf("sequencer next payload overflows uint64")
	}
	payloadTimestamp := parent.Time + s.release.L2BlockTime
	expectedNumber := parent.Number + 1
	blockDelta := expectedNumber - s.release.CheckpointNumber
	if blockDelta > (math.MaxUint64-s.release.CheckpointTimestamp)/s.release.L2BlockTime {
		return 0, fmt.Errorf("checkpoint-relative payload timestamp overflows uint64")
	}
	expectedTimestamp := s.release.CheckpointTimestamp + blockDelta*s.release.L2BlockTime
	if payloadTimestamp != expectedTimestamp {
		return 0, fmt.Errorf("sequencer parent is off the checkpoint-relative L2 timestamp formula")
	}
	return payloadTimestamp, nil
}

func (s *CatchupSchedule) SealDeadline(parent eth.L2BlockRef, sealingDuration time.Duration) (time.Time, error) {
	payloadTimestamp, err := s.nextPayloadTimestamp(parent)
	if err != nil {
		return time.Time{}, err
	}
	due, err := s.PayloadDueTime(payloadTimestamp)
	if err != nil {
		return time.Time{}, err
	}
	return due.Add(-sealingDuration), nil
}
