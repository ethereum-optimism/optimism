package stack

import (
	"context"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

// ContextWithKind annotates the context with the given kind of service
func ContextWithKind(ctx context.Context, kind Kind) context.Context {
	return logfilter.AddLogAttrToContext(ctx, "kind", kind)
}

// KindFromContext extracts the kind from the context.
func KindFromContext(ctx context.Context) Kind {
	v, _ := logfilter.ValueFromContext[Kind](ctx, "kind")
	return v
}

// KindSelector creates a log-filter that applies the given inner log-filter only if it matches the given kind.
// For logs of the specified kind, it applies the inner filter.
// For logs of other kinds, it returns false (filters them out).
func KindSelector(kind Kind) logfilter.Selector {
	return logfilter.Select("kind", kind)
}

// ContextWithChainID annotates the context with the given chainID of service
func ContextWithChainID(ctx context.Context, chainID eth.ChainID) context.Context {
	return logfilter.AddLogAttrToContext(ctx, "chainID", chainID)
}

// ChainIDFromContext extracts the chain ID from the context.
func ChainIDFromContext(ctx context.Context) eth.ChainID {
	v, _ := logfilter.ValueFromContext[eth.ChainID](ctx, "chainID")
	return v
}

// ChainIDSelector creates a log-filter that applies the given inner log-filter only if it matches the given chainID.
// This can be composed with logfilter package utils like logfilter.MuteAll or logfilter.Level
// to adjust logging for a specific chain ID.
func ChainIDSelector(chainID eth.ChainID) logfilter.Selector {
	return logfilter.Select("chainID", chainID)
}

// ContextWithID attaches a component ID to the context.
// This also automatically attaches the chain ID and component kind to the context, if available from the ID.
func ContextWithID(ctx context.Context, id slog.LogValuer) context.Context {
	if idWithChainID, ok := id.(ChainIDProvider); ok {
		ctx = ContextWithChainID(ctx, idWithChainID.ChainID())
	}
	if idWithKind, ok := id.(KindProvider); ok {
		ctx = ContextWithKind(ctx, idWithKind.Kind())
	}
	ctx = logfilter.AddLogAttrToContext(ctx, "id", id)
	return ctx
}

func IDFromContext[T slog.LogValuer](ctx context.Context) T {
	v, _ := logfilter.ValueFromContext[T](ctx, "id")
	return v
}

// IDSelector creates a log-filter that applies the given inner log-filter only if it matches the given ID.
// This can be composed with logfilter package utils like logfilter.MuteAll or logfilter.Level
// to adjust logging for a specific chain ID.
func IDSelector(id slog.LogValuer) logfilter.Selector {
	return logfilter.Select("id", id)
}
