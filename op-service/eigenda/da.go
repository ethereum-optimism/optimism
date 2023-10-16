package eigenda

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/Layr-Labs/eigenda/encoding/utils/codec"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type IEigenDA interface {
	RetrieveBlob(ctx context.Context, BatchHeaderHash []byte, BlobIndex uint32) ([]byte, error)
	DisperseBlob(ctx context.Context, txData []byte) (*disperser.BlobInfo, error)
}

type EigenDA struct {
	Config

	Log log.Logger
}

func (m *EigenDA) RetrieveBlob(ctx context.Context, BatchHeaderHash []byte, BlobIndex uint32) ([]byte, error) {
	config := &tls.Config{}
	credential := credentials.NewTLS(config)
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(credential)}
	conn, err := grpc.Dial(m.RPC, dialOptions...)
	if err != nil {
		return nil, err
	}
	daClient := disperser.NewDisperserClient(conn)

	reply, err := daClient.RetrieveBlob(ctx, &disperser.RetrieveBlobRequest{
		BatchHeaderHash: BatchHeaderHash,
		BlobIndex:       BlobIndex,
	})
	if err != nil {
		return nil, err
	}

	// decode modulo bn254
	decodedData := codec.RemoveEmptyByteFromPaddedBytes(reply.Data)

	return decodedData, nil
}

func (m *EigenDA) DisperseBlob(ctx context.Context, txData []byte) (*disperser.BlobInfo, error) {
	m.Log.Info("Attempting to disperse blob to EigenDA")
	config := &tls.Config{}
	credential := credentials.NewTLS(config)
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(credential)}
	conn, err := grpc.Dial(m.RPC, dialOptions...)
	if err != nil {
		return nil, err
	}
	daClient := disperser.NewDisperserClient(conn)

	// encode modulo bn254
	encodedTxData := codec.ConvertByPaddingEmptyByte(txData)

	disperseReq := &disperser.DisperseBlobRequest{
		Data: encodedTxData,
	}
	disperseRes, err := daClient.DisperseBlob(ctx, disperseReq)

	if err != nil || disperseRes == nil {
		m.Log.Error("Unable to disperse blob to EigenDA, aborting", "err", err)
		return nil, err
	}

	if disperseRes.Result == disperser.BlobStatus_UNKNOWN ||
		disperseRes.Result == disperser.BlobStatus_FAILED {
		m.Log.Error("Unable to disperse blob to EigenDA, aborting", "err", err)
		return nil, fmt.Errorf("reply status is %d", disperseRes.Result)
	}

	base64RequestID := base64.StdEncoding.EncodeToString(disperseRes.RequestId)

	m.Log.Info("Blob disepersed to EigenDA, now waiting for confirmation", "requestID", base64RequestID)

	var statusRes *disperser.BlobStatusReply
	timeoutTime := time.Now().Add(m.StatusQueryTimeout)
	// Wait before first status check
	time.Sleep(m.StatusQueryRetryInterval)
	for time.Now().Before(timeoutTime) {
		statusRes, err = daClient.GetBlobStatus(ctx, &disperser.BlobStatusRequest{
			RequestId: disperseRes.RequestId,
		})
		if err != nil {
			m.Log.Warn("Unable to retrieve blob dispersal status, will retry", "requestID", base64RequestID, "err", err)
		} else if statusRes.Status == disperser.BlobStatus_CONFIRMED {
			m.Log.Info("EigenDA blob confirmed, waiting for finalization", "requestID", base64RequestID)
		} else if statusRes.Status == disperser.BlobStatus_FINALIZED {
			// Happy path
			batchHeaderHashHex := fmt.Sprintf("0x%s", hex.EncodeToString(statusRes.Info.BlobVerificationProof.BatchMetadata.BatchHeaderHash))
			m.Log.Info("Successfully dispersed blob to EigenDA", "requestID", base64RequestID, "batchHeaderHash", batchHeaderHashHex)
			return statusRes.Info, nil
		} else if statusRes.Status == disperser.BlobStatus_UNKNOWN ||
			statusRes.Status == disperser.BlobStatus_FAILED {
			m.Log.Error("EigenDA blob dispersal failed in processing", "requestID", base64RequestID, "err", err)
			return nil, fmt.Errorf("eigenDA blob dispersal failed in processing with reply status %d", statusRes.Status)
		} else {
			m.Log.Warn("Still waiting for confirmation from EigenDA", "requestID", base64RequestID)
		}

		// Wait before first status check
		time.Sleep(m.StatusQueryRetryInterval)
	}

	return nil, fmt.Errorf("timed out getting EigenDA status for dispersed blob key: %s", base64RequestID)
}
