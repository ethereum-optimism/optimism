package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func FuzzRoundtripIdentifierJSONMarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, origin []byte, blockNumber uint64, logIndex uint32, timestamp uint64, chainID []byte) {
		if len(chainID) > 32 {
			chainID = chainID[:32]
		}

		id := Identifier{
			Origin:      common.BytesToAddress(origin),
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   timestamp,
			ChainID:     eth.ChainIDFromBig(new(big.Int).SetBytes(chainID)),
		}

		raw, err := json.Marshal(&id)
		require.NoError(t, err)

		var dec Identifier
		require.NoError(t, json.Unmarshal(raw, &dec))

		require.Equal(t, id.Origin, dec.Origin)
		require.Equal(t, id.BlockNumber, dec.BlockNumber)
		require.Equal(t, id.LogIndex, dec.LogIndex)
		require.Equal(t, id.Timestamp, dec.Timestamp)
		require.Equal(t, id.ChainID, dec.ChainID)
	})
}

func TestChainIndex(t *testing.T) {
	var x ChainIndex
	require.NoError(t, json.Unmarshal([]byte(`"1"`), &x))
	require.Equal(t, ChainIndex(1), x)
	data, err := json.Marshal(x)
	require.NoError(t, err)
	require.Equal(t, `"1"`, string(data))

	require.NoError(t, json.Unmarshal([]byte(`"4294967295"`), &x))
	require.Equal(t, ChainIndex(0xff_ff_ff_ff), x)
	data, err = json.Marshal(x)
	require.NoError(t, err)
	require.Equal(t, `"4294967295"`, string(data))

	require.ErrorContains(t, json.Unmarshal([]byte(`"-1"`), &x), "invalid")
	require.ErrorContains(t, json.Unmarshal([]byte(`"4294967296"`), &x), "out of range")
}

func TestHashing(t *testing.T) {
	keccak256 := func(name string, parts ...[]byte) (h common.Hash) {
		t.Logf("%s = H(", name)
		for _, p := range parts {
			t.Logf("  %x,", p)
		}
		t.Logf(")")
		h = crypto.Keccak256Hash(parts...)
		t.Logf("%s = %s", name, h)
		return h
	}
	id := Identifier{
		Origin:      common.HexToAddress("0xe0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9"),
		BlockNumber: 0xa1a2_a3a4_a5a6_a7a8,
		LogIndex:    0xb1b2_b3b4,
		Timestamp:   0xc1c2_c3c4_c5c6_c7c8,
		ChainID:     eth.ChainIDFromUInt64(0xd1d2_d3d4_d5d6_d7d8),
	}
	payloadHash := keccak256("payloadHash", []byte("example payload")) // aka msgHash
	logHash := keccak256("logHash", id.Origin[:], payloadHash[:])
	x := PayloadHashToLogHash(payloadHash, id.Origin)
	require.Equal(t, logHash, x, "check op-supervisor version of log-hashing matches intermediate value")

	var idPacked []byte
	idPacked = append(idPacked, make([]byte, 12)...)
	idPacked = binary.BigEndian.AppendUint64(idPacked, id.BlockNumber)
	idPacked = binary.BigEndian.AppendUint64(idPacked, id.Timestamp)
	idPacked = binary.BigEndian.AppendUint32(idPacked, id.LogIndex)
	t.Logf("idPacked: %x", idPacked)

	idLogHash := keccak256("idLogHash", logHash[:], idPacked)
	chainID := id.ChainID.Bytes32()
	bareChecksum := keccak256("bareChecksum", idLogHash[:], chainID[:])

	checksum := bareChecksum
	checksum[0] = 0x03
	t.Logf("Checksum: %s", checksum)
}

