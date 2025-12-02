package apis

import "context"

type BatcherActivity interface {
	StartBatcher(ctx context.Context) error
	StopBatcher(ctx context.Context) error
	FlushBatcher(ctx context.Context) error
}

type BatcherAdminServer interface {
	CommonAdminServer
	BatcherActivity
}

type BatcherAdminClient interface {
	CommonAdminClient
	BatcherActivity
}
