package projection_test

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// commitments.json (§F.5): the byte rules of §B.2, §C.2 and §C.3 as Go computes them.

type commitmentProofVector struct {
	Index    uint64        `json:"index"`
	Leaf     common.Hash   `json:"leaf"`
	Siblings []common.Hash `json:"siblings"`
}
type badCommitmentProofVector struct {
	commitmentProofVector
	Reason string `json:"reason"`
}
type treeVector struct {
	Domain    hexutil.Bytes              `json:"domain"`
	N         uint64                     `json:"n"`
	Leaves    []common.Hash              `json:"leaves"`
	Root      common.Hash                `json:"root"`
	Proofs    []commitmentProofVector    `json:"proofs"`
	BadProofs []badCommitmentProofVector `json:"bad_proofs"`
}
type outputLeafVector struct {
	BlockNumber uint64      `json:"block_number"`
	OutputRoot  common.Hash `json:"output_root"`
	Leaf        common.Hash `json:"leaf"`
}
type logVector struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}
type messageLeafVector struct {
	Name          string         `json:"name"`
	BlockNumber   uint64         `json:"block_number"`
	RenderedIndex uint32         `json:"rendered_index"`
	Kind          uint8          `json:"kind"`
	To            common.Address `json:"to"`
	Calldata      hexutil.Bytes  `json:"calldata"`
	Log           logVector      `json:"log"`
	MessageHash   common.Hash    `json:"message_hash"`
	Leaf          common.Hash    `json:"leaf"`
}
type configHashVector struct {
	Name    string            `json:"name"`
	Config  projection.Config `json:"config"`
	Context vectorContext     `json:"context"`
	Hash    common.Hash       `json:"hash"`
}
type privateConfigHashVector struct {
	PrivateRollupJSON hexutil.Bytes `json:"private_rollup_json"`
	L1ChainConfigJSON hexutil.Bytes `json:"l1_chain_config_json"`
	Hash              common.Hash   `json:"hash"`
}
type depSetHashVector struct {
	ChainIDs []eth.ChainID `json:"chain_ids"`
	Hash     common.Hash   `json:"hash"`
}
type statementVector struct {
	ChainID                   common.Hash `json:"chain_id"`
	ProjectionConfigHash      common.Hash `json:"projection_config_hash"`
	PrivateConfigHash         common.Hash `json:"private_config_hash"`
	DepSetHash                common.Hash `json:"dep_set_hash"`
	ParentHash                common.Hash `json:"parent_hash"`
	AnchorNumber              uint64      `json:"anchor_number"`
	AnchorHash                common.Hash `json:"anchor_hash"`
	AnchorOutputRoot          common.Hash `json:"anchor_output_root"`
	RecoveryHash              common.Hash `json:"recovery_hash"`
	FirstBlock                uint64      `json:"first_block"`
	LastBlock                 uint64      `json:"last_block"`
	ParentOutputRoot          common.Hash `json:"parent_output_root"`
	PrivateTerminalBlockHash  common.Hash `json:"private_terminal_block_hash"`
	PrivateTerminalParentHash common.Hash `json:"private_terminal_parent_hash"`
	L1Head                    common.Hash `json:"l1_head"`
	PrivateDataHash           common.Hash `json:"private_data_hash"`
	ProjectionHash            common.Hash `json:"projection_hash"`
	OutputsRoot               common.Hash `json:"outputs_root"`
	MessagesRoot              common.Hash `json:"messages_root"`
	TerminalOutput            common.Hash `json:"terminal_output"`
}
type publicValuesVector struct {
	Statement    statementVector `json:"statement"`
	PublicValues hexutil.Bytes   `json:"public_values"`
}
type constantsVector struct {
	PublicValuesMagic   common.Hash   `json:"public_values_magic"`
	ConfigDomain        hexutil.Bytes `json:"config_domain"`
	PrivateConfigDomain hexutil.Bytes `json:"private_config_domain"`
	DepSetDomain        hexutil.Bytes `json:"dep_set_domain"`
	OutputsDomain       hexutil.Bytes `json:"outputs_domain"`
	MessagesDomain      hexutil.Bytes `json:"messages_domain"`
}
type commitmentVectors struct {
	Constants           constantsVector         `json:"constants"`
	Trees               []treeVector            `json:"trees"`
	OutputLeaf          outputLeafVector        `json:"output_leaf"`
	MessageLeaves       []messageLeafVector     `json:"message_leaves"`
	ConfigHashes        []configHashVector      `json:"config_hashes"`
	PrivateConfigHash   privateConfigHashVector `json:"private_config_hash"`
	DependencySetHashes []depSetHashVector      `json:"dependency_set_hashes"`
	PublicValues        publicValuesVector      `json:"public_values"`
}

