package sdmreplay

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// BlockNumberSource resolves the current head block number.
type BlockNumberSource interface {
	GetBlockNumber(ctx context.Context) (uint64, error)
}

// ResolveBlockNum parses a block selector string which can be:
// - "latest" -> fetches current block number
// - "latest-N" -> fetches current block number minus N
// - "0x..." -> hex block number
// - decimal string
func ResolveBlockNum(ctx context.Context, src BlockNumberSource, selector string) (uint64, error) {
	selector = strings.TrimSpace(selector)

	if selector == "latest" {
		return src.GetBlockNumber(ctx)
	}
	if strings.HasPrefix(selector, "latest-") {
		offset, err := strconv.ParseUint(strings.TrimPrefix(selector, "latest-"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid latest-N offset: %w", err)
		}
		latest, err := src.GetBlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		if offset > latest {
			return 0, fmt.Errorf("offset %d exceeds latest block %d", offset, latest)
		}
		return latest - offset, nil
	}
	return ParseBlockNum(selector)
}

// ParseBlockNum parses a hex or decimal block number.
func ParseBlockNum(s string) (uint64, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

// ParseHexUint64 parses a 0x-prefixed hex string into a uint64.
func ParseHexUint64(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return strconv.ParseUint(s, 16, 64)
}
