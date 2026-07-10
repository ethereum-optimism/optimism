package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The Direct Sync wire carrier: a small JSON envelope around the canonical
// OP-network payload bytes — `snappy(signature(65) || SSZ(payload-envelope))`,
// byte-identical to what OP gossip carries. `signed` is the authoritative
// hot/cold marker (a cold-tier payload has a zeroed signature and MUST NOT be
// gossiped); `seq` is the producer's monotonic feed cursor (hot tier only).
type directSyncMsg struct {
	Version uint8           `json:"version"`
	Signed  bool            `json:"signed"`
	Seq     *hexutil.Uint64 `json:"seq"`
	Payload hexutil.Bytes   `json:"payload"`
}

// decodedDirectSyncPayload is one decoded carrier: the typed envelope (for
// req-resp serving and block/timestamp metadata) plus the decompressed
// `signature || SSZ` gossip body for verbatim publishing.
type decodedDirectSyncPayload struct {
	envelope  *eth.ExecutionPayloadEnvelope
	signature eth.Bytes65
	// raw is the decompressed `signature(65) || SSZ` body — exactly what the
	// gossip topic carries after its own snappy layer.
	raw []byte
}

func (d *decodedDirectSyncPayload) blockNumber() uint64 {
	return uint64(d.envelope.ExecutionPayload.BlockNumber)
}

func (d *decodedDirectSyncPayload) timestamp() uint64 {
	return uint64(d.envelope.ExecutionPayload.Timestamp)
}

// blockVersionFromWire maps the carrier's version (1..=4, the OP-network
// envelope versions) onto op-node's BlockVersion enum.
func blockVersionFromWire(version uint8) (eth.BlockVersion, error) {
	switch version {
	case 1:
		return eth.BlockV1, nil
	case 2:
		return eth.BlockV2, nil
	case 3:
		return eth.BlockV3, nil
	case 4:
		return eth.BlockV4, nil
	default:
		return eth.BlockV1, fmt.Errorf("unknown direct-sync payload version %d", version)
	}
}

// decodeDirectSyncPayload decompresses and decodes one carrier message. The
// SSZ decode mirrors op-node's gossip validator: envelope SSZ (beacon root +
// payload) for v3+, bare payload SSZ otherwise.
func decodeDirectSyncPayload(msg *directSyncMsg) (*decodedDirectSyncPayload, error) {
	version, err := blockVersionFromWire(msg.Version)
	if err != nil {
		return nil, err
	}
	raw, err := snappy.Decode(nil, msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to snappy-decode direct-sync payload: %w", err)
	}
	if len(raw) < 66 {
		return nil, errors.New("direct-sync payload too short for signature + SSZ")
	}
	var signature eth.Bytes65
	copy(signature[:], raw[:65])
	// OP gossip carries the raw recovery id (0/1) in the v byte; normalize a
	// legacy 27/28 defensively. v is recovery metadata, not signed content, so
	// this never invalidates the signature.
	if v := signature[64]; v == 27 || v == 28 {
		signature[64] = v - 27
		raw = append([]byte(nil), raw...) // do not alias the shared buffer
		raw[64] = signature[64]
	}
	payloadBytes := raw[65:]

	var envelope eth.ExecutionPayloadEnvelope
	if version.HasParentBeaconBlockRoot() {
		if err := envelope.UnmarshalSSZ(version, uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)); err != nil {
			return nil, fmt.Errorf("failed to decode v%d execution payload envelope: %w", msg.Version, err)
		}
	} else {
		var payload eth.ExecutionPayload
		if err := payload.UnmarshalSSZ(version, uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)); err != nil {
			return nil, fmt.Errorf("failed to decode v%d execution payload: %w", msg.Version, err)
		}
		envelope = eth.ExecutionPayloadEnvelope{ExecutionPayload: &payload}
	}
	return &decodedDirectSyncPayload{
		envelope:  &envelope,
		signature: signature,
		raw:       raw,
	}, nil
}
