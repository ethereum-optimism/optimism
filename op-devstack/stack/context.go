package stack

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

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
