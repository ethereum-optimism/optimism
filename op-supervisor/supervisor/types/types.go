package types

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	errLogIndexTooLarge        = errors.New("log index too large")
	errNilSafetyLevel          = errors.New("nil safety level")
	errUnrecognizedSafetyLevel = errors.New("unrecognized safety level")
	errInvalidParentBlock      = errors.New("invalid parent block")
)

var ExecutingMessageEventTopic = crypto.Keccak256Hash([]byte("ExecutingMessage(bytes32,(address,uint256,uint256,uint256,uint256))"))

type Revision uint64

// RevisionAny is used as indicator to ignore the revision during lookups.
// This is used in the cross-safe queries,
// where there will only ever be a single derived block per derived block number,
// but where the revision is still tracked to match the local-safe DB block replacements.
// We use the max-uint64 value, since this is reserved, and will not be allowed to decode/encode.
const RevisionAny = ^Revision(0)

func (r Revision) Any() bool {
	return r == RevisionAny
}

// Number returns the block-number, where the revision started (i.e. the invalidated/replacement block height)
func (r Revision) Number() uint64 {
	return uint64(r) &^ uint64(1<<63)
}

func (r Revision) String() string {
	if r.Any() {
		return "Rev(any)"
	}
	return fmt.Sprintf("Rev(%d)", r.Number())
}

// Cmp returns:
// 0 if the revision matches any block number
// 1 if the revision is higher than the given number
// 0 if the revision is equal than the given number
// -1 if the revision is lower than the given number
func (r Revision) Cmp(blockNum uint64) int {
	if r.Any() {
		return 0
	}
	if r.Number() > blockNum {
		return 1
	}
	if r.Number() == blockNum {
		return 0
	}
	return -1
}

// ContainsQuery contains all the information needed to check a message
// against a chain's database, to determine if it is valid (ie all invariants hold).
type ContainsQuery struct {
	Timestamp uint64
	BlockNum  uint64
	LogIdx    uint32
	Checksum  MessageChecksum
}

type ExecutingMessage struct {
	ChainID   eth.ChainID
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Checksum  MessageChecksum
}

func (s *ExecutingMessage) String() string {
	return fmt.Sprintf("ExecMsg(chain: %s, block: %d, log: %d, time: %d, checksum: %s)",
		s.ChainID, s.BlockNum, s.LogIdx, s.Timestamp, s.Checksum)
}

type Message struct {
	Identifier  Identifier  `json:"identifier"`
	PayloadHash common.Hash `json:"payloadHash"`
}

func (m *Message) ToCheckSumArgs() ChecksumArgs {
	return ChecksumArgs{
		BlockNumber: m.Identifier.BlockNumber,
		LogIndex:    m.Identifier.LogIndex,
		Timestamp:   m.Identifier.Timestamp,
		ChainID:     m.Identifier.ChainID,
		LogHash:     PayloadHashToLogHash(m.PayloadHash, m.Identifier.Origin),
	}
}

func (m *Message) Checksum() MessageChecksum {
	return m.ToCheckSumArgs().Checksum()
}

func (m *Message) Access() Access {
	return m.ToCheckSumArgs().Access()
}

func (m *Message) DecodeEvent(topics []common.Hash, data []byte) error {
	if len(topics) != 2 { // event hash, indexed payloadHash
		return fmt.Errorf("unexpected number of event topics: %d", len(topics))
	}
	if topics[0] != ExecutingMessageEventTopic {
		return fmt.Errorf("unexpected event topic %q", topics[0])
	}
	if len(data) != 32*5 {
		return fmt.Errorf("unexpected identifier data length: %d", len(data))
	}
	take := func(length uint) []byte {
		taken := data[:length]
		data = data[length:]
		return taken
	}
	takeZeroes := func(length uint) error {
		for _, v := range take(length) {
			if v != 0 {
				return errors.New("expected zero")
			}
		}
		return nil
	}
	if err := takeZeroes(12); err != nil {
		return fmt.Errorf("invalid address padding: %w", err)
	}
	m.Identifier.Origin = common.Address(take(20))
	if err := takeZeroes(32 - 8); err != nil {
		return fmt.Errorf("invalid block number padding: %w", err)
	}
	m.Identifier.BlockNumber = binary.BigEndian.Uint64(take(8))
	if err := takeZeroes(32 - 4); err != nil {
		return fmt.Errorf("invalid log index padding: %w", err)
	}
	m.Identifier.LogIndex = binary.BigEndian.Uint32(take(4))
	if err := takeZeroes(32 - 8); err != nil {
		return fmt.Errorf("invalid timestamp padding: %w", err)
	}
	m.Identifier.Timestamp = binary.BigEndian.Uint64(take(8))
	m.Identifier.ChainID = eth.ChainIDFromBytes32([32]byte(take(32)))
	m.PayloadHash = topics[1]
	return nil
}