func indexLeaves(n int) []common.Hash {
	leaves := make([]common.Hash, n)
	for i := range leaves {
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(i))
		leaves[i] = crypto.Keccak256Hash(b[:])
	}
	return leaves
}

func tree(t *testing.T, domain string, n int) treeVector {
	leaves := indexLeaves(n)
	// Slices are non-nil so the JSON carries [] rather than null.
	tv := treeVector{Domain: []byte(domain), N: uint64(n), Leaves: leaves, Root: projection.CommitmentRoot(domain, leaves), Proofs: []commitmentProofVector{}}
	for i := range leaves {
		siblings, err := projection.CommitmentProof(leaves, i)
		require.NoError(t, err)
		if siblings == nil {
			siblings = []common.Hash{}
		}
		tv.Proofs = append(tv.Proofs, commitmentProofVector{uint64(i), leaves[i], siblings})
	}
	if n == 0 {
		tv.BadProofs = append(tv.BadProofs, badCommitmentProofVector{commitmentProofVector{0, indexLeaves(1)[0], []common.Hash{}}, "empty tree has no members"})
		return tv
	}
	last := tv.Proofs[n-1]
	extra := badCommitmentProofVector{commitmentProofVector{last.Index, last.Leaf, append(append([]common.Hash(nil), last.Siblings...), common.Hash{1})}, "unconsumed sibling"}
	tv.BadProofs = append(tv.BadProofs, extra)
	if len(tv.Proofs[0].Siblings) > 0 {
		flipped := append([]common.Hash(nil), tv.Proofs[0].Siblings...)
		flipped[len(flipped)-1][0] ^= 1
		tv.BadProofs = append(tv.BadProofs, badCommitmentProofVector{commitmentProofVector{0, leaves[0], flipped}, "flipped sibling"})
	}
	wrongLeaf := tv.Proofs[0]
	wrongLeaf.Leaf[31] ^= 1
	tv.BadProofs = append(tv.BadProofs, badCommitmentProofVector{wrongLeaf, "wrong leaf"})
	tv.BadProofs = append(tv.BadProofs, badCommitmentProofVector{commitmentProofVector{uint64(n), leaves[0], tv.Proofs[0].Siblings}, "index out of range"})
	return tv
}

// sampleMessages returns an export and an import with both their replay calldata and the private
// log that renders to it.
func sampleMessages(t *testing.T) (exportData []byte, exportLog *types.Log, importData []byte, importLog *types.Log) {
	sent := &wire.SentMessage{Destination: big.NewInt(902), Nonce: big.NewInt(7), Sender: common.Address{0xaa}, Target: common.Address{0xbb}, Message: []byte("a message that spans more than one word of data")}
	exportData, err := wire.EncodeReplaySentMessage(sent)
	require.NoError(t, err)
	topics, data := wire.SentMessageLog(sent)
	exportLog = &types.Log{Address: predeploys.L2toL2CrossDomainMessengerAddr, Topics: topics, Data: data}

	msg := messages.Message{Identifier: messages.Identifier{Origin: common.Address{9}, BlockNumber: 4, LogIndex: 2, Timestamp: 1002, ChainID: eth.ChainIDFromUInt64(902)}, PayloadHash: common.Hash{7, 7}}
	importData, err = (&txintent.ExecTrigger{Msg: msg}).EncodeInput()
	require.NoError(t, err)
	// ExecutingMessage(bytes32 indexed msgHash, Identifier id): data is the 160-byte identifier.
	id := make([]byte, 160)
	copy(id[12:32], msg.Identifier.Origin[:])
	binary.BigEndian.PutUint64(id[56:64], msg.Identifier.BlockNumber)
	binary.BigEndian.PutUint64(id[88:96], uint64(msg.Identifier.LogIndex))
	binary.BigEndian.PutUint64(id[120:128], msg.Identifier.Timestamp)
	chainID := msg.Identifier.ChainID.Bytes32()
	copy(id[128:160], chainID[:])
	importLog = &types.Log{Address: predeploys.CrossL2InboxAddr, Topics: []common.Hash{messages.ExecutingMessageEventTopic, msg.PayloadHash}, Data: id}
	decoded, err := messages.MessageFromLog(importLog)
	require.NoError(t, err)
	require.Equal(t, &msg, decoded)
	return
}

