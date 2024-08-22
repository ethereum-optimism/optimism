package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	builderSpec "github.com/attestantio/go-builder-client/spec"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const PathGetPayload = "/eth/v1/builder/payload"
const GenesisForkVersionMainnet = "0x00000000" // NOTE: Optimism does not have any fork version. Use Mainnet fork version for now.

type BuilderAPIConfig struct {
	Timeout  time.Duration
	Endpoint string
}

type BuilderAPIClient struct {
	log           log.Logger
	config        *BuilderAPIConfig
	rollupCfg     *rollup.Config
	httpClient    *client.BasicHTTPClient
	domainBuilder phase0.Domain
}

func verifySignature(submission *builderSpec.VersionedSubmitBlockRequest, domainBuilder phase0.Domain) error {
	bid, err := submission.BidTrace()
	if err != nil {
		return err
	}

	signature, err := submission.Signature()
	if err != nil {
		return err
	}

	builderPubKey := bid.BuilderPubkey

	ok, err := ssz.VerifySignature(bid, domainBuilder, builderPubKey[:], signature[:])
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("invalid builder signature")
	}
	return nil
}

func computeDomain(domainType phase0.DomainType, forkVersionHex, genesisValidatorsRootHex string) (domain phase0.Domain, err error) {
	genesisValidatorsRoot := phase0.Root(common.HexToHash(genesisValidatorsRootHex))
	forkVersionBytes, err := hexutil.Decode(forkVersionHex)
	if err != nil || len(forkVersionBytes) != 4 {
		return domain, errors.New("invalid fork version")
	}
	var forkVersion [4]byte
	copy(forkVersion[:], forkVersionBytes[:4])
	return ssz.ComputeDomain(domainType, forkVersion, genesisValidatorsRoot), nil
}

func NewBuilderClient(log log.Logger, rollupCfg *rollup.Config, endpoint string, timeout time.Duration) *BuilderAPIClient {
	domainBuilder, err := computeDomain(ssz.DomainTypeAppBuilder, GenesisForkVersionMainnet, phase0.Root{}.String())
	if err != nil {
		log.Error("failed to compute domain", "error", err)
	}

	httpClient := client.NewBasicHTTPClient(endpoint, log)
	config := &BuilderAPIConfig{
		Timeout:  timeout,
		Endpoint: endpoint,
	}

	return &BuilderAPIClient{
		httpClient:    httpClient,
		config:        config,
		rollupCfg:     rollupCfg,
		log:           log,
		domainBuilder: domainBuilder,
	}
}

func (s *BuilderAPIClient) Enabled() bool {
	return true
}

func (s *BuilderAPIClient) Timeout() time.Duration {
	return s.config.Timeout
}

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *BuilderAPIClient) GetPayload(ctx context.Context, ref eth.L2BlockRef, log log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
	submitBlockRequest := new(builderSpec.VersionedSubmitBlockRequest)
	slot := ref.Number + 1
	parentHash := ref.Hash
	url := fmt.Sprintf("%s/%d/%s", PathGetPayload, slot, parentHash.String())
	header := http.Header{"Accept": {"application/json"}}
	resp, err := s.httpClient.Get(ctx, url, nil, header)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp httpErrorResp
		if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
			log.Warn("failed to unmarshal error response", "error", err, "response", string(bodyBytes))
			return nil, fmt.Errorf("HTTP error response: %v", resp.Status)
		}
		return nil, fmt.Errorf("HTTP error response: %v", errResp.Message)
	}

	if err := json.Unmarshal(bodyBytes, submitBlockRequest); err != nil {
		return nil, err
	}

	if err := verifySignature(submitBlockRequest, s.domainBuilder); err != nil {
		return nil, err
	}

	// selects expected data version from the optimism version.
	// Bedrock - Bellatrix
	// Canyon - Capella
	// Ecotone - Deneb
	var expectedVersion spec.DataVersion
	if s.rollupCfg.IsEcotone(ref.Time) {
		expectedVersion = spec.DataVersionDeneb
	} else if s.rollupCfg.IsCanyon(ref.Time) {
		expectedVersion = spec.DataVersionCapella
	} else {
		expectedVersion = spec.DataVersionBellatrix
	}

	if expectedVersion != submitBlockRequest.Version {
		return nil, fmt.Errorf("expected version %s, got %s", expectedVersion, submitBlockRequest.Version)
	}

	envelope, err := getExecutionPayloadEnvelope(submitBlockRequest, ref)
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

