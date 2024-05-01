package eigenda

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/Layr-Labs/eigenda/encoding/utils/codec"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Useful to dinstiguish between plain calldata and alt-da blob refs
// Support seamless migration of existing rollups using ETH DA
const DerivationVersionEigenda = 0xed

type IEigenDA interface {
	RetrieveBlob(ctx context.Context, BatchHeaderHash []byte, BlobIndex uint32) ([]byte, error)
	DisperseBlob(ctx context.Context, txData []byte) (*disperser.BlobInfo, error)
}

type EigenDAClient struct {
	Config

	Log log.Logger
}

func NewEigenDAClient(log log.Logger, config Config) *EigenDAClient {
	return &EigenDAClient{
		Log:    log,
		Config: config,
	}
}

func (m *EigenDAClient) RetrieveBlob(ctx context.Context, BatchHeaderHash []byte, BlobIndex uint32) ([]byte, error) {
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

	// Return exact data with buffer removed
	reader := bytes.NewReader(decodedData)
	length, err := binary.ReadUvarint(reader)
	if err != nil {
		return nil, fmt.Errorf("EigenDA client failed to decode length uvarint prefix")
	}
	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil {
		return nil, fmt.Errorf("EigenDA client failed to copy unpadded data into final buffer")
	}
	if uint64(n) != length {
		return nil, fmt.Errorf("EigenDA client failed, data length does not match length prefix")
	}

	return data, nil
}

func (m *EigenDAClient) DisperseBlob(ctx context.Context, data []byte) (*Cert, error) {
	m.Log.Info("Attempting to disperse blob to EigenDA")
	config := &tls.Config{}
	credential := credentials.NewTLS(config)
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(credential)}
	conn, err := grpc.Dial(m.RPC, dialOptions...)
	if err != nil {
		return nil, err
	}
	daClient := disperser.NewDisperserClient(conn)

	// encode data length
	data = append(ConvertIntToVarUInt(len(data)), data...)

	// encode modulo bn254
	data = codec.ConvertByPaddingEmptyByte(data)

	disperseReq := &disperser.DisperseBlobRequest{
		Data: data,
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

	timeoutTime := time.Now().Add(m.StatusQueryTimeout)
	ticker := time.NewTicker(m.StatusQueryRetryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(timeoutTime) {
				return nil, fmt.Errorf("timed out waiting for EigenDA blob to confirm blob with request id: %s", base64RequestID)
			}
			statusRes, err := daClient.GetBlobStatus(ctx, &disperser.BlobStatusRequest{
				RequestId: disperseRes.RequestId,
			})
			if err != nil {
				m.Log.Warn("Unable to retrieve blob dispersal status, will retry", "requestID", base64RequestID, "err", err)
				continue
			}

			switch statusRes.Status {
			case disperser.BlobStatus_PROCESSING:
				m.Log.Warn("Still waiting for confirmation from EigenDA", "requestID", base64RequestID)
			case disperser.BlobStatus_FAILED:
				m.Log.Error("EigenDA blob dispersal failed in processing", "requestID", base64RequestID, "err", err)
			case disperser.BlobStatus_INSUFFICIENT_SIGNATURES:
				m.Log.Error("EigenDA blob dispersal failed in processing with insufficient signatures", "requestID", base64RequestID, "err", err)
			case disperser.BlobStatus_CONFIRMED:
				m.Log.Info("EigenDA blob confirmed, waiting for finalization", "requestID", base64RequestID)
			case disperser.BlobStatus_FINALIZED:
				batchHeaderHashHex := fmt.Sprintf("0x%s", hex.EncodeToString(statusRes.Info.BlobVerificationProof.BatchMetadata.BatchHeaderHash))
				m.Log.Info("Successfully dispersed blob to EigenDA", "requestID", base64RequestID, "batchHeaderHash", batchHeaderHashHex)
				blobInfo := statusRes.Info
				quorumIDs := make([]uint32, len(blobInfo.BlobHeader.BlobQuorumParams))
				for i := range quorumIDs {
					quorumIDs[i] = blobInfo.BlobHeader.BlobQuorumParams[i].QuorumNumber
				}
				cert := &Cert{
					BatchHeaderHash:      blobInfo.BlobVerificationProof.BatchMetadata.BatchHeaderHash,
					BlobIndex:            blobInfo.BlobVerificationProof.BlobIndex,
					ReferenceBlockNumber: blobInfo.BlobVerificationProof.BatchMetadata.BatchHeader.ReferenceBlockNumber,
					QuorumIDs:            quorumIDs,
				}
				return cert, nil
			default:
				return nil, fmt.Errorf("EigenDA blob dispersal failed in processing with reply status %d", statusRes.Status)
			}
		}
	}
}

func ConvertIntToVarUInt(v int) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(v))
	return buf[:n]

}
