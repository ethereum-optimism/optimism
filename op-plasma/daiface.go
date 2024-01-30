package plasma

import "context"

type DAStorage interface {
	GetInput(ctx context.Context, key []byte) ([]byte, error)
	SetInput(ctx context.Context, img []byte) ([]byte, error)
}
