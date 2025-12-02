package contracts

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	methodVersion = "version"
)

type version[C any] struct {
	prefixes string
	factory  func() (C, error)
}

type VersionedBuilder[C any] struct {
	versions []version[C]
}

func (v *VersionedBuilder[C]) AddVersion(major int, minor int, factory func() (C, error)) {
	v.versions = append(v.versions, version[C]{fmt.Sprintf("%d.%d.", major, minor), factory})
}

func (v *VersionedBuilder[C]) Build(ctx context.Context, caller *batching.MultiCaller, contractAbi *abi.ABI, addr common.Address, defaultVersion func() (C, error)) (C, error) {
	var nilC C
	result, err := caller.SingleCall(ctx, rpcblock.Latest, batching.NewContractCall(contractAbi, addr, methodVersion))
	if err != nil {
		return nilC, fmt.Errorf("failed to retrieve version of dispute game %v: %w", addr, err)
	}
	contractVersion := result.GetString(0)
	for _, version := range v.versions {
		if strings.HasPrefix(contractVersion, version.prefixes) {
			return version.factory()
		}
	}
	return defaultVersion()
}
