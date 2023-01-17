package rollup

import (
	"encoding/hex"
	"github.com/celestiaorg/go-cnc"
	"time"
)

type DAConfig struct {
	Rpc         string
	NamespaceId [8]byte
	Client      *cnc.Client
}

func NewDAConfig(rpc string, namespaceId string) (*DAConfig, error) {
	var nid [8]byte
	n, err := hex.DecodeString(namespaceId)
	if err != nil {
		return &DAConfig{}, err
	}
	copy(nid[:], n)
	daClient, err := cnc.NewClient(rpc, cnc.WithTimeout(30*time.Second))
	if err != nil {
		return &DAConfig{}, err
	}

	return &DAConfig{
		NamespaceId: nid,
		Rpc:         rpc,
		Client:      daClient,
	}, nil
}
