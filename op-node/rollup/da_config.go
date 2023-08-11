package rollup

import (
	"context"
	"encoding/hex"

	openrpc "github.com/rollkit/celestia-openrpc"
	"github.com/rollkit/celestia-openrpc/types/share"
)

type DAConfig struct {
	Rpc       string
	Namespace share.Namespace
	Client    *openrpc.Client
	AuthToken string
}

func NewDAConfig(rpc, token, ns string) (*DAConfig, error) {
	nsBytes, err := hex.DecodeString(ns)
	if err != nil {
		return &DAConfig{}, err
	}

	namespace, err := share.NewBlobNamespaceV0(nsBytes)
	if err != nil {
		return nil, err
	}

	client, err := openrpc.NewClient(context.Background(), rpc, token)
	if err != nil {
		return &DAConfig{}, err
	}

	return &DAConfig{
		Namespace: namespace,
		Rpc:       rpc,
		Client:    client,
	}, nil
}