func commitmentVectorsFor(t *testing.T) commitmentVectors {
	var out commitmentVectors
	out.Constants = constantsVector{
		PublicValuesMagic: projection.PublicValuesMagic, ConfigDomain: []byte(projection.ConfigDomain),
		PrivateConfigDomain: []byte(projection.PrivateConfigDomain), DepSetDomain: []byte(projection.DepSetDomain),
		OutputsDomain: []byte(projection.OutputsDomain), MessagesDomain: []byte(projection.MessagesDomain),
	}
	for _, n := range []int{0, 1, 2, 3, 4, 5, 7, 8} {
		out.Trees = append(out.Trees, tree(t, projection.OutputsDomain, n))
	}
	out.Trees = append(out.Trees, tree(t, projection.MessagesDomain, 0), tree(t, projection.MessagesDomain, 3))
	out.OutputLeaf = outputLeafVector{BlockNumber: 12, OutputRoot: common.Hash{0x22}, Leaf: projection.OutputLeaf(12, common.Hash{0x22})}

	exportData, exportLog, importData, importLog := sampleMessages(t)
	m, err := wire.DecodeReplaySentMessage(exportData)
	require.NoError(t, err)
	initHash := projection.ExportMessageHash(m)
	execHash := projection.ImportMessageHash([192]byte(importData[4:196]))
	out.MessageLeaves = []messageLeafVector{
		{Name: "init", BlockNumber: 12, RenderedIndex: 0, Kind: projection.MessageKindInit, To: predeploys.L2toL2CrossDomainMessengerAddr, Calldata: exportData,
			Log: logVector{exportLog.Address, exportLog.Topics, exportLog.Data}, MessageHash: initHash, Leaf: projection.MessageLeaf(12, 0, projection.MessageKindInit, initHash)},
		{Name: "exec", BlockNumber: 12, RenderedIndex: 1, Kind: projection.MessageKindExec, To: predeploys.CrossL2InboxAddr, Calldata: importData,
			Log: logVector{importLog.Address, importLog.Topics, importLog.Data}, MessageHash: execHash, Leaf: projection.MessageLeaf(12, 1, projection.MessageKindExec, execHash)},
	}
	out.ConfigHashes = append(out.ConfigHashes,
		configHashVector{"execution_mock", mockConfig(), toVectorContext(context()), projection.ConfigHash(&[]projection.Config{mockConfig()}[0], context())},
		configHashVector{"sp1", sp1Config(false), toVectorContext(context()), projection.ConfigHash(&[]projection.Config{sp1Config(false)}[0], context())},
	)
	events := sp1Config(true)
	events.AllowEvents = true
	geometry := context()
	geometry.ChainID, geometry.GenesisNumber, geometry.GenesisTime, geometry.BlockTime = big.NewInt(902), 5, 7, 1
	out.ConfigHashes = append(out.ConfigHashes, configHashVector{"sp1_mock_events_geometry", events, toVectorContext(geometry), projection.ConfigHash(&events, geometry)})

	out.PrivateConfigHash = privateConfigHashVector{vectorPrivateRollupJSON, vectorL1ChainConfigJSON, projection.PrivateConfigHash(vectorPrivateRollupJSON, vectorL1ChainConfigJSON)}
	wide := eth.ChainIDFromBig(new(big.Int).Lsh(big.NewInt(1), 200))
	for _, ids := range [][]eth.ChainID{
		{},
		vectorDepSet,
		{eth.ChainIDFromUInt64(902), eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(901)},
		{wide, eth.ChainIDFromUInt64(10), eth.ChainIDFromUInt64(901)},
	} {
		out.DependencySetHashes = append(out.DependencySetHashes, depSetHashVector{ids, projection.DependencySetHash(ids)})
	}

	// A full statement with every word distinct.
	word := func(i byte) common.Hash { return crypto.Keccak256Hash([]byte{i}) }
	sv := statementVector{
		ChainID: common.BigToHash(big.NewInt(901)), ProjectionConfigHash: word(2), PrivateConfigHash: word(3), DepSetHash: word(4),
		ParentHash: word(5), AnchorNumber: 0x0102030405060708, AnchorHash: word(7), AnchorOutputRoot: word(8), RecoveryHash: word(9),
		FirstBlock: 0x1112131415161718, LastBlock: 0x2122232425262728, ParentOutputRoot: word(12), PrivateTerminalBlockHash: word(13),
		PrivateTerminalParentHash: word(14), L1Head: word(15), PrivateDataHash: word(16), ProjectionHash: word(17),
		OutputsRoot: word(18), MessagesRoot: word(19), TerminalOutput: word(20),
	}
	pv := projection.PublicValues(statementFrom(sv))
	out.PublicValues = publicValuesVector{sv, pv[:]}
	return out
}

