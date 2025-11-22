// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package metrics

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

// NoopMetricsImpl implements the Metricer interface with no-op methods.
type NoopMetricsImpl struct {
	txmetrics.NoopTxMetrics
	contractMetrics.NoopMetrics
	NoopVmMetrics
}

var _ Metricer = (*NoopMetricsImpl)(nil)

// StartBalanceMetrics returns nil, providing no real resource closer.
func (*NoopMetricsImpl) StartBalanceMetrics(_ log.Logger, _ *ethclient.Client, _ common.Address) io.Closer {
	return nil
}

// NoopMetrics is a pre-initialized instance of NoopMetricsImpl.
var NoopMetrics Metricer = new(NoopMetricsImpl)

// --- General Status Metrics ---

func (*NoopMetricsImpl) RecordInfo(_ string) {}
func (*NoopMetricsImpl) RecordUp()           {}

// --- Game Action Metrics ---

func (*NoopMetricsImpl) RecordGameMove()          {}
func (*NoopMetricsImpl) RecordGameStep()          {}
func (*NoopMetricsImpl) RecordGameL2Challenge()   {}
func (*NoopMetricsImpl) RecordActedL1Block(_ uint64) {}

// --- Preimage Metrics ---

func (*NoopMetricsImpl) RecordPreimageChallenged()    {}
func (*NoopMetricsImpl) RecordPreimageChallengeFailed() {}
func (*NoopMetricsImpl) RecordLargePreimageCount(_ int) {}

// --- Bond Metrics ---

func (*NoopMetricsImpl) RecordBondClaimFailed()  {}
func (*NoopMetricsImpl) RecordBondClaimed(_ uint64) {}

// --- Time and Duration Metrics ---

func (*NoopMetricsImpl) RecordClaimResolutionTime(_ float64) {}
func (*NoopMetricsImpl) RecordGameActTime(_ float64)         {}

// --- Game Status and Monitoring Metrics ---

func (*NoopMetricsImpl) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {}

func (*NoopMetricsImpl) RecordGameUpdateScheduled() {}
func (*NoopMetricsImpl) RecordGameUpdateCompleted() {}

// --- Executor Pool Metrics ---

func (*NoopMetricsImpl) IncActiveExecutors() {}
func (*NoopMetricsImpl) DecActiveExecutors() {}
func (*NoopMetricsImpl) IncIdleExecutors()   {}
func (*NoopMetricsImpl) DecIdleExecutors()   {}

// --- Cache Metrics ---

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)       {}

// ToTypedVmMetrics returns a new NoopMetrics instance wrapped with VM type context.
func (m *NoopMetricsImpl) ToTypedVmMetrics(vmType string) TypedVmMetricer {
	return NewTypedVmMetrics(m, vmType)
}
