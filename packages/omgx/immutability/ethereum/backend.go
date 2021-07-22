// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// Symbol is the lowercase crypto token symbol
	Symbol string = "eth"
)

// Factory returns the backend
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := Backend(conf)
	if err != nil {
		return nil, err
	}
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

// FactoryType returns the factory
func FactoryType(backendType logical.BackendType) logical.Factory {
	return func(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
		b, err := Backend(conf)
		if err != nil {
			return nil, err
		}
		b.BackendType = backendType
		if err = b.Setup(ctx, conf); err != nil {
			return nil, err
		}
		return b, nil
	}
}

// Backend returns the backend
func Backend(conf *logical.BackendConfig) (*PluginBackend, error) {
	var b PluginBackend
	b.Backend = &framework.Backend{
		Help: "",
		Paths: framework.PathAppend(
			ConfigPaths(&b),
			WalletPaths(&b),
			PlasmaPaths(&b),
			AccountPaths(&b),
			OvmPaths(&b),
		),
		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"wallets/",
			},
		},
		Secrets:     []*framework.Secret{},
		BackendType: logical.TypeLogical,
	}
	return &b, nil
}

// PluginBackend implements the Backend for this plugin
type PluginBackend struct {
	*framework.Backend
}

// QualifiedPath prepends the token symbol to the path
func QualifiedPath(subpath string) string {
	return subpath
}

// ContractPath prepends the token symbol to the path
func ContractPath(contract, method string) string {
	return fmt.Sprintf("%s/%s/%s", QualifiedPath("wallets/"+framework.GenericNameRegex("name")+"/accounts/"+framework.GenericNameRegex("address")), contract, method)
}

// SealWrappedPaths returns the paths that are seal wrapped
func SealWrappedPaths(b *PluginBackend) []string {
	return []string{
		QualifiedPath("wallets/"),
	}
}
