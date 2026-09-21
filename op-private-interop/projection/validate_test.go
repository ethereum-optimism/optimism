package projection_test

import (
	"encoding/json"
	"errors"
	"flag"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var updateVectors = flag.Bool("update-projection-vectors", false, "regenerate shared Go/Kona projection admission vectors")

type block struct {
	Timestamp    uint64          `json:"timestamp"`
	Epoch        uint64          `json:"epoch"`
	Transactions []hexutil.Bytes `json:"transactions"`
}
type span []block

func (s span) GetBlockCount() int                         { return len(s) }
func (s span) GetBlockTimestamp(i int) uint64             { return s[i].Timestamp }
func (s span) GetBlockEpochNum(i int) uint64              { return s[i].Epoch }
func (s span) GetBlockTransactions(i int) []hexutil.Bytes { return s[i].Transactions }

type vector struct {
	Name   string            `json:"name"`
	Config projection.Config `json:"config"`
	Blocks span              `json:"blocks"`
	Accept bool              `json:"accept"`
	Digest common.Hash       `json:"digest"`
}

func context() projection.Context {
	return projection.Context{ChainID: big.NewInt(901), GenesisTime: 1000, BlockTime: 2, ParentHash: common.Hash{1}}
}
func signed(t *testing.T, tx *types.DynamicFeeTx) hexutil.Bytes {
	t.Helper()
	key, err := crypto.HexToECDSA(strings.Repeat("11", 32))
	require.NoError(t, err)
	out, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(901)), tx)
	require.NoError(t, err)
	raw, err := out.MarshalBinary()
	require.NoError(t, err)
	return raw
}
func transaction(t *testing.T, to common.Address, data []byte, al types.AccessList) hexutil.Bytes {
	return signed(t, &types.DynamicFeeTx{ChainID: big.NewInt(901), Gas: 500000, GasFeeCap: new(big.Int), GasTipCap: new(big.Int), Value: new(big.Int), To: &to, Data: data, AccessList: al})
}
func claim(t *testing.T, first, last uint64, proof []byte) hexutil.Bytes {
	data, err := wire.EncodePostClaim(&codec.RangeClaim{FirstBlock: first, LastBlock: last, Proof: proof, PrivateTerminalBlockHash: common.Hash{2}, PrivateTerminalParentHash: common.Hash{3}})
	require.NoError(t, err)
	return transaction(t, predeploys.ClaimRegistryAddr, data, nil)
}
func export(t *testing.T) hexutil.Bytes {
	data, err := wire.EncodeReplaySentMessage(&wire.SentMessage{Destination: big.NewInt(902), Nonce: big.NewInt(1), Sender: common.Address{1}, Target: common.Address{2}, Message: []byte("hello")})
	require.NoError(t, err)
	return transaction(t, predeploys.L2toL2CrossDomainMessengerAddr, data, nil)
}
func imported(t *testing.T, wide bool) hexutil.Bytes {
	chain := big.NewInt(902)
	if wide {
		chain.Lsh(chain, 80)
	}
	trigger := &txintent.ExecTrigger{Msg: messages.Message{Identifier: messages.Identifier{Origin: common.Address{9}, BlockNumber: 4, LogIndex: 2, Timestamp: 1002, ChainID: eth.ChainIDFromBig(chain)}, PayloadHash: common.Hash{7}}}
	data, err := trigger.EncodeInput()
	require.NoError(t, err)
	al, err := trigger.AccessList()
	require.NoError(t, err)
	return transaction(t, predeploys.CrossL2InboxAddr, data, al)
}
func mutate(t *testing.T, raw hexutil.Bytes, f func(*types.DynamicFeeTx)) hexutil.Bytes {
	var tx types.Transaction
	require.NoError(t, tx.UnmarshalBinary(raw))
	fields := &types.DynamicFeeTx{ChainID: tx.ChainId(), Nonce: tx.Nonce(), Gas: tx.Gas(), GasFeeCap: tx.GasFeeCap(), GasTipCap: tx.GasTipCap(), To: tx.To(), Value: tx.Value(), Data: tx.Data(), AccessList: tx.AccessList()}
	f(fields)
	return signed(t, fields)
}
func vectors(t *testing.T) []vector {
	var out []vector
	add := func(name string, accept bool, change func(*vector)) {
		v := vector{Name: name, Accept: accept, Config: projection.Config{Verifier: projection.InsecureStub}, Blocks: span{{1020, 5, []hexutil.Bytes{claim(t, 10, 12, []byte("dummy"))}}, {1022, 5, nil}, {1024, 6, []hexutil.Bytes{export(t), imported(t, false)}}}}
		if change != nil {
			change(&v)
		}
		out = append(out, v)
	}
	add("mixed", true, nil)
	add("empty_messages", true, func(v *vector) { v.Blocks[2].Transactions = nil })
	add("empty_proof", true, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 12, nil) })
	add("wide_chain_id", true, func(v *vector) { v.Blocks[2].Transactions[1] = imported(t, true) })
	add("missing_claim", false, func(v *vector) { v.Blocks[0].Transactions = nil })
	add("misplaced_claim", false, func(v *vector) {
		v.Blocks[0].Transactions = append([]hexutil.Bytes{export(t)}, v.Blocks[0].Transactions...)
	})
	add("duplicate_claim", false, func(v *vector) { v.Blocks[2].Transactions = append(v.Blocks[2].Transactions, claim(t, 10, 12, nil)) })
	add("wrong_first", false, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 9, 12, nil) })
	add("wrong_last", false, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 13, nil) })
	add("truncated_span", false, func(v *vector) { v.Blocks = v.Blocks[:2] })
	add("noncontiguous", false, func(v *vector) { v.Blocks[2].Timestamp++ })
	add("unknown_mode", false, func(v *vector) { v.Config.Verifier = "unknown" })
	add("max_proof", true, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 12, make([]byte, codec.MaxProofSize)) })
	changes := []struct {
		name string
		f    func(*types.DynamicFeeTx)
	}{
		{"creation", func(tx *types.DynamicFeeTx) { tx.To = nil }},
		{"value", func(tx *types.DynamicFeeTx) { tx.Value = big.NewInt(1) }},
		{"fee", func(tx *types.DynamicFeeTx) { tx.GasFeeCap = big.NewInt(1) }},
		{"tip", func(tx *types.DynamicFeeTx) { tx.GasTipCap = big.NewInt(1) }},
		{"gas_zero", func(tx *types.DynamicFeeTx) { tx.Gas = 0 }},
		{"gas_overflow", func(tx *types.DynamicFeeTx) { tx.Gas = projection.MaxTxGas + 1 }},
		{"destination", func(tx *types.DynamicFeeTx) { a := common.Address{9}; tx.To = &a }},
		{"selector", func(tx *types.DynamicFeeTx) { tx.Data[0] ^= 1 }},
		{"trailing_bytes", func(tx *types.DynamicFeeTx) { tx.Data = append(tx.Data, 0) }},
		{"dirty_address", func(tx *types.DynamicFeeTx) { tx.Data[4+64] = 1 }},
		{"bad_offset", func(tx *types.DynamicFeeTx) { tx.Data[4+159] = 0 }},
		{"extra_access_list", func(tx *types.DynamicFeeTx) { tx.AccessList = types.AccessList{{Address: common.Address{1}}} }},
	}
	for _, change := range changes {
		add("late_"+change.name, false, func(v *vector) { v.Blocks[2].Transactions[0] = mutate(t, v.Blocks[2].Transactions[0], change.f) })
	}
	add("bad_import_checksum", false, func(v *vector) {
		v.Blocks[2].Transactions[1] = mutate(t, v.Blocks[2].Transactions[1], func(tx *types.DynamicFeeTx) { tx.AccessList[0].StorageKeys[1][5] ^= 1 })
	})
	add("duplicate_import_access", false, func(v *vector) {
		v.Blocks[2].Transactions[1] = mutate(t, v.Blocks[2].Transactions[1], func(tx *types.DynamicFeeTx) { tx.AccessList = append(tx.AccessList, tx.AccessList[0]) })
	})
	add("legacy_transaction", false, func(v *vector) {
		key, err := crypto.HexToECDSA(strings.Repeat("11", 32))
		require.NoError(t, err)
		a := predeploys.L2toL2CrossDomainMessengerAddr
		tx, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(901)), &types.LegacyTx{To: &a, Gas: 500000, GasPrice: new(big.Int), Value: new(big.Int)})
		require.NoError(t, err)
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		v.Blocks[2].Transactions[0] = raw
	})
	add("empty_transaction", false, func(v *vector) { v.Blocks[2].Transactions[0] = nil })
	for _, enabled := range []bool{false, true} {
		add(map[bool]string{false: "events_disabled", true: "events_enabled"}[enabled], enabled, func(v *vector) {
			v.Config.AllowEvents = enabled
			data, err := wire.EncodeReplayEvent([]common.Hash{{4}}, []byte("event"))
			require.NoError(t, err)
			v.Blocks[2].Transactions[0] = transaction(t, predeploys.EventReplayerAddr, data, nil)
		})
	}
	return out
}

