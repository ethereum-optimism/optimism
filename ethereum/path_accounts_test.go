package ethereum

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/util"
)

func createAccount(t *testing.T, path string, b logical.Backend, storage logical.Storage) map[string]interface{} {
	accountsData := map[string]interface{}{}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      path,
		Storage:   storage,
		Data:      accountsData,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
	t.Log(util.PrettyPrint(resp.Data))
	return resp.Data
}

func TestAccount_Read(t *testing.T) {
	b, storage := getBackendConfigured(t)

	// we must create a wallet
	walletPath := "wallets/fixed-token-test"

	createWallet(t, walletPath, b, storage)
	account := createAccount(t, walletPath+"/accounts", b, storage)
	// create an address
	// harvest address
	address := account["address"].(string)
	// request token balance for address
	accountPath := walletPath + "/accounts/" + address
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      accountPath,
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
	if !reflect.DeepEqual(resp.Data, account) {
		t.Fatalf("Expected did not equal actual: expected %#v\n got %#v\n", account, resp.Data)
	}
}
