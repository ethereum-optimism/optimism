package rollup

import (
	"encoding/hex"
	"time"

	"github.com/celestiaorg/go-cnc"
)

type DAConfig struct {
	Rpc       string
	Namespace cnc.Namespace
	Client    *cnc.Client
}

func NewDAConfig(rpc string, ns string) (*DAConfig, error) {
	nsBytes, err := hex.DecodeString(ns)
	if err != nil {
		return &DAConfig{}, err
	}

	namespace := cnc.MustNewV0(nsBytes)

	daClient, err := cnc.NewClient(rpc, cnc.WithTimeout(30*time.Second))
	if err != nil {
		return &DAConfig{}, err
	}

	return &DAConfig{
		Namespace: namespace,
		Rpc:       rpc,
		Client:    daClient,
	}, nil
}