func TestProjectionVectors(t *testing.T) {
	if *updateVectors {
		cases := vectors(t)
		for i := range cases {
			v := &cases[i]
			result, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
			if v.Accept {
				require.NoError(t, err, v.Name)
				v.Digest = result.ProjectionHash
			} else {
				require.Error(t, err, v.Name)
			}
		}
		raw, err := json.MarshalIndent(cases, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll("testdata", 0755))
		require.NoError(t, os.WriteFile("testdata/ranges.json", append(raw, '\n'), 0644))
	}
	raw, err := os.ReadFile("testdata/ranges.json")
	require.NoError(t, err)
	var cases []vector
	require.NoError(t, json.Unmarshal(raw, &cases))
	for _, v := range cases {
		t.Run(v.Name, func(t *testing.T) {
			before, err := json.Marshal(v)
			require.NoError(t, err)
			ctx := context()
			first, err := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
			if v.Accept {
				require.NoError(t, err)
				require.Equal(t, v.Digest, first.ProjectionHash)
			} else {
				require.Error(t, err)
			}
			second, again := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
			require.Equal(t, first, second)
			require.Equal(t, err, again)
			after, e := json.Marshal(v)
			require.NoError(t, e)
			require.Equal(t, before, after)
			require.Equal(t, context(), ctx)
		})
	}
}

