package apis

import (
	"context"
	"log/slog"
)

type LogLevelClient interface {
	SetLogLevel(ctx context.Context, lvl slog.Level) error
}

type LogLevelServer interface {
	SetLogLevel(ctx context.Context, lvl string) error
}

type CommonAdminClient interface {
	LogLevelClient
}

type CommonAdminServer interface {
	LogLevelServer
}