func getExecutionPayloadEnvelope(blockRequest *builderSpec.VersionedSubmitBlockRequest, ref eth.L2BlockRef) (*eth.ExecutionPayloadEnvelope, error) {
	switch blockRequest.Version {
	case spec.DataVersionBellatrix:
		return bellatrixExecutionPayloadToExecutionPayload(blockRequest.Bellatrix.ExecutionPayload), nil
	case spec.DataVersionCapella:
		return capellaExecutionPayloadToExecutionPayload(blockRequest.Capella.ExecutionPayload), nil
	case spec.DataVersionDeneb:
		return denebExecutionPayloadToExecutionPayload(blockRequest.Deneb.ExecutionPayload), nil
	default:
		return nil, fmt.Errorf("unsupported version: %s", blockRequest.Version)
	}
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

func bellatrixExecutionPayloadToExecutionPayload(payload *bellatrix.ExecutionPayload) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(payload.Transactions))
	for i, tx := range payload.Transactions {
		txs[i] = eth.Data(tx)
	}

	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.Hash(payload.ParentHash),
			FeeRecipient:  common.Address(payload.FeeRecipient),
			StateRoot:     eth.Bytes32(payload.StateRoot),
			ReceiptsRoot:  eth.Bytes32(payload.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(payload.LogsBloom),
			PrevRandao:    eth.Bytes32(payload.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(payload.BlockNumber),
			GasLimit:      eth.Uint64Quantity(payload.GasLimit),
			GasUsed:       eth.Uint64Quantity(payload.GasUsed),
			Timestamp:     eth.Uint64Quantity(payload.Timestamp),
			ExtraData:     eth.BytesMax32(payload.ExtraData),
			BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(0).SetBytes(reverse(payload.BaseFeePerGas[:]))),
			BlockHash:     common.BytesToHash(payload.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   nil,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		},
		ParentBeaconBlockRoot: nil,
	}
	return envelope
}

func capellaExecutionPayloadToExecutionPayload(payload *capella.ExecutionPayload) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(payload.Transactions))
	for i, tx := range payload.Transactions {
		txs[i] = eth.Data(tx)
	}

	withdrawals := make([]*types.Withdrawal, len(payload.Withdrawals))
	for i, withdrawal := range payload.Withdrawals {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(withdrawal.Index),
			Validator: uint64(withdrawal.ValidatorIndex),
			Address:   common.BytesToAddress(withdrawal.Address[:]),
			Amount:    uint64(withdrawal.Amount),
		}
	}
	ws := types.Withdrawals(withdrawals)

	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.Hash(payload.ParentHash),
			FeeRecipient:  common.Address(payload.FeeRecipient),
			StateRoot:     eth.Bytes32(payload.StateRoot),
			ReceiptsRoot:  eth.Bytes32(payload.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(payload.LogsBloom),
			PrevRandao:    eth.Bytes32(payload.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(payload.BlockNumber),
			GasLimit:      eth.Uint64Quantity(payload.GasLimit),
			GasUsed:       eth.Uint64Quantity(payload.GasUsed),
			Timestamp:     eth.Uint64Quantity(payload.Timestamp),
			ExtraData:     eth.BytesMax32(payload.ExtraData),
			BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(0).SetBytes(reverse(payload.BaseFeePerGas[:]))),
			BlockHash:     common.BytesToHash(payload.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   &ws,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		},
		ParentBeaconBlockRoot: nil,
	}
	return envelope
}

func denebExecutionPayloadToExecutionPayload(payload *deneb.ExecutionPayload) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(payload.Transactions))
	for i, tx := range payload.Transactions {
		txs[i] = eth.Data(tx)
	}

	withdrawals := make([]*types.Withdrawal, len(payload.Withdrawals))
	for i, withdrawal := range payload.Withdrawals {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(withdrawal.Index),
			Validator: uint64(withdrawal.ValidatorIndex),
			Address:   common.BytesToAddress(withdrawal.Address[:]),
			Amount:    uint64(withdrawal.Amount),
		}
	}
	ws := types.Withdrawals(withdrawals)

	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.Hash(payload.ParentHash),
			FeeRecipient:  common.Address(payload.FeeRecipient),
			StateRoot:     eth.Bytes32(payload.StateRoot),
			ReceiptsRoot:  eth.Bytes32(payload.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(payload.LogsBloom),
			PrevRandao:    eth.Bytes32(payload.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(payload.BlockNumber),
			GasLimit:      eth.Uint64Quantity(payload.GasLimit),
			GasUsed:       eth.Uint64Quantity(payload.GasUsed),
			Timestamp:     eth.Uint64Quantity(payload.Timestamp),
			ExtraData:     eth.BytesMax32(payload.ExtraData),
			BaseFeePerGas: eth.Uint256Quantity(*payload.BaseFeePerGas),
			BlockHash:     common.BytesToHash(payload.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   &ws,
			BlobGasUsed:   (*eth.Uint64Quantity)(&payload.BlobGasUsed),
			ExcessBlobGas: (*eth.Uint64Quantity)(&payload.ExcessBlobGas),
		},
		ParentBeaconBlockRoot: nil,
	}
	return envelope
}