var (
	testOrigin      = common.HexToAddress("0xe0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9")
	testBlockNumber = uint64(0xa1a2_a3a4_a5a6_a7a8)
	testLogIndex    = uint32(0xb1b2_b3b4)
	testTimestamp   = uint64(0xc1c2_c3c4_c5c6_c7c8)
	testChainID     = eth.ChainIDFromUInt64(0xd1d2_d3d4_d5d6_d7d8)
	testPayload     = []byte("example payload")
	testMsgHash     = common.HexToHash("0x8017559a85b12c04b14a1a425d53486d1015f833714a09bd62f04152a7e2ae9b")
	testLogHash     = common.HexToHash("0xf9ed05990c887d3f86718aabd7e940faaa75d6a5cd44602e89642586ce85f2aa")
	testChecksum    = MessageChecksum(common.HexToHash("0x03749e87fd7789575de9906569deb05aaf220dc4cfab3d8abbfd34a2e1d7d357"))
	testLookupEntry = common.HexToHash("0x01000000d1d2d3d4d5d6d7d8a1a2a3a4a5a6a7a8c1c2c3c4c5c6c7c8b1b2b3b4")
)

func TestMessage(t *testing.T) {
	msg := Message{
		Identifier: Identifier{
			Origin:      testOrigin,
			BlockNumber: testBlockNumber,
			LogIndex:    testLogIndex,
			Timestamp:   testTimestamp,
			ChainID:     testChainID,
		},
		PayloadHash: testMsgHash,
	}
	t.Run("checksum", func(t *testing.T) {
		require.Equal(t, testChecksum, msg.Checksum())
	})
	t.Run("json roundtrip", func(t *testing.T) {
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		var out Message
		require.NoError(t, json.Unmarshal(data, &out))
		require.Equal(t, msg, out)
	})
}

func TestChecksumArgs(t *testing.T) {
	args := ChecksumArgs{
		BlockNumber: testBlockNumber,
		LogIndex:    testLogIndex,
		Timestamp:   testTimestamp,
		ChainID:     testChainID,
		LogHash:     testLogHash,
	}
	t.Run("checksum", func(t *testing.T) {
		require.Equal(t, testChecksum, args.Checksum())
	})
	t.Run("as query", func(t *testing.T) {
		q := args.Query()
		require.Equal(t, testBlockNumber, q.BlockNum)
		require.Equal(t, testTimestamp, q.Timestamp)
		require.Equal(t, testLogIndex, q.LogIdx)
		require.Equal(t, testChecksum, q.Checksum)
	})
	t.Run("as access", func(t *testing.T) {
		acc := args.Access()
		require.Equal(t, testBlockNumber, acc.BlockNumber)
		require.Equal(t, testTimestamp, acc.Timestamp)
		require.Equal(t, testLogIndex, acc.LogIndex)
		require.Equal(t, testChainID, acc.ChainID)
		require.Equal(t, testChecksum, acc.Checksum)
	})
}

func TestIdentifier(t *testing.T) {
	id := Identifier{
		Origin:      testOrigin,
		BlockNumber: testBlockNumber,
		LogIndex:    testLogIndex,
		Timestamp:   testTimestamp,
		ChainID:     testChainID,
	}
	t.Run("json roundtrip", func(t *testing.T) {
		data, err := json.Marshal(id)
		require.NoError(t, err)
		var out Identifier
		require.NoError(t, json.Unmarshal(data, &out))
		require.Equal(t, id, out)
	})
}

func TestSafetyLevel(t *testing.T) {
	for _, lvl := range []SafetyLevel{
		Finalized,
		CrossSafe,
		LocalSafe,
		CrossUnsafe,
		LocalUnsafe,
		Invalid,
	} {
		upper := strings.ToUpper(lvl.String())
		var x SafetyLevel
		require.ErrorContains(t, json.Unmarshal([]byte(fmt.Sprintf("%q", upper)), &x), "unrecognized", "case sensitive")
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf("%q", lvl.String())), &x))
		dat, err := json.Marshal(x)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%q", lvl.String()), string(dat))
	}
	var x SafetyLevel
	require.ErrorContains(t, json.Unmarshal([]byte(`""`), &x), "unrecognized", "empty")
	require.ErrorContains(t, json.Unmarshal([]byte(`"foobar"`), &x), "unrecognized", "other")
}

