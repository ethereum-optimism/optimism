package interopsmoke

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

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
		userA: &remoteUser{chain: &remoteChain{name: "L2A"}},
		userB: &remoteUser{chain: &remoteChain{name: "L2B"}},
	}
	for _, tc := range []struct {
		direction string
		wantNames []string
		wantErr   bool
	}{
		{direction: directionBoth, wantNames: []string{"A->B", "B->A"}},
		{direction: "", wantNames: []string{"A->B", "B->A"}},
		{direction: directionAToB, wantNames: []string{"A->B"}},
		{direction: directionBToA, wantNames: []string{"B->A"}},
		{direction: "sideways", wantErr: true},
	} {
		t.Run(tc.direction, func(t *testing.T) {
			env.direction = tc.direction
			dirs, err := invalidDirections(env)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if len(dirs) != len(tc.wantNames) {
				t.Fatalf("got %d directions, want %d", len(dirs), len(tc.wantNames))
			}
			for i, want := range tc.wantNames {
				if dirs[i].name != want {
					t.Fatalf("direction %d = %s, want %s", i, dirs[i].name, want)
				}
			}
		})
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
