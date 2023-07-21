package rollup

import (
	"context"
	"encoding/hex"

	openrpc "github.com/rollkit/celestia-openrpc"
	openrpcns "github.com/rollkit/celestia-openrpc/types/namespace"
	"github.com/rollkit/celestia-openrpc/types/share"
)

type DAConfig struct {
	Rpc       string
	Namespace openrpcns.Namespace
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
		Namespace: namespace.ToAppNamespace(),
		Rpc:       rpc,
		Client:    client,
	}, nil
}
