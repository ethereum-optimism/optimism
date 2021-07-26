// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"reflect"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/util"
)

const (
	configPath string = "config"
)

func getBackend(t *testing.T) (logical.Backend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24
	b, err := Backend(nil)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	err = b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	return b, config.StorageView
}

func getBackendConfigured(t *testing.T) (logical.Backend, logical.Storage) {
	b, storage := getBackend(t)
	// to test successful token read, we must be configured
	configData := map[string]interface{}{
		"chain_id": "4",
		"rpc_url":  "https://rinkeby.infura.io",
	}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      configPath,
		Storage:   storage,
		Data:      configData,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
	return b, storage
}

func TestConfig_Read(t *testing.T) {
	b, storage := getBackend(t)

	data := map[string]interface{}{
		"chain_id":        "4",
		"rpc_url":         "https://rinkeby.infura.io",
		"blacklist":       []string{},
		"whitelist":       []string{},
		"bound_cidr_list": []string{"192.168.1.0/16"},
	}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      configPath,
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      configPath,
		Storage:   storage,
		Data:      nil,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	t.Log(util.PrettyPrint(resp.Data))
	if !reflect.DeepEqual(resp.Data, data) {
		t.Fatalf("Expected did not equal actual: expected %#v\n got %#v\n", data, resp.Data)
	}
}
