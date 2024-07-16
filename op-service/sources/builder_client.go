package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	errHTTPErrorResponse = errors.New("HTTP error response")
)

const PathGetPayload = "/eth/v1/builder/payload"

type BuilderAPIConfig struct {
	Endpoint string
}

func BuilderAPIDefaultConfig() *BuilderAPIConfig {
	return &BuilderAPIConfig{
		Endpoint: "",
	}
}

type BuilderAPIClient struct {
	log        log.Logger
	config     *BuilderAPIConfig
	httpClient *client.BasicHTTPClient
}

func NewBuilderAPIClient(log log.Logger, config *BuilderAPIConfig) *BuilderAPIClient {
	httpClient := client.NewBasicHTTPClient(config.Endpoint, log)

	return &BuilderAPIClient{
		httpClient: httpClient,
		config:     config,
		log:        log,
	}
}

func (s *BuilderAPIClient) Enabled() bool {
	return s.config.Endpoint != ""
}

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type VersionedExecutionPayload struct {
	Version   spec.DataVersion
	Bellatrix *bellatrix.ExecutionPayload
	Capella   *capella.ExecutionPayload
	Deneb     *deneb.ExecutionPayload
}

type versionJSON struct {
	Version spec.DataVersion `json:"version"`
}

type bellatrixVersionedExecutionPayloadJSON struct {
	Data *bellatrix.ExecutionPayload `json:"data"`
}

type capellaVersionedExecutionPayloadJSON struct {
	Data *capella.ExecutionPayload `json:"data"`
}

type denebVersionedExecutionPayloadJSON struct {
	Data *deneb.ExecutionPayload `json:"data"`
}

func (v *VersionedExecutionPayload) UnmarshalJSON(input []byte) error {
	var metadata versionJSON
	if err := json.Unmarshal(input, &metadata); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}
	v.Version = metadata.Version
	switch v.Version {
	case spec.DataVersionBellatrix:
		var data bellatrixVersionedExecutionPayloadJSON
		if err := json.Unmarshal(input, &data); err != nil {
			return errors.Wrap(err, "invalid JSON")
		}
		v.Bellatrix = data.Data
	case spec.DataVersionCapella:
		var data capellaVersionedExecutionPayloadJSON
		if err := json.Unmarshal(input, &data); err != nil {
			return errors.Wrap(err, "invalid JSON")
		}
		v.Capella = data.Data
	case spec.DataVersionDeneb:
		var data denebVersionedExecutionPayloadJSON
		if err := json.Unmarshal(input, &data); err != nil {
			return errors.Wrap(err, "invalid JSON")
		}
	default:
		return fmt.Errorf("unsupported data version %v", metadata.Version)
	}

	return nil
}

func (s *BuilderAPIClient) GetPayload(ctx context.Context, ref eth.L2BlockRef, log log.Logger) (*eth.ExecutionPayloadEnvelope, *big.Int, error) {
	responsePayload := new(VersionedExecutionPayload)
	slot := ref.Number + 1
	parentHash := ref.Hash
	url := fmt.Sprintf("%s/%d/%s", PathGetPayload, slot, parentHash.String())
	header := http.Header{"Accept": {"application/json"}}
	resp, err := s.httpClient.Get(ctx, url, nil, header)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, errHTTPErrorResponse
	}

	if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
		return nil, nil, err
	}

	// TODO: Get profit from response
	profit := common.Big0

	return versionedExecutionPayloadToExecutionPayloadEnvelope(responsePayload), profit, nil
}

func reverse(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	for i := len(dst)/2 - 1; i >= 0; i-- {
		opp := len(dst) - 1 - i
		dst[i], dst[opp] = dst[opp], dst[i]
	}
	return dst
}

func versionedExecutionPayloadToExecutionPayloadEnvelope(request *VersionedExecutionPayload) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(request.Deneb.Transactions))

	for i, tx := range request.Deneb.Transactions {
		txs[i] = eth.Data(tx)
	}

	withdrawals := make([]*types.Withdrawal, len(request.Deneb.Withdrawals))
	for i, withdrawal := range request.Deneb.Withdrawals {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(withdrawal.Index),
			Validator: uint64(withdrawal.ValidatorIndex),
			Address:   common.BytesToAddress(withdrawal.Address[:]),
			Amount:    uint64(withdrawal.Amount),
		}
	}

	ws := types.Withdrawals(withdrawals)

	blobGasUsed := eth.Uint64Quantity(request.Deneb.BlobGasUsed)
	excessBlobGas := eth.Uint64Quantity(request.Deneb.ExcessBlobGas)

	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.Hash(request.Deneb.ParentHash),
			FeeRecipient:  common.Address(request.Deneb.FeeRecipient),
			StateRoot:     eth.Bytes32(request.Deneb.StateRoot),
			ReceiptsRoot:  eth.Bytes32(request.Deneb.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(request.Deneb.LogsBloom),
			PrevRandao:    eth.Bytes32(request.Deneb.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(request.Deneb.BlockNumber),
			GasLimit:      eth.Uint64Quantity(request.Deneb.GasLimit),
			GasUsed:       eth.Uint64Quantity(request.Deneb.GasUsed),
			Timestamp:     eth.Uint64Quantity(request.Deneb.Timestamp),
			ExtraData:     eth.BytesMax32(request.Deneb.ExtraData),
			BaseFeePerGas: hexutil.U256(*request.Deneb.BaseFeePerGas),
			BlockHash:     common.BytesToHash(request.Deneb.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   &ws,
			BlobGasUsed:   &blobGasUsed,
			ExcessBlobGas: &excessBlobGas,
		},
		ParentBeaconBlockRoot: nil, // OP-Stack ecotone upgrade related field. Not needed for PoC.
	}
	return payload
}