type ChecksumArgs struct {
	BlockNumber uint64
	LogIndex    uint32
	Timestamp   uint64
	ChainID     eth.ChainID
	LogHash     common.Hash
}

func (args ChecksumArgs) Checksum() MessageChecksum {
	idPacked := make([]byte, 12, 32) // 12 zero bytes, as padding to 32 bytes
	idPacked = binary.BigEndian.AppendUint64(idPacked, args.BlockNumber)
	idPacked = binary.BigEndian.AppendUint64(idPacked, args.Timestamp)
	idPacked = binary.BigEndian.AppendUint32(idPacked, args.LogIndex)
	idLogHash := crypto.Keccak256Hash(args.LogHash[:], idPacked)
	chainID := args.ChainID.Bytes32()
	out := crypto.Keccak256Hash(idLogHash[:], chainID[:])
	out[0] = 0x03 // type/version byte
	return MessageChecksum(out)
}

func (args ChecksumArgs) Access() Access {
	return Access{
		BlockNumber: args.BlockNumber,
		Timestamp:   args.Timestamp,
		LogIndex:    args.LogIndex,
		ChainID:     args.ChainID,
		Checksum:    args.Checksum(),
	}
}

func (args ChecksumArgs) Query() ContainsQuery {
	return ContainsQuery{
		BlockNum:  args.BlockNumber,
		Timestamp: args.Timestamp,
		LogIdx:    args.LogIndex,
		Checksum:  args.Checksum(),
	}
}

type Identifier struct {
	Origin      common.Address
	BlockNumber uint64
	LogIndex    uint32
	Timestamp   uint64
	ChainID     eth.ChainID // flat, not a pointer, to make Identifier safe as map key
}

