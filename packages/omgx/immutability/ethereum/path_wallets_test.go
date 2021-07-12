// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/util"
)

func createWallet(t *testing.T, path string, b logical.Backend, storage logical.Storage) map[string]interface{} {
	walletData := map[string]interface{}{
		"mnemonic": "radar limb wish goose acquire toddler produce dynamic wear raccoon example basket",
	}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      path,
		Storage:   storage,
		Data:      walletData,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	t.Log(util.PrettyPrint(resp.Data))
	return resp.Data
}

func TestWallet_Read(t *testing.T) {
	b, storage := getBackendConfigured(t)
	walletPath := "wallets/test-rinkeby"
	// create a wallet
	created := createWallet(t, walletPath, b, storage)

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      walletPath,
		Storage:   storage,
		Data:      nil,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	t.Log(util.PrettyPrint(resp.Data))
	if !reflect.DeepEqual(resp.Data, created) {
		t.Fatalf("Expected did not equal actual: expected %#v\n got %#v\n", created, resp.Data)
	}
}
