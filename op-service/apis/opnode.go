package apis

import "context"

type OpnodeAdminServer interface {
	RollupAdminServer
	ResetDerivationPipeline(ctx context.Context) error
}
