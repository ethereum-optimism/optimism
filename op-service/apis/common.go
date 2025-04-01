package apis

import "context"

type Version interface {
	Version(ctx context.Context) (string, error)
}
