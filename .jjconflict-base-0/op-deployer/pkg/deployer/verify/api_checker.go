package verify

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type VerificationStatus struct {
	IsVerified          bool
	IsFullyVerified     bool
	IsPartiallyVerified bool
}

type APIChecker interface {
	CanCheck() bool
	CheckStatus(ctx context.Context, address common.Address) (*VerificationStatus, error)
	GetDefaultURL(chainID uint64) (string, error)
	GetChainArg(chainID uint64) (string, error)
}