type execDescrTestCase struct {
	name             string
	ed               ExecutingDescriptor
	expiryWindow     uint64
	initMsgTimestamp uint64
	errStr           string // empty if no error
}

func TestExecutingDescriptorAccessCheck(t *testing.T) {
	testCases := []execDescrTestCase{
		{
			name: "success",
			ed: ExecutingDescriptor{
				Timestamp: 3,
				Timeout:   0,
			},
			expiryWindow:     10,
			initMsgTimestamp: 2,
		},
		{
			name: "future exec",
			ed: ExecutingDescriptor{
				Timestamp: 3,
				Timeout:   0,
			},
			expiryWindow:     10,
			initMsgTimestamp: 4,
			errStr:           "broke timestamp invariant",
		},
		{
			name: "access-list checks are extra strict, don't allow intra-timestamp",
			ed: ExecutingDescriptor{
				Timestamp: 3,
				Timeout:   0,
			},
			expiryWindow:     10,
			initMsgTimestamp: 3,
			errStr:           "not allow intra-timestamp",
		},
		{
			name: "attempt init-msg timestamp overflow",
			ed: ExecutingDescriptor{
				Timestamp: (^uint64(0)) - 2,
				Timeout:   0,
			},
			expiryWindow:     10,
			initMsgTimestamp: (^uint64(0)) - 3,
			errStr:           "overflow",
		},
		{
			name: "expired",
			ed: ExecutingDescriptor{
				Timestamp: 100,
				Timeout:   0,
			},
			expiryWindow:     10,
			initMsgTimestamp: 89,
			errStr:           "expired",
		},
		{
			name: "timeout overflow",
			ed: ExecutingDescriptor{
				Timestamp: 100,
				Timeout:   (^uint64(0)) - 3,
			},
			expiryWindow:     10,
			initMsgTimestamp: 99,
			errStr:           "overflow",
		},
		{
			name: "timeout, valid at exec timestamp, but not shortly after",
			ed: ExecutingDescriptor{
				Timestamp: 100,
				Timeout:   10, //timeout asks for 100+10=110
			},
			expiryWindow:     10, // valid till 95+10 = 105
			initMsgTimestamp: 95,
			errStr:           "timeout",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ed.AccessCheck(tc.expiryWindow, tc.initMsgTimestamp)
			if tc.errStr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.errStr)
			}
		})
	}
}

func TestPayloadHashToLogHash(t *testing.T) {
	logHash := PayloadHashToLogHash(testMsgHash, testOrigin)
	require.Equal(t, testLogHash, logHash)
}

func TestLogToMessagePayload(t *testing.T) {
	payload := LogToMessagePayload(&ethTypes.Log{
		Data: testPayload,
	})
	require.Equal(t, hexutil.Encode(testPayload), hexutil.Encode(payload))

	t.Run("1 topic", func(t *testing.T) {
		v := LogToMessagePayload(&ethTypes.Log{
			Data: []byte(`foobar`),
			Topics: []common.Hash{
				crypto.Keccak256Hash([]byte(`topic0`)),
			},
		})
		expected := make([]byte, 0)
		expected = append(expected, crypto.Keccak256([]byte(`topic0`))...)
		expected = append(expected, []byte(`foobar`)...)
		require.Equal(t, expected, v)
	})

	t.Run("4 topics", func(t *testing.T) {
		v := LogToMessagePayload(&ethTypes.Log{
			Data: []byte(`foobar`),
			Topics: []common.Hash{
				crypto.Keccak256Hash([]byte(`topic0`)),
				crypto.Keccak256Hash([]byte(`topic1`)),
				crypto.Keccak256Hash([]byte(`topic2`)),
				crypto.Keccak256Hash([]byte(`topic3`)),
			},
		})
		expected := make([]byte, 0)
		expected = append(expected, crypto.Keccak256([]byte(`topic0`))...)
		expected = append(expected, crypto.Keccak256([]byte(`topic1`))...)
		expected = append(expected, crypto.Keccak256([]byte(`topic2`))...)
		expected = append(expected, crypto.Keccak256([]byte(`topic3`))...)
		expected = append(expected, []byte(`foobar`)...)
		require.Equal(t, expected, v)
	})
}

