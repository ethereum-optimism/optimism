package writes

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type RPC interface {
	CallContext(context.Context, any, string, ...any) error
}

// Source reads complete block outcomes from an operator's private execution RPC.
// Failure is never interpreted as an empty write set.
type Source struct{ RPC RPC }

func (s Source) FetchWrites(ctx context.Context, hash common.Hash) ([]Record, error) {
	var result *struct {
		BlockHash common.Hash `json:"blockHash"`
		Writes    *[]Record   `json:"writes"`
	}
	if err := s.RPC.CallContext(ctx, &result, "debug_privateBlockWrites", hash); err != nil {
		return nil, err
	}
	if result == nil || result.Writes == nil || result.BlockHash != hash {
		return nil, fmt.Errorf("%w: incomplete or wrong-block RPC response", ErrInvalid)
	}
	if _, err := Encode(*result.Writes); err != nil {
		return nil, err
	}
	return *result.Writes, nil
}
