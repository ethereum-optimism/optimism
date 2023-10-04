package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/Layr-Labs/eigenda/api/grpc/retriever"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 4 {
		panic("usage: op-eigenda <eth calldata>")
	}
	data, _ := hex.DecodeString(os.Args[2])
	blobInfo := &disperser.BlobInfo{}
	err := proto.Unmarshal(data, blobInfo)
	if err != nil {
		panic(err)
	}

	// TODO: What should we log here?
	// fmt.Printf("eigenda block height: %v; tx index: %v\n", height, index)
	// fmt.Println("-----------------------------------------")

	// TODO: Set the target to the default retreiver RPC endpoint for local development
	conn, err := grpc.Dial("http://localhost:26658", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()
	client := retriever.NewRetrieverClient(conn)

	// TODO: Confirm the correctness of this lookup
	quorumID := blobInfo.BlobHeader.BlobQuorumParams[0].QuorumNumber
	confirmationBlockNumber := blobInfo.BlobVerificationProof.BatchMetadata.ConfirmationBlockNumber
	blobReply, err := client.RetrieveBlob(context.Background(), &retriever.BlobRequest{
		BatchHeaderHash:      blobInfo.BlobVerificationProof.BatchMetadata.BatchHeaderHash,
		BlobIndex:            blobInfo.BlobVerificationProof.BlobIndex,
		ReferenceBlockNumber: confirmationBlockNumber,
		QuorumId:             quorumID,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("optimism block data on eigenda: %x\n", blobReply.Data)
}
