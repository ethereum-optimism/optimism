package rpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type BatcherDriver interface {
	StartBatchSubmitting() error
	StopBatchSubmitting(ctx context.Context) error
	ChangeDataAvailability(daType flags.DataAvailabilityType, frameCount *uint)
	SanityCheckConfigUpdate(daType flags.DataAvailabilityType, frameCount *uint) error
}

type adminAPI struct {
	*rpc.CommonAdminAPI
	b BatcherDriver
}

func NewAdminAPI(dr BatcherDriver, m metrics.RPCMetricer, log log.Logger) *adminAPI {
	return &adminAPI{
		CommonAdminAPI: rpc.NewCommonAdminAPI(m, log),
		b:              dr,
	}
}

func GetAdminAPI(api *adminAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "admin",
		Service:   api,
	}
}

func (a *adminAPI) UseDAType(
	ctx context.Context,
	kind flags.DataAvailabilityType,
	frameCount *uint,
) error {
	if !flags.ValidDataAvailabilityType(kind) {
		return fmt.Errorf("unknown data-availability type: %q", kind)
	}
	if kind == flags.BlobsType && frameCount == nil {
		return fmt.Errorf("cant use blobs and have nil framecount second argument")
	}
	if kind == flags.CalldataType && frameCount != nil {
		return fmt.Errorf("cant use calldata with framecount second argument")
	}
	if kind == flags.BlobsType && frameCount != nil && (*frameCount == 0 || *frameCount > 6) {
		return fmt.Errorf("cant use blobs with frame count outside range [1,6] given: %d", *frameCount)
	}

	if err := a.b.SanityCheckConfigUpdate(kind, frameCount); err != nil {
		return err
	}

	if err := a.b.StopBatchSubmitting(ctx); err != nil {
		return err
	}
	// cant fail, we already sanity checked it
	a.b.ChangeDataAvailability(kind, frameCount)
	return a.b.StartBatchSubmitting()
}

func (a *adminAPI) StartBatcher(_ context.Context) error {
	return a.b.StartBatchSubmitting()
}

func (a *adminAPI) StopBatcher(ctx context.Context) error {
	return a.b.StopBatchSubmitting(ctx)
}
