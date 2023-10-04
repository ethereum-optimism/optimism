package rollup

import (
	"github.com/Layr-Labs/eigenda/api/grpc/retriever"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DAConfig struct {
	Rpc    string
	Client retriever.RetrieverClient
	// AuthToken string
}

func NewDAConfig(rpc string) (*DAConfig, error) {
	conn, err := grpc.Dial(rpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return &DAConfig{}, err
	}
	defer func() { _ = conn.Close() }()
	client := retriever.NewRetrieverClient(conn)

	return &DAConfig{
		Rpc:    rpc,
		Client: client,
	}, nil
}
