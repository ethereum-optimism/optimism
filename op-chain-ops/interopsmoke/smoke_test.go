package interopsmoke

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestNewSmokeEnvRequiresAtLeastTwoRPCURLs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		l2URLs []string
	}{
		{name: "zero RPC URLs"},
		{name: "one RPC URL", l2URLs: []string{"http://127.0.0.1:0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := newSmokeEnv(context.Background(), io.Discard, tc.l2URLs, "")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "at least two L2 RPC URLs are required") {
				t.Fatalf("error = %v, want minimum RPC URL error", err)
			}
		})
	}
}

func TestValidateL2URLs(t *testing.T) {
	for _, tc := range []struct {
		urls    []string
		wantErr bool
	}{
		{[]string{"http://a", "http://b"}, false},
		{[]string{"http://a"}, true},
	} {
		if err := validateL2URLs(tc.urls); (err != nil) != tc.wantErr {
			t.Fatalf("validateL2URLs(%v) error = %v, wantErr %v", tc.urls, err, tc.wantErr)
		}
	}
}

func TestValidateChainIDs(t *testing.T) {
	env := &smokeEnv{chains: []*remoteChain{
		{name: "L2A", chainID: eth.ChainIDFromUInt64(1)},
		{name: "L2B", chainID: eth.ChainIDFromUInt64(1)},
	}}
	if err := validateChainIDs(env); err == nil {
		t.Fatal("expected duplicate ID error")
	}
}

func TestValidateInvalidMessageOptions(t *testing.T) {
	for _, tc := range []struct {
		name               string
		blocks, txPerBlock uint
		wantErr            bool
	}{
		{"valid", 2, 3, false},
		{"zero blocks", 0, 1, true},
		{"zero tx per block", 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInvalidMessageOptions(tc.blocks, tc.txPerBlock); (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestInvalidDirections(t *testing.T) {
	env := &smokeEnv{
		users: []*remoteUser{
			{chain: &remoteChain{name: "L2A"}},
			{chain: &remoteChain{name: "L2B"}},
			{chain: &remoteChain{name: "L2C"}},
		},
	}
	dirs, err := invalidDirections(env)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"A->B", "A->C", "B->A", "B->C", "C->A", "C->B"}
	if len(dirs) != len(wantNames) {
		t.Fatalf("got %d directions, want %d", len(dirs), len(wantNames))
	}
	for i, want := range wantNames {
		if dirs[i].name != want {
			t.Fatalf("direction %d = %s, want %s", i, dirs[i].name, want)
		}
	}
}

func TestOrderedPairs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		chainNames []string
		want       []string
	}{
		{
			name:       "two chains",
			chainNames: []string{"L2A", "L2B"},
			want:       []string{"A->B", "B->A"},
		},
		{
			name:       "three chains",
			chainNames: []string{"L2A", "L2B", "L2C"},
			want:       []string{"A->B", "A->C", "B->A", "B->C", "C->A", "C->B"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := &smokeEnv{}
			for _, name := range tc.chainNames {
				env.users = append(env.users, &remoteUser{chain: &remoteChain{name: name}})
			}

			pairs, err := orderedPairs(env)
			if err != nil {
				t.Fatal(err)
			}
			if len(pairs) != len(tc.want) {
				t.Fatalf("got %d pairs, want %d", len(pairs), len(tc.want))
			}
			for i, want := range tc.want {
				if pairs[i].name != want {
					t.Fatalf("pair %d = %s, want %s", i, pairs[i].name, want)
				}
			}
		})
	}
}

func TestFirstLogFrom(t *testing.T) {
	messenger := common.HexToAddress("0x4200000000000000000000000000000000000023")
	eventLogger := common.HexToAddress("0x1111111111111111111111111111111111111111")
	logs := []*types.Log{
		{Address: messenger},
		{Address: eventLogger},
		{Address: eventLogger},
	}

	if got := firstLogFrom(logs, eventLogger); got != 1 {
		t.Fatalf("first log from EventLogger = %d, want 1", got)
	}
	if got := firstLogFrom(logs, messenger); got != 0 {
		t.Fatalf("first log from messenger = %d, want 0", got)
	}
	if got := firstLogFrom(logs, common.Address{}); got != -1 {
		t.Fatalf("absent origin = %d, want -1", got)
	}
	if got := firstLogFrom(nil, eventLogger); got != -1 {
		t.Fatalf("no logs = %d, want -1", got)
	}
}
