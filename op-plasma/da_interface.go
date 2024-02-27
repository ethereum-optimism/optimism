package plasma

import "context"

type DataClient interface {
	GetInput(ctx context.Context, key []byte) ([]byte, error)
	SetInput(ctx context.Context, img []byte) ([]byte, error)
}