func TestAccess(t *testing.T) {
	acc := Access{
		BlockNumber: testBlockNumber,
		Timestamp:   testTimestamp,
		LogIndex:    testLogIndex,
		ChainID:     testChainID,
		Checksum:    MessageChecksum(testChecksum),
	}
	t.Run("json roundtrip", func(t *testing.T) {
		data, err := json.Marshal(acc)
		require.NoError(t, err)
		var out Access
		require.NoError(t, json.Unmarshal(data, &out))
		require.Equal(t, acc, out)
	})
}

func TestParseAccess(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, _, err := ParseAccess(nil)
		require.ErrorIs(t, err, errExpectedEntry)
	})
	t.Run("unexpected 0 type", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: 0x00},
		})
		require.ErrorIs(t, err, errUnexpectedEntryType)
		require.ErrorContains(t, err, "expected lookup")
	})
	t.Run("unexpected arbitrary type", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: 10},
		})
		require.ErrorIs(t, err, errUnexpectedEntryType)
		require.ErrorContains(t, err, "expected lookup")
	})
	t.Run("unexpected non-zero padding", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: PrefixLookup, 1: 0x01}, // valid lookup prefix byte, but non-zero value in padding area
		})
		require.ErrorIs(t, err, errMalformedEntry)
		require.ErrorContains(t, err, "expected zero bytes")
	})
	t.Run("incomplete", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: PrefixLookup}, // valid lookup, but no checksum after
		})
		require.ErrorIs(t, err, errExpectedEntry)
	})
	t.Run("unexpected 0 type after checksum", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: PrefixLookup},
			{0: 0},
		})
		require.ErrorIs(t, err, errUnexpectedEntryType)
	})
	t.Run("unexpected lookup repeat", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: PrefixLookup},
			{0: PrefixLookup},
		})
		require.ErrorIs(t, err, errUnexpectedEntryType)
	})
	t.Run("unexpected arbitrary type after checksum", func(t *testing.T) {
		_, _, err := ParseAccess([]common.Hash{
			{0: PrefixLookup},
			{0: 10}, // unexpected type byte
		})
		require.ErrorIs(t, err, errUnexpectedEntryType)
	})
	t.Run("valid but zero", func(t *testing.T) {
		remaining, acc, err := ParseAccess([]common.Hash{
			{0: PrefixLookup},   // valid lookup entry
			{0: PrefixChecksum}, // valid checksum entry
		})
		require.NoError(t, err)
		require.Equal(t, Access{
			BlockNumber: 0,
			Timestamp:   0,
			LogIndex:    0,
			ChainID:     eth.ChainID{},
			Checksum:    MessageChecksum{0: PrefixChecksum},
		}, acc)
		require.Empty(t, remaining)
	})
	t.Run("valid", func(t *testing.T) {
		acc := Access{
			BlockNumber: testBlockNumber,
			Timestamp:   testTimestamp,
			LogIndex:    testLogIndex,
			ChainID:     testChainID,
			Checksum:    MessageChecksum(testChecksum),
		}
		remaining, parsed, err := ParseAccess([]common.Hash{
			testLookupEntry,
			common.Hash(acc.Checksum),
		})
		require.NoError(t, err)
		require.Equal(t, acc, parsed)
		require.Empty(t, remaining)
	})
	t.Run("repeat", func(t *testing.T) {
		acc := Access{
			BlockNumber: testBlockNumber,
			Timestamp:   testTimestamp,
			LogIndex:    testLogIndex,
			ChainID:     testChainID,
			Checksum:    MessageChecksum(testChecksum),
		}
		remaining, parsed, err := ParseAccess([]common.Hash{
			testLookupEntry,
			common.Hash(acc.Checksum),
			testLookupEntry,
			common.Hash(acc.Checksum),
		})
		require.NoError(t, err)
		require.Equal(t, acc, parsed)
		require.Len(t, remaining, 2)
		remaining2, parsed2, err := ParseAccess(remaining)
		require.NoError(t, err)
		require.Equal(t, acc, parsed2)
		require.Empty(t, remaining2)
	})
	t.Run("with chainID extension", func(t *testing.T) {
		acc := Access{
			BlockNumber: testBlockNumber,
			Timestamp:   testTimestamp,
			LogIndex:    testLogIndex,
			ChainID:     eth.ChainIDFromBytes32([32]byte{0: 7, 31: 10}),
			Checksum:    MessageChecksum(testChecksum),
		}
		remaining, parsed, err := ParseAccess([]common.Hash{
			acc.lookupEntry(),
			acc.chainIDExtensionEntry(),
			common.Hash(acc.Checksum),
		})
		require.NoError(t, err)
		require.Equal(t, acc, parsed)
		require.Empty(t, remaining)
	})
}

