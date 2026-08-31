package render

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
)

// This file is the ONE place the rendering's contract bindings live, so that rebinding them is a
// one-file diff. Nothing else in op-private-interop or op-batcher may hardcode a selector or a
// topic.
//
// The two replay signatures below are FINAL, bound to the contracts on karl/private-interop:
//
//	L2ToL2CrossDomainMessengerReplay.replaySentMessage(uint256,uint256,address,address,bytes)
//	EventReplayer.replayEvent(bytes32[],bytes)
//
// The messenger replay is installed at the L2ToL2CrossDomainMessenger predeploy address and emits a
// SentMessage byte-identical to the private chain's, so an export renders at the address every
// stock consumer expects. EventReplayer emits at ITS OWN address — see EventReplayerAddr.
//
// The claim binding:
//
//	ClaimRegistry.postClaim(RangeClaim) — LOG-LESS, no event
//
// The struct is the one op-private-interop/codec already encodes, and this file owns only the
// selector: the codec stays the single owner of the claim's bytes, and the argument list of a
// function taking one dynamic struct encodes exactly as abi.encode of that struct, so the calldata
// is a selector followed by exactly what codec.Encode produces.
//
// This file holds NO contract addresses. The EventReplayer and the ClaimRegistry are predeploys on
// the rendering -- predeploys.EventReplayerAddr (0x...002F) and predeploys.ClaimRegistryAddr
// (0x...002E), placed by the PRIVATE_INTEROP_RENDERING genesis -- but an OperatorTxBuilder starts
// with both unset and REFUSES to construct a transaction until a caller wires them in
// (SetEventReplayer, SetRegistry). Defaulting them here would trade a loud configuration error for
// a silent one: a deployment that placed either contract elsewhere would then send to the standard
// address, and a replay transaction that reverts is a rendering block that quietly lost a log. The
// predeploy constants are what a caller should wire in absent a reason not to.

var (
	// SentMessageEventTopic is topic0 of L2ToL2CrossDomainMessenger's SentMessage.
	//
	// Declared here because the repo has no Go constant for it — the contract carries it as
	// SENT_MESSAGE_EVENT_SELECTOR and every Go consumer so far has only ever needed to decode logs
	// it already knows are messenger logs. The rendering needs to RECOGNISE one.
	SentMessageEventTopic = crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)"))

	// RelayedMessageEventTopic is topic0 of L2ToL2CrossDomainMessenger's RelayedMessage. The
	// private chain emits it whenever it IMPORTS a message, so it is in the emitter set and must be
	// rendered — but the replay messenger cannot emit it. See ReplayEvent.
	RelayedMessageEventTopic = crypto.Keccak256Hash([]byte("RelayedMessage(uint256,uint256,bytes32,bytes32)"))

	// ReplaySentMessageSig is the messenger replay's entry point.
	ReplaySentMessageSig = "replaySentMessage(uint256,uint256,address,address,bytes)"

	// ReplayEventSig is the generic replayer's entry point.
	ReplayEventSig = "replayEvent(bytes32[],bytes)"

	// PostClaimSig is the registry's entry point, over the struct the codec encodes.
	//
	// The field list is DERIVED from the codec's own ABI type rather than written out here. A
	// hand-written copy is a second source of truth, and when the claim gained
	// privateTerminalParentHash this line kept the old nine-field list: the batcher then sent
	// selector 0x41a02b4d to a registry whose postClaim is 0x4db071ca, so every claim transaction
	// hit a contract with no fallback and reverted. Nothing in Go noticed, because the follow module
	// compares incoming calldata against THIS SAME constant -- producer and reader agreed with each
	// other and only the chain disagreed. TestPostClaimSelectorMatchesTheDeployedRegistry pins the
	// result against the compiled contract so the two can never drift apart again.
	PostClaimSig = "postClaim(" + codec.ClaimTupleType() + ")"
)

// ReplaySentMessageSelector and ReplayEventSelector are the 4-byte selectors of the signatures
// above.
var (
	ReplaySentMessageSelector = selector(ReplaySentMessageSig)
	ReplayEventSelector       = selector(ReplayEventSig)
	PostClaimSelector         = selector(PostClaimSig)
)

