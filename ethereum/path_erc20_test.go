// Copyright (C) Immutability, LLC - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, August 2019

package ethereum

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omisego/immutability-eth-plugin/util"
)

func TestTokenBalance_Read(t *testing.T) {
	b, storage := getBackendConfigured(t)

	// we must create a wallet
	walletPath := "wallets/fixed-token-test"

	createWallet(t, walletPath, b, storage)
	account := createAccount(t, walletPath+"/accounts", b, storage)
	// create an address
	// harvest address
	address := account["address"].(string)
	// request token balance for address
	tokenBalancePath := walletPath + "/accounts/" + address + "/erc-20/balanceOf"

	tokenBalanceData := map[string]interface{}{
		"contract": "0xdFADF516B98B687d2F8BB02872dd34D46B722B3C",
	}

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      tokenBalancePath,
		Storage:   storage,
		Data:      tokenBalanceData,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
	t.Log(util.PrettyPrint(resp.Data))

}
