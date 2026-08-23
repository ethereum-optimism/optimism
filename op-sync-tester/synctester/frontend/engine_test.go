package frontend

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// payloadStub answers every getPayload with the same stored envelope — the gossip shape the
// backend keeps, carrying only the payload and the parent beacon block root.
type payloadStub struct {
	apis.EngineAPI
	env *eth.ExecutionPayloadEnvelope
}

func (s *payloadStub) GetPayloadV1(context.Context, eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return s.env, nil
}
func (s *payloadStub) GetPayloadV2(context.Context, eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return s.env, nil
}
func (s *payloadStub) GetPayloadV3(context.Context, eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return s.env, nil
}
func (s *payloadStub) GetPayloadV4(context.Context, eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return s.env, nil
}
func (s *payloadStub) GetPayloadV5(context.Context, eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return s.env, nil
}

// The getPayload responses must carry every field the execution-apis spec requires for their
// version — a spec-strict client (alloy's payload envelopes, behind kona-node) hard-fails on a
// missing blockValue/blobsBundle/shouldOverrideBuilder/parentBeaconBlockRoot/executionRequests,
// which is exactly how the sync tester wedged a kona verifier that had to build blocks: the
// stored gossip envelope carries none of them. Go clients ignore the extra fields, so filling
// the spec shape in is compatible both ways.
func TestGetPayloadResponsesAreSpecShaped(t *testing.T) {
	beaconRoot := common.HexToHash("0xbeac04")
	fe := NewEngineFrontend(&payloadStub{env: &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &beaconRoot,
		// A non-nil (empty) transaction list, so the payload itself marshals without nulls and
		// the null check below covers only the wrapper's fields.
		ExecutionPayload: &eth.ExecutionPayload{BlockNumber: 1, Transactions: []eth.Data{}},
	}})
	ctx := context.Background()

	type call struct {
		get      func() (*GetPayloadResponse, error)
		required []string
		absent   []string
	}
	v3Fields := []string{`"executionPayload"`, `"blockValue"`, `"blobsBundle"`, `"shouldOverrideBuilder"`, `"parentBeaconBlockRoot"`}
	calls := map[string]call{
		"v2": {
			get:      func() (*GetPayloadResponse, error) { return fe.GetPayloadV2(ctx, eth.PayloadID{}) },
			required: []string{`"executionPayload"`, `"blockValue"`},
			absent:   []string{`"blobsBundle"`, `"executionRequests"`},
		},
		"v3": {
			get:      func() (*GetPayloadResponse, error) { return fe.GetPayloadV3(ctx, eth.PayloadID{}) },
			required: v3Fields,
			absent:   []string{`"executionRequests"`},
		},
		"v4": {
			get:      func() (*GetPayloadResponse, error) { return fe.GetPayloadV4(ctx, eth.PayloadID{}) },
			required: append([]string{`"executionRequests"`}, v3Fields...),
		},
		"v5": {
			get:      func() (*GetPayloadResponse, error) { return fe.GetPayloadV5(ctx, eth.PayloadID{}) },
			required: append([]string{`"executionRequests"`}, v3Fields...),
		},
	}
	for name, tc := range calls {
		t.Run(name, func(t *testing.T) {
			resp, err := tc.get()
			require.NoError(t, err)
			raw, err := json.Marshal(resp)
			require.NoError(t, err)
			body := string(raw)
			for _, field := range tc.required {
				require.Contains(t, body, field, "spec-required field must be served")
			}
			for _, field := range tc.absent {
				require.NotContains(t, body, field, "field is not part of this version's response")
			}
			require.NotContains(t, body, "null", "spec-strict clients reject null for required arrays")
			if strings.Contains(body, `"parentBeaconBlockRoot"`) {
				require.Contains(t, body, beaconRoot.String(), "the stored beacon root must be relayed")
			}
		})
	}
}