// EncodePostClaim builds the calldata for the claim transaction: a selector and a delegation to the
// codec, which owns the bytes on both sides of the wire.
func EncodePostClaim(claim *codec.RangeClaim) ([]byte, error) {
	body, err := codec.Encode(claim)
	if err != nil {
		return nil, fmt.Errorf("encoding claim for postClaim: %w", err)
	}
	return append(PostClaimSelector[:], body...), nil
}

func selector(sig string) [4]byte {
	var out [4]byte
	copy(out[:], crypto.Keccak256([]byte(sig))[:4])
	return out
}

// SentMessage is the private chain's SentMessage event, decoded into the fields the messenger
// replay takes.
//
// The replay call takes the DECODED fields rather than the raw topics and data, because the replay
// contract re-emits the event itself: it is the contract, not the operator, that guarantees the
// emitted log is byte-identical to a natively emitted one. RoundTrip below is the check that the
// decode was faithful, and it runs on every export.
type SentMessage struct {
	Destination *big.Int
	Nonce       *big.Int
	Sender      common.Address
	Target      common.Address
	Message     []byte
}

var sentMessageDataArgs = mustArgs("address", "bytes")

// DecodeSentMessage reads a private SentMessage log.
//
//	topic0 = SentMessage(...)      topic1 = destination     topic2 = target     topic3 = nonce
//	data   = abi.encode(address sender, bytes message)
func DecodeSentMessage(topics []common.Hash, data []byte) (*SentMessage, error) {
	if len(topics) != 4 {
		return nil, fmt.Errorf("SentMessage has %d topics, expected 4", len(topics))
	}
	if topics[0] != SentMessageEventTopic {
		return nil, fmt.Errorf("topic0 %s is not SentMessage", topics[0])
	}
	values, err := sentMessageDataArgs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("decoding the SentMessage data section: %w", err)
	}
	sender, ok := values[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("SentMessage sender is %T", values[0])
	}
	message, ok := values[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("SentMessage message is %T", values[1])
	}
	out := &SentMessage{
		Destination: new(big.Int).SetBytes(topics[1][:]),
		Nonce:       new(big.Int).SetBytes(topics[3][:]),
		Sender:      sender,
		Target:      common.BytesToAddress(topics[2][:]),
		Message:     message,
	}
	// A target topic with anything in its top 12 bytes is not an address, and truncating it would
	// render a message to a DIFFERENT target than the private chain sent it to.
	if topics[2] != common.BytesToHash(out.Target[:]) {
		return nil, fmt.Errorf("SentMessage target topic %s is not an address", topics[2])
	}
	return out, nil
}

// EncodeReplaySentMessage builds the calldata for one export re-emission.
func EncodeReplaySentMessage(m *SentMessage) ([]byte, error) {
	data, err := replaySentMessageArgs.Pack(m.Destination, m.Nonce, m.Sender, m.Target, m.Message)
	if err != nil {
		return nil, fmt.Errorf("encoding replaySentMessage: %w", err)
	}
	return append(ReplaySentMessageSelector[:], data...), nil
}

// EncodeReplayEvent builds the calldata for one generic log re-emission.
func EncodeReplayEvent(topics []common.Hash, data []byte) ([]byte, error) {
	if len(topics) > 4 {
		// The EVM has no log5, and EventReplayer refuses it. Refusing here too means the failure is
		// a build error naming the log rather than a reverted transaction on a live rendering.
		return nil, fmt.Errorf("a log with %d topics cannot be emitted; the maximum is 4", len(topics))
	}
	arr := make([][32]byte, len(topics))
	for i, h := range topics {
		arr[i] = h
	}
	packed, err := replayEventArgs.Pack(arr, data)
	if err != nil {
		return nil, fmt.Errorf("encoding replayEvent: %w", err)
	}
	return append(ReplayEventSelector[:], packed...), nil
}

var (
	replaySentMessageArgs = mustArgs("uint256", "uint256", "address", "address", "bytes")
	replayEventArgs       = mustArgs("bytes32[]", "bytes")
)

func mustArgs(types ...string) abi.Arguments {
	out := make(abi.Arguments, 0, len(types))
	for _, t := range types {
		ty, err := abi.NewType(t, "", nil)
		if err != nil {
			panic(fmt.Errorf("private-interop ABI type %q: %w", t, err))
		}
		out = append(out, abi.Argument{Type: ty})
	}
	return out
}
