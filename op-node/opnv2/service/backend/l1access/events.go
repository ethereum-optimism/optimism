package l1access

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type LatestL1RequestEvent struct {
}

func (ev LatestL1RequestEvent) String() string {
	return "latest-l1-request"
}

type LatestL1UpdateEvent struct {
	LatestL1 eth.BlockRef
}

func (ev LatestL1UpdateEvent) String() string {
	return "latest-l1-update"
}

type ConfirmedL1RequestEvent struct {
}

func (ev ConfirmedL1RequestEvent) String() string {
	return "confirmed-l1-request"
}

type ConfirmedL1UpdateEvent struct {
	// ConfirmedL1 block. Empty if we failed to retrieve it.
	ConfirmedL1 eth.BlockRef
}

func (ev ConfirmedL1UpdateEvent) String() string {
	return "confirmed-l1-update"
}

type FinalizedL1RequestEvent struct {
}

func (ev FinalizedL1RequestEvent) String() string {
	return "finalized-l1-request"
}

type FinalizedL1UpdateEvent struct {
	FinalizedL1 eth.BlockRef
}

func (ev FinalizedL1UpdateEvent) String() string {
	return "finalized-l1-update"
}

type TemporaryL1AccessErrorEvent struct {
	LatestL1    eth.BlockRef
	ConfirmedL1 eth.BlockRef
	FinalizedL1 eth.BlockRef
	Err         error
}

func (ev TemporaryL1AccessErrorEvent) String() string {
	return "temporary-l1-access-error"
}

type ByNumberL1RequestEvent struct {
	Num uint64
}

func (ev ByNumberL1RequestEvent) String() string {
	return "by-number-l1-request"
}

type RetrievedL1BlockEvent struct {
	Ref eth.BlockRef
}

func (ev RetrievedL1BlockEvent) String() string {
	return "retrieved-l1-block"
}