func TestEncodeAccessList(t *testing.T) {
	acc := Access{
		BlockNumber: testBlockNumber,
		Timestamp:   testTimestamp,
		LogIndex:    testLogIndex,
		ChainID:     testChainID,
		Checksum:    MessageChecksum(testChecksum),
	}
	t.Run("valid single", func(t *testing.T) {
		accList := EncodeAccessList([]Access{acc})
		require.Len(t, accList, 2)
		require.Equal(t, testLookupEntry, accList[0])
		require.Equal(t, common.Hash(testChecksum), accList[1])
		_, result, err := ParseAccess(accList)
		require.NoError(t, err)
		require.Equal(t, acc, result, "roundtrip")
	})
	t.Run("valid repeat", func(t *testing.T) {
		accList := EncodeAccessList([]Access{
			acc,
			acc,
		})
		require.Len(t, accList, 4)
		require.Equal(t, testLookupEntry, accList[0])
		require.Equal(t, common.Hash(testChecksum), accList[1])
		require.Equal(t, testLookupEntry, accList[2])
		require.Equal(t, common.Hash(testChecksum), accList[3])
	})
	t.Run("roundtrip", func(t *testing.T) {
		accObjects := make([]Access, 0)
		rng := rand.New(rand.NewSource(1234))
		randB32 := func() (out [32]byte) {
			rng.Read(out[:])
			return
		}
		// test a big random access-list
		count := 200
		for i := 0; i < count; i++ {
			chainID := eth.ChainIDFromBytes32(randB32())
			if rng.Intn(5) < 2 { // don't make them all full random bytes32
				chainID = eth.ChainIDFromUInt64(rng.Uint64())
			}
			checksum := randB32()
			checksum[0] = PrefixChecksum
			accObjects = append(accObjects, Access{
				BlockNumber: rng.Uint64(),
				Timestamp:   rng.Uint64(),
				LogIndex:    rng.Uint32(),
				ChainID:     chainID,
				Checksum:    checksum,
			})
		}
		list := EncodeAccessList(accObjects)
		var result []Access
		for i := 0; i < count && len(list) > 0; i++ {
			remaining, v, err := ParseAccess(list)
			require.NoError(t, err)
			result = append(result, v)
			list = remaining
		}
		require.Empty(t, list, "need to exhaust entries, expecting to be done")
		require.Equal(t, accObjects, result, "roundtrip of random entries should work")
	})
}