func statementFrom(v statementVector) *projection.Statement {
	s := &projection.Statement{
		ChainID: v.ChainID, ParentHash: v.ParentHash, ProjectionHash: v.ProjectionHash,
		Continuation:         projection.Continuation{Anchor: eth.BlockID{Number: v.AnchorNumber, Hash: v.AnchorHash}, OutputRoot: v.AnchorOutputRoot, RecoveryHash: v.RecoveryHash},
		ProjectionConfigHash: v.ProjectionConfigHash, PrivateConfigHash: v.PrivateConfigHash,
		OutputsRoot: v.OutputsRoot, MessagesRoot: v.MessagesRoot, TerminalOutput: v.TerminalOutput,
	}
	s.Claim.DepSetHash, s.Claim.FirstBlock, s.Claim.LastBlock = v.DepSetHash, v.FirstBlock, v.LastBlock
	s.Claim.ParentOutputRoot, s.Claim.PrivateTerminalBlockHash, s.Claim.PrivateTerminalParentHash = v.ParentOutputRoot, v.PrivateTerminalBlockHash, v.PrivateTerminalParentHash
	s.Claim.L1Head, s.Claim.PrivateDataHash = v.L1Head, v.PrivateDataHash
	return s
}

func TestCommitmentVectors(t *testing.T) {
	generated := commitmentVectorsFor(t)
	syncVectors(t, "testdata/commitments.json", generated)
	for _, tv := range generated.Trees {
		domain := string(tv.Domain)
		for _, p := range tv.Proofs {
			require.True(t, projection.VerifyCommitmentProof(domain, tv.N, p.Index, p.Leaf, p.Siblings, tv.Root), "n=%d index=%d", tv.N, p.Index)
		}
		require.NotEmpty(t, tv.BadProofs)
		for _, p := range tv.BadProofs {
			require.False(t, projection.VerifyCommitmentProof(domain, tv.N, p.Index, p.Leaf, p.Siblings, tv.Root), "n=%d %s", tv.N, p.Reason)
		}
	}
	// Both paths to a message leaf, calldata and private log, give the same hash.
	for _, m := range generated.MessageLeaves {
		l := &types.Log{Address: m.Log.Address, Topics: m.Log.Topics, Data: m.Log.Data}
		leaves, err := render.MessageLeaves(m.BlockNumber, []render.RenderedLog{{Log: l, RenderedLogIndex: m.RenderedIndex}})
		require.NoError(t, err)
		require.Equal(t, []common.Hash{m.Leaf}, leaves, m.Name)
		if m.Kind == projection.MessageKindInit {
			require.Equal(t, m.MessageHash, crypto.Keccak256Hash(messages.LogToMessagePayload(l)), m.Name)
		} else {
			require.Equal(t, m.MessageHash, crypto.Keccak256Hash(l.Data, l.Topics[1][:]), m.Name)
		}
	}
}