type identifierMarshaling struct {
	Origin      common.Address `json:"origin"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	LogIndex    hexutil.Uint64 `json:"logIndex"`
	Timestamp   hexutil.Uint64 `json:"timestamp"`
	ChainID     hexutil.U256   `json:"chainID"`
}

func (id Identifier) MarshalJSON() ([]byte, error) {
	var enc identifierMarshaling
	enc.Origin = id.Origin
	enc.BlockNumber = hexutil.Uint64(id.BlockNumber)
	enc.LogIndex = hexutil.Uint64(id.LogIndex)
	enc.Timestamp = hexutil.Uint64(id.Timestamp)
	enc.ChainID = (hexutil.U256)(id.ChainID)
	return json.Marshal(&enc)
}

func (id *Identifier) UnmarshalJSON(input []byte) error {
	var dec identifierMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	id.Origin = dec.Origin
	id.BlockNumber = uint64(dec.BlockNumber)
	if dec.LogIndex > math.MaxUint32 {
		return fmt.Errorf("%w: %d", errLogIndexTooLarge, dec.LogIndex)
	}
	id.LogIndex = uint32(dec.LogIndex)
	id.Timestamp = uint64(dec.Timestamp)
	id.ChainID = (eth.ChainID)(dec.ChainID)
	return nil
}

func (id Identifier) ChecksumArgs(msgHash common.Hash) ChecksumArgs {
	return ChecksumArgs{
		BlockNumber: id.BlockNumber,
		Timestamp:   id.Timestamp,
		LogIndex:    id.LogIndex,
		ChainID:     id.ChainID,
		LogHash:     PayloadHashToLogHash(msgHash, id.Origin),
	}
}

type SafetyLevel string

func (lvl SafetyLevel) String() string {
	return string(lvl)
}

// Validate returns true if the SafetyLevel is one of the recognized levels
func (lvl SafetyLevel) Validate() bool {
	switch lvl {
	case Invalid, Finalized, CrossSafe, LocalSafe, CrossUnsafe, LocalUnsafe:
		return true
	default:
		return false
	}
}

func (lvl SafetyLevel) MarshalText() ([]byte, error) {
	return []byte(lvl), nil
}

func (lvl *SafetyLevel) UnmarshalText(text []byte) error {
	if lvl == nil {
		return errNilSafetyLevel
	}
	x := SafetyLevel(text)
	if !x.Validate() {
		return fmt.Errorf("%w: %q", errUnrecognizedSafetyLevel, text)
	}
	*lvl = x
	return nil
}

const (
	// Finalized is CrossSafe, with the additional constraint that every
	// dependency is derived only from finalized L1 input data.
	// This matches RPC label "finalized".
	Finalized SafetyLevel = "finalized"
	// CrossSafe is as safe as LocalSafe, with all its dependencies
	// also fully verified to be reproducible from L1.
	// This matches RPC label "safe".
	CrossSafe SafetyLevel = "safe"
	// LocalSafe is verified to be reproducible from L1,
	// without any verified cross-L2 dependencies.
	// This does not have an RPC label.
	LocalSafe SafetyLevel = "local-safe"
	// CrossUnsafe is as safe as LocalUnsafe,
	// but with verified cross-L2 dependencies that are at least CrossUnsafe.
	// This does not have an RPC label.
	CrossUnsafe SafetyLevel = "cross-unsafe"
	// LocalUnsafe is the safety of the tip of the chain. This matches RPC label "unsafe".
	LocalUnsafe SafetyLevel = "unsafe"
	// Invalid is the safety of when the message or block is not matching the expected data.
	Invalid SafetyLevel = "invalid"
)

type ExecutingDescriptor struct {
	// Timestamp is the timestamp of the executing message
	Timestamp uint64

	// Timeout, requests verification to still hold at Timestamp+Timeout (incl.). Defaults to 0.
	// I.e. Timestamp is used as lower-bound validity, and Timeout defines the span to the upper-bound.
	Timeout uint64
}

func (ed *ExecutingDescriptor) AccessCheck(expiryWindow uint64, initMsgTimestamp uint64) error {
	// Check upper-bound invariant, strictly
	// (for access-lists we don't afford to check intra-timestamp dependencies)
	if ed.Timestamp < initMsgTimestamp {
		return fmt.Errorf("message broke timestamp invariant: exec: %d, init: %d, %w",
			ed.Timestamp, initMsgTimestamp, ErrConflict)
	}
	if ed.Timestamp == initMsgTimestamp {
		return fmt.Errorf("access-list check does not allow intra-timestamp (%d): %w", ed.Timestamp, ErrConflict)
	}

	// Check message expiry
	expiryAt := initMsgTimestamp + expiryWindow
	if expiryAt < initMsgTimestamp {
		return fmt.Errorf("message timestamp too high, overflows: %d, %w",
			initMsgTimestamp, ErrConflict)
	}
	if ed.Timestamp > expiryAt {
		return fmt.Errorf("cannot message execute at %d, message expired at %d: %w",
			ed.Timestamp, expiryAt, ErrConflict)
	}
	if ed.Timeout == 0 {
		// If no timeout, then just checking the exact execution time was sufficient
		return nil
	}

	// If a timeout is set, check if executing late is still within the expiry window
	if ed.Timestamp+ed.Timeout < ed.Timestamp {
		return fmt.Errorf("message timeout too high, overflows: %d, %w",
			ed.Timestamp, ErrConflict)
	}
	if v := ed.Timestamp + ed.Timeout; v > expiryAt {
		return fmt.Errorf("cannot execute message at timeout %d, expired at %d: %w",
			v, expiryAt, ErrConflict)
	}
	return nil
}

type executingDescriptorMarshaling struct {
	Timestamp hexutil.Uint64 `json:"timestamp"`
	Timeout   hexutil.Uint64 `json:"timeout,omitempty"`
}

func (ed ExecutingDescriptor) MarshalJSON() ([]byte, error) {
	var enc executingDescriptorMarshaling
	enc.Timestamp = hexutil.Uint64(ed.Timestamp)
	enc.Timeout = hexutil.Uint64(ed.Timeout)
	return json.Marshal(&enc)
}

func (ed *ExecutingDescriptor) UnmarshalJSON(input []byte) error {
	var dec executingDescriptorMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	ed.Timestamp = uint64(dec.Timestamp)
	ed.Timeout = uint64(dec.Timeout)
	return nil
}

type ReferenceView struct {
	Local eth.BlockID `json:"local"`
	Cross eth.BlockID `json:"cross"`
}

func (v ReferenceView) String() string {
	return fmt.Sprintf("View(local: %s, cross: %s)", v.Local, v.Cross)
}

type BlockSeal struct {
	Hash      common.Hash
	Number    uint64
	Timestamp uint64
}

func (s BlockSeal) String() string {
	return fmt.Sprintf("BlockSeal(hash:%s, number:%d, time:%d)", s.Hash, s.Number, s.Timestamp)
}

func (s BlockSeal) ID() eth.BlockID {
	return eth.BlockID{Hash: s.Hash, Number: s.Number}
}

func (s BlockSeal) MustWithParent(parent eth.BlockID) eth.BlockRef {
	ref, err := s.WithParent(parent)
	if err != nil {
		panic(err)
	}
	return ref
}

func (s BlockSeal) WithParent(parent eth.BlockID) (eth.BlockRef, error) {
	// prevent parent attachment if the parent is not the previous block,
	// and the block is not the genesis block
	if s.Number != parent.Number+1 && s.Number != 0 {
		return eth.BlockRef{}, fmt.Errorf("%w: parent %s to combine with %s", errInvalidParentBlock, parent, s)
	}
	return eth.BlockRef{
		Hash:       s.Hash,
		Number:     s.Number,
		ParentHash: parent.Hash,
		Time:       s.Timestamp,
	}, nil
}

func (s BlockSeal) ForceWithParent(parent eth.BlockID) eth.BlockRef {
	return eth.BlockRef{
		Hash:       s.Hash,
		Number:     s.Number,
		ParentHash: parent.Hash,
		Time:       s.Timestamp,
	}
}

func BlockSealFromRef(ref eth.BlockRef) BlockSeal {
	return BlockSeal{
		Hash:      ref.Hash,
		Number:    ref.Number,
		Timestamp: ref.Time,
	}
}

// PayloadHashToLogHash converts the payload hash to the log hash
// it is the concatenation of the log's address and the hash of the log's payload,
// which is then hashed again. This is the hash that is stored in the log storage.
// The logHash can then be used to traverse from the executing message
// to the log the referenced initiating message.
func PayloadHashToLogHash(payloadHash common.Hash, addr common.Address) common.Hash {
	msg := make([]byte, 0, 2*common.HashLength)
	msg = append(msg, addr.Bytes()...)
	msg = append(msg, payloadHash.Bytes()...)
	return crypto.Keccak256Hash(msg)
}

// LogToMessagePayload is the data that is hashed to get the payloadHash
// it is the concatenation of the log's topics and data
// the implementation is based on the interop messaging spec
func LogToMessagePayload(l *ethTypes.Log) []byte {
	msg := make([]byte, 0)
	for _, topic := range l.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	msg = append(msg, l.Data...)
	return msg
}

// DerivedBlockRefPair is a pair of block refs, where Derived (L2) is derived from Source (L1).
type DerivedBlockRefPair struct {
	Source  eth.BlockRef `json:"source"`
	Derived eth.BlockRef `json:"derived"`
}

func (refs *DerivedBlockRefPair) IDs() DerivedIDPair {
	return DerivedIDPair{
		Source:  refs.Source.ID(),
		Derived: refs.Derived.ID(),
	}
}

func (refs *DerivedBlockRefPair) Seals() DerivedBlockSealPair {
	return DerivedBlockSealPair{
		Source:  BlockSealFromRef(refs.Source),
		Derived: BlockSealFromRef(refs.Derived),
	}
}

func (refs DerivedBlockRefPair) String() string {
	return fmt.Sprintf("refPair(source: %s, derived: %s)", refs.Source, refs.Derived)
}

// DerivedBlockSealPair is a pair of block seals, where Derived (L2) is derived from Source (L1).
type DerivedBlockSealPair struct {
	Source  BlockSeal `json:"source"`
	Derived BlockSeal `json:"derived"`
}

func (seals *DerivedBlockSealPair) IDs() DerivedIDPair {
	return DerivedIDPair{
		Source:  seals.Source.ID(),
		Derived: seals.Derived.ID(),
	}
}

func (seals DerivedBlockSealPair) String() string {
	return fmt.Sprintf("sealPair(source: %s, derived: %s)", seals.Source, seals.Derived)
}

// DerivedIDPair is a pair of block IDs, where Derived (L2) is derived from Source (L1).
type DerivedIDPair struct {
	Source  eth.BlockID `json:"source"`
	Derived eth.BlockID `json:"derived"`
}

func (ids DerivedIDPair) String() string {
	return fmt.Sprintf("idPair(source: %s, derived: %s)", ids.Source, ids.Derived)
}

type BlockReplacement struct {
	Replacement eth.BlockRef `json:"replacement"`
	Invalidated common.Hash  `json:"invalidated"`
}

// ManagedEvent is an event sent by the managed node to the supervisor,
// to share an update. One of the fields will be non-null; different kinds of updates may be sent.
type ManagedEvent struct {
	Reset                  *string              `json:"reset,omitempty"`
	UnsafeBlock            *eth.BlockRef        `json:"unsafeBlock,omitempty"`
	DerivationUpdate       *DerivedBlockRefPair `json:"derivationUpdate,omitempty"`
	ExhaustL1              *DerivedBlockRefPair `json:"exhaustL1,omitempty"`
	ReplaceBlock           *BlockReplacement    `json:"replaceBlock,omitempty"`
	DerivationOriginUpdate *eth.BlockRef        `json:"derivationOriginUpdate,omitempty"`
}

// MessageChecksum represents a message checksum, as used for access-list checks.
type MessageChecksum common.Hash

func (mc MessageChecksum) MarshalText() ([]byte, error) {
	return common.Hash(mc).MarshalText()
}

func (mc *MessageChecksum) UnmarshalText(data []byte) error {
	return (*common.Hash)(mc).UnmarshalText(data)
}

func (mc MessageChecksum) String() string {
	return common.Hash(mc).String()
}

// Access represents access to a message, parsed from an access-list
type Access struct {
	BlockNumber uint64
	Timestamp   uint64
	LogIndex    uint32
	ChainID     eth.ChainID
	Checksum    MessageChecksum
}

// lookupEntry encodes a lookup entry for an access-list
func (acc Access) lookupEntry() common.Hash {
	var out common.Hash
	out[0] = PrefixLookup
	binary.BigEndian.PutUint64(out[4:12], (*uint256.Int)(&acc.ChainID).Uint64())
	binary.BigEndian.PutUint64(out[12:20], acc.BlockNumber)
	binary.BigEndian.PutUint64(out[20:28], acc.Timestamp)
	binary.BigEndian.PutUint32(out[28:32], acc.LogIndex)
	return out
}

// chainIDExtensionEntry encodes a chainID-extension entry for an access-list
func (acc Access) chainIDExtensionEntry() common.Hash {
	var out common.Hash
	dat := (*uint256.Int)(&acc.ChainID).Bytes32()
	out[0] = PrefixChainIDExtension
	copy(out[8:32], dat[0:24])
	return out
}

type accessMarshaling struct {
	BlockNumber hexutil.Uint64  `json:"blockNumber"`
	Timestamp   hexutil.Uint64  `json:"timestamp"`
	LogIndex    uint32          `json:"logIndex"`
	ChainID     eth.ChainID     `json:"chainID"`
	Checksum    MessageChecksum `json:"checksum"`
}

func (a Access) MarshalJSON() ([]byte, error) {
	enc := accessMarshaling{
		BlockNumber: hexutil.Uint64(a.BlockNumber),
		Timestamp:   hexutil.Uint64(a.Timestamp),
		LogIndex:    a.LogIndex,
		ChainID:     a.ChainID,
		Checksum:    a.Checksum,
	}
	return json.Marshal(&enc)
}

func (a *Access) UnmarshalJSON(input []byte) error {
	var dec accessMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	a.BlockNumber = uint64(dec.BlockNumber)
	a.Timestamp = uint64(dec.Timestamp)
	a.LogIndex = dec.LogIndex
	a.ChainID = dec.ChainID
	a.Checksum = dec.Checksum
	return nil
}

const (
	PrefixLookup           = 1
	PrefixChainIDExtension = 2
	PrefixChecksum         = 3
)

var (
	errExpectedEntry       = errors.New("expected entry")
	errMalformedEntry      = errors.New("malformed entry")
	errUnexpectedEntryType = errors.New("unexpected entry type")
)

// ParseAccess parses some access-list entries into an Access, and returns the remaining entries.
// This process can be repeated until no entries are left, to parse an access-list.
func ParseAccess(entries []common.Hash) ([]common.Hash, Access, error) {
	if len(entries) == 0 {
		return nil, Access{}, errExpectedEntry
	}
	entry := entries[0]
	entries = entries[1:]
	if typeByte := entry[0]; typeByte != PrefixLookup {
		return nil, Access{}, fmt.Errorf("expected lookup, got entry type %d: %w",
			typeByte, errUnexpectedEntryType)
	}
	if ([3]byte)(entry[1:4]) != ([3]byte{}) {
		return nil, Access{}, fmt.Errorf("expected zero bytes: %w", errMalformedEntry)
	}
	var access Access
	access.ChainID = eth.ChainIDFromUInt64(binary.BigEndian.Uint64(entry[4:12]))
	access.BlockNumber = binary.BigEndian.Uint64(entry[12:20])
	access.Timestamp = binary.BigEndian.Uint64(entry[20:28])
	access.LogIndex = binary.BigEndian.Uint32(entry[28:32])

	if len(entries) == 0 {
		return nil, Access{}, errExpectedEntry
	}
	entry = entries[0]
	entries = entries[1:]
	if typeByte := entry[0]; typeByte == PrefixChainIDExtension {
		if ([7]byte)(entry[1:8]) != ([7]byte{}) {
			return nil, Access{}, fmt.Errorf("expected zero bytes: %w", errMalformedEntry)
		}
		// The lower 8 bytes is set to the uint64 in the first entry.
		// The upper 24 bytes are set with this extension entry.
		chIDBytes32 := access.ChainID.Bytes32()
		copy(chIDBytes32[0:24], entry[8:32])
		access.ChainID = eth.ChainIDFromBytes32(chIDBytes32)
		if len(entries) == 0 {
			return nil, Access{}, errExpectedEntry
		}
		entry = entries[0]
		entries = entries[1:]
	}
	if typeByte := entry[0]; typeByte != PrefixChecksum {
		return nil, Access{}, fmt.Errorf("expected checksum, got entry type %d: %w",
			typeByte, errUnexpectedEntryType)
	}
	access.Checksum = MessageChecksum(entry)
	return entries, access, nil
}

func EncodeAccessList(accesses []Access) []common.Hash {
	out := make([]common.Hash, 0, len(accesses)*2)
	for _, acc := range accesses {
		out = append(out, acc.lookupEntry())

		if !(*uint256.Int)(&acc.ChainID).IsUint64() {
			out = append(out, acc.chainIDExtensionEntry())
		}

		if acc.Checksum[0] != PrefixChecksum {
			panic("invalid checksum entry")
		}
		out = append(out, common.Hash(acc.Checksum))
	}
	return out
}
