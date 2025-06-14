package controller

import (
	"iter"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ClockState interface {
	Now() time.Time
}

type State interface {
	ClockState

	IterPipelines(predicates ...Predicate[*PipelineState]) iter.Seq[*PipelineState]

	IterDBs(predicates ...Predicate[*ChainDBState]) iter.Seq[*ChainDBState]
	ChainDB(chainID eth.ChainID) (state *ChainDBState, ok bool)

	IterRWELs(predicates ...Predicate[*RWELState]) iter.Seq[*RWELState]

	IterRELs(predicates ...Predicate[*RELState]) iter.Seq[*RELState]

	IterRunCfgs(predicates ...Predicate[*RunCfgState]) iter.Seq[*RunCfgState]

	Payloads(chainID eth.ChainID) (state *PayloadsState, ok bool)

	L1State() *L1AccessState
}