func TestCommitmentRootShape(t *testing.T) {
	require.Equal(t, crypto.Keccak256Hash([]byte(projection.OutputsDomain), make([]byte, 8), make([]byte, 32)), projection.CommitmentRoot(projection.OutputsDomain, nil))
	leaves := indexLeaves(3)
	n01 := crypto.Keccak256Hash([]byte{1}, leaves[0][:], leaves[1][:])
	n22 := crypto.Keccak256Hash([]byte{1}, leaves[2][:], leaves[2][:])
	top := crypto.Keccak256Hash([]byte{1}, n01[:], n22[:])
	require.Equal(t, crypto.Keccak256Hash([]byte(projection.MessagesDomain), []byte{0, 0, 0, 0, 0, 0, 0, 3}, top[:]), projection.CommitmentRoot(projection.MessagesDomain, leaves))
	// The records root is the same tree under its own domain.
	require.Equal(t, projection.RecordsRoot(leaves), projection.CommitmentRoot("optimism.private-projection.v2\x00", leaves))
	// Domain separation.
	require.NotEqual(t, projection.CommitmentRoot(projection.OutputsDomain, leaves), projection.CommitmentRoot(projection.MessagesDomain, leaves))
	// Odd-level duplication cannot alias lengths.
	require.NotEqual(t, projection.CommitmentRoot(projection.OutputsDomain, leaves), projection.CommitmentRoot(projection.OutputsDomain, append(leaves, leaves[2])))
	// Index 2 of 3 is the last node of an odd level: no sibling is emitted there.
	siblings, err := projection.CommitmentProof(leaves, 2)
	require.NoError(t, err)
	require.Equal(t, []common.Hash{n01}, siblings)
	_, err = projection.CommitmentProof(leaves, 3)
	require.Error(t, err)
	_, err = projection.CommitmentProof(leaves, -1)
	require.Error(t, err)
	root := projection.CommitmentRoot(projection.OutputsDomain, leaves)
	require.False(t, projection.VerifyCommitmentProof(projection.OutputsDomain, 4, 2, leaves[2], siblings, root), "wrong count")
	require.False(t, projection.VerifyCommitmentProof(projection.MessagesDomain, 3, 2, leaves[2], siblings, root), "wrong domain")
	require.False(t, projection.VerifyCommitmentProof(projection.OutputsDomain, 3, 2, leaves[2], nil, root), "missing sibling")
}

func TestLeafEncodings(t *testing.T) {
	root := common.Hash{0x22}
	require.Equal(t, crypto.Keccak256Hash([]byte{0}, []byte{0, 0, 0, 0, 0, 0, 0, 12}, root[:]), projection.OutputLeaf(12, root))
	h := common.Hash{0x33}
	require.Equal(t, crypto.Keccak256Hash([]byte{0}, []byte{0, 0, 0, 0, 0, 0, 0, 12}, []byte{0, 0, 0, 5}, []byte{2}, h[:]), projection.MessageLeaf(12, 5, projection.MessageKindExec, h))
	require.NotEqual(t, projection.MessageLeaf(12, 5, projection.MessageKindInit, h), projection.MessageLeaf(12, 5, projection.MessageKindExec, h))

	exportData, exportLog, importData, importLog := sampleMessages(t)
	m, err := wire.DecodeReplaySentMessage(exportData)
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(messages.LogToMessagePayload(exportLog)), projection.ExportMessageHash(m))
	// The import hash is keccak(ExecutingMessage data ‖ msgHash topic).
	require.Equal(t, crypto.Keccak256Hash(importLog.Data, importLog.Topics[1][:]), projection.ImportMessageHash([192]byte(importData[4:196])))
	// Wide chain IDs and wide destinations are hashed as full words.
	wide := &wire.SentMessage{Destination: new(big.Int).Lsh(big.NewInt(1), 255), Nonce: new(big.Int).Lsh(big.NewInt(3), 200), Target: common.Address{1}}
	topics, _ := wire.SentMessageLog(wide)
	require.Equal(t, common.BigToHash(wide.Destination), topics[1])
	require.Equal(t, common.BigToHash(wide.Nonce), topics[3])
}
