package stack

import (
	"context"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logfilter"
)

type kindCtxKeyType struct{}

var kindCtxKey = kindCtxKeyType{}

const UnknownKind Kind = ""

// KindFromContext reads what the kind of service the context is focused on. This may be UnknownKind if unspecified.
func KindFromContext(ctx context.Context) Kind {
	v := ctx.Value(kindCtxKey)
	if v == nil {
		return UnknownKind
	}
	return v.(Kind)
}

// ContextWithKind annotates the context with the given kind of service
func ContextWithKind(ctx context.Context, kind Kind) context.Context {
	return context.WithValue(ctx, kindCtxKey, kind)
}

// KindLogFilter creates a log-filter that applies the given inner log-filter only if it matches the given kind.
// This can be composed with logfilter package utils like logfilter.Mute or logfilter.Add
// to adjust logging for a specific service kind.
func KindLogFilter(kind Kind, filter logfilter.LogFilter) logfilter.LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		v := KindFromContext(ctx)
		if v == kind {
			return filter(ctx, lvl)
		}
		return lvl
	}
}

type chainIDCtxKeyType struct{}

var chainIDCtxKey = chainIDCtxKeyType{}

// ChainIDFromContext reads what the chainID of service the context is focused on. This may be UnknownChainID if unspecified.
func ChainIDFromContext(ctx context.Context) eth.ChainID {
	v := ctx.Value(chainIDCtxKey)
	if v == nil {
		return eth.ChainID{}
	}
	return v.(eth.ChainID)
}

// ContextWithChainID annotates the context with the given chainID of service
func ContextWithChainID(ctx context.Context, chainID eth.ChainID) context.Context {
	return context.WithValue(ctx, chainIDCtxKey, chainID)
}

// ChainIDLogFilter creates a log-filter that applies the given inner log-filter only if it matches the given chainID.
// This can be composed with logfilter package utils like logfilter.Mute or logfilter.Add
// to adjust logging for a specific chain ID.
func ChainIDLogFilter(chainID eth.ChainID, filter logfilter.LogFilter) logfilter.LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		v := ChainIDFromContext(ctx)
		if v == chainID {
			return filter(ctx, lvl)
		}
		return lvl
	}
}