type rejectVerifier struct{}

func (rejectVerifier) Verify(projection.Statement, []byte) error {
	return errors.New("deliberate proof rejection")
}

type bindingVerifier struct {
	want  projection.Statement
	proof []byte
}

func (v bindingVerifier) Verify(got projection.Statement, proof []byte) error {
	a, _ := json.Marshal(v.want)
	b, _ := json.Marshal(got)
	if string(a) != string(b) || string(proof) != string(v.proof) {
		return errors.New("wrong statement or proof")
	}
	return nil
}
func TestProofGate(t *testing.T) {
	v := vectors(t)[0]
	statement, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	require.Empty(t, statement.Claim.Proof)
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, rejectVerifier{})
	require.ErrorContains(t, err, "deliberate proof rejection")
	checker := bindingVerifier{*statement, []byte("dummy")}
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, checker)
	require.NoError(t, err)
	v.Blocks[2].Transactions = nil
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, checker)
	require.ErrorContains(t, err, "wrong statement")
}

func TestProofStatementBindsExecutionEnvelopeAndContext(t *testing.T) {
	v := vectors(t)[0]
	want, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	verifier := bindingVerifier{*want, []byte("dummy")}
	for _, field := range []string{"nonce", "gas", "parent"} {
		t.Run(field, func(t *testing.T) {
			candidate := vectors(t)[0]
			ctx := context()
			if field == "parent" {
				ctx.ParentHash[1]++
			} else {
				candidate.Blocks[2].Transactions[0] = mutate(t, candidate.Blocks[2].Transactions[0], func(tx *types.DynamicFeeTx) {
					if field == "nonce" {
						tx.Nonce++
					} else {
						tx.Gas++
					}
				})
			}
			_, err := projection.ValidateProjectionRange(&candidate.Config, ctx, candidate.Blocks, verifier)
			require.ErrorContains(t, err, "wrong statement")
		})
	}
}

func TestProjectionContextBounds(t *testing.T) {
	for _, field := range []string{"zero_chain", "zero_interval", "height_overflow", "before_genesis", "empty_range"} {
		t.Run(field, func(t *testing.T) {
			v, ctx := vectors(t)[0], context()
			switch field {
			case "zero_chain":
				ctx.ChainID = new(big.Int)
			case "zero_interval":
				ctx.BlockTime = 0
			case "height_overflow":
				ctx.GenesisNumber = ^uint64(0)
			case "before_genesis":
				ctx.GenesisTime = 1021
			case "empty_range":
				v.Blocks = nil
			}
			_, err := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
			require.Error(t, err)
		})
	}
}
