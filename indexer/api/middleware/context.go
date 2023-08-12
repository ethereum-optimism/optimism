package middleware

import (
	"context"
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/log"
)

type contextKey string

const (
	loggerKey              contextKey = "logger"
	bridgeTransfersViewKey contextKey = "bridgeTransfersView"
)

// Setters
func setLogger(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func setBridgeTransfersView(ctx context.Context, bv database.BridgeTransfersView) context.Context {
	return context.WithValue(ctx, bridgeTransfersViewKey, bv)
}

// Getters
func GetLogger(ctx context.Context) log.Logger {
	logger, ok := ctx.Value(loggerKey).(log.Logger)
	if !ok {
		panic("Logger not found in context!")
	}
	return logger
}

func GetBridgeTransfersView(ctx context.Context) database.BridgeTransfersView {
	bv, ok := ctx.Value(bridgeTransfersViewKey).(database.BridgeTransfersView)
	if !ok {
		panic("BridgeTransfersView not found in context!")
	}
	return bv
}

func ContextMiddleware(logger log.Logger, bv database.BridgeTransfersView) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := setLogger(r.Context(), logger)
			ctx = setBridgeTransfersView(ctx, bv)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
