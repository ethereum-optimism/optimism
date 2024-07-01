package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	builderApi "github.com/attestantio/go-builder-client/api"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
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

func (s *BuilderAPIClient) GetPayload(ctx context.Context, ref eth.L2BlockRef, log log.Logger) (*eth.ExecutionPayloadEnvelope, *big.Int, error) {
	responsePayload := new(builderApi.VersionedExecutionPayload)
	url := fmt.Sprintf("%s/%d/%s", PathGetPayload, ref.Number+1, ref.Hash)
	log.Info("Fetching payload", "url", url)
	header := http.Header{"Accept": {"application/json"}}
	resp, err := s.httpClient.Get(ctx, url, nil, header)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	log.Info("Response", "status", resp.Status, "header", resp.Header, "statuscode", resp.StatusCode, "body", resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Error("Failed to read response body", "error", err)
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp httpErrorResp
		err = json.Unmarshal(bodyBytes, &errResp)
		if err != nil {
			log.Error("Failed to unmarshal error response", "error", err)
		}
		log.Error("HTTP error response", "status", resp.Status, "code", errResp.Code, "message", errResp.Message)
		return nil, nil, errHTTPErrorResponse
	}

	if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
		log.Error("Failed to unmarshal response payload", "error", err)
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

func versionedExecutionPayloadToExecutionPayloadEnvelope(request *builderApi.VersionedExecutionPayload) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(request.Capella.Transactions))

	for i, tx := range request.Capella.Transactions {
		txs[i] = eth.Data(tx)
	}

	withdrawals := make([]*types.Withdrawal, len(request.Capella.Withdrawals))
	for i, withdrawal := range request.Capella.Withdrawals {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(withdrawal.Index),
			Validator: uint64(withdrawal.ValidatorIndex),
			Address:   common.BytesToAddress(withdrawal.Address[:]),
			Amount:    uint64(withdrawal.Amount),
		}
	}

	ws := types.Withdrawals(withdrawals)

	baseFeePerGas := new(big.Int).SetBytes(reverse(request.Capella.BaseFeePerGas[:]))
	baseFeePerGasUint := uint256.NewInt(0)
	baseFeePerGasUint.SetFromBig(baseFeePerGas)

	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.Hash(request.Capella.ParentHash),
			FeeRecipient:  common.Address(request.Capella.FeeRecipient),
			StateRoot:     eth.Bytes32(request.Capella.StateRoot),
			ReceiptsRoot:  eth.Bytes32(request.Capella.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(request.Capella.LogsBloom),
			PrevRandao:    eth.Bytes32(request.Capella.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(request.Capella.BlockNumber),
			GasLimit:      eth.Uint64Quantity(request.Capella.GasLimit),
			GasUsed:       eth.Uint64Quantity(request.Capella.GasUsed),
			Timestamp:     eth.Uint64Quantity(request.Capella.Timestamp),
			ExtraData:     eth.BytesMax32(request.Capella.ExtraData),
			BaseFeePerGas: hexutil.U256(*baseFeePerGasUint),
			BlockHash:     common.BytesToHash(request.Capella.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   &ws,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		},
		ParentBeaconBlockRoot: nil, // OP-Stack ecotone upgrade related field. Not needed for PoC.
	}
	return payload
}
