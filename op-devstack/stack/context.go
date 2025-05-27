package stack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
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
	ctx = log.RegisterLogAttrOnContext(ctx, "kind", kindCtxKey)
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
	ctx = log.RegisterLogAttrOnContext(ctx, "chainID", chainIDCtxKey)
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

type idCtxKeyType struct{}

var idCtxKey = idCtxKeyType{}

// IDLogFilter creates a log-filter that applies the given inner log-filter only if it matches the given ID.
// This can be composed with logfilter package utils like logfilter.Mute or logfilter.Add
// to adjust logging for a specific chain ID.
func IDLogFilter(id any, filter logfilter.LogFilter) logfilter.LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		v := ctx.Value(idCtxKey)
		if v == nil {
			return lvl
		}
		switch v := v.(type) {
		case IDWithChain:
			if id, ok := id.(IDWithChain); ok {
				if v.ChainID() == id.ChainID() && v.Kind() == id.Kind() && v.Key() == id.Key() {
					return filter(ctx, lvl)
				}
			}
		case IDOnlyChainID:
			if id, ok := id.(IDOnlyChainID); ok {
				if v.ChainID() == id.ChainID() && v.Kind() == id.Kind() {
					return filter(ctx, lvl)
				}
			}
		case GenericID:
			if id, ok := id.(GenericID); ok {
				if v.Kind() == id.Kind() {
					return filter(ctx, lvl)
				}
			}
		default:
			panic(fmt.Sprintf("unsupported ID type: %T", v))
		}
		return lvl
	}
}

func ContextWithID(ctx context.Context, id any) context.Context {
	if idWithChainID, ok := id.(interface{ ChainID() eth.ChainID }); ok {
		ctx = ContextWithChainID(ctx, idWithChainID.ChainID())
	}
	if idWithKind, ok := id.(interface{ Kind() Kind }); ok {
		ctx = ContextWithKind(ctx, idWithKind.Kind())
	}
	ctx = context.WithValue(ctx, idCtxKey, id)
	ctx = log.RegisterLogAttrOnContext(ctx, "id", idCtxKey)
	return ctx
}
