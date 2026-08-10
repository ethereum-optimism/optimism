package interopsmoke

import (
	"context"
	"io"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/urfave/cli/v2"
)

func TestFlowOutput(t *testing.T) {
	amount := eth.OneHundredthEther
	receipt := &types.Receipt{
		TxHash:      common.HexToHash("0x1234"),
		BlockNumber: big.NewInt(123),
	}
	output := flowOutput(
		"Bridge",
		&remoteChain{chainID: eth.ChainIDFromUInt64(900)},
		&remoteChain{chainID: eth.ChainIDFromUInt64(901)},
		&amount,
		receipt,
	)

	for _, want := range []string{
		"source chain ID 900",
		"destination chain ID 901",
		"amount 10000000000000000 wei",
		"tx 0x0000000000000000000000000000000000000000000000000000000000001234",
		"included in block 123",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output = %q, want %q", output, want)
		}
	}
}

func TestMessageLandingOutput(t *testing.T) {
	receipt := &types.Receipt{
		TxHash:      common.HexToHash("0x1234"),
		BlockNumber: big.NewInt(123),
	}
	source := &remoteChain{name: "L2A", chainID: eth.ChainIDFromUInt64(900)}
	destination := &remoteChain{name: "L2B", chainID: eth.ChainIDFromUInt64(901)}

	initOutput := messageLandingOutput("Initiating message", source, destination, source, receipt)
	execOutput := messageLandingOutput("Executing message", source, destination, destination, receipt)
	for _, want := range []string{
		"Initiating message landed on source chain L2A",
		"Executing message landed on destination chain L2B",
		"source chain ID 900",
		"destination chain ID 901",
		"included in block 123",
	} {
		if !strings.Contains(initOutput+execOutput, want) {
			t.Errorf("output = %q, want %q", initOutput+execOutput, want)
		}
	}
}

func TestIterationsFlagDefaultsToOne(t *testing.T) {
	var iterations *cli.UintFlag
	for _, flag := range Subcommands("OP_UP")[0].Flags {
		if flag.Names()[0] == iterationsFlagName {
			iterations, _ = flag.(*cli.UintFlag)
			break
		}
	}
	if iterations == nil {
		t.Fatal("iterations flag not found")
	}
	if iterations.Value != 1 {
		t.Fatalf("iterations default = %d, want 1", iterations.Value)
	}
}

func TestRunIterations(t *testing.T) {
	calls := 0
	if err := runIterations(3, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestValidateIterations(t *testing.T) {
	if err := validateIterations(0); err == nil {
		t.Fatal("expected zero iterations to fail")
	}
	if err := validateIterations(1); err != nil {
		t.Fatalf("one iteration failed: %v", err)
	}
}

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

func TestInvalidMessageDirectionFlag(t *testing.T) {
	var invalidMessage *cli.Command
	for _, command := range Subcommands("OP_UP") {
		if command.Name == "invalid-message" {
			invalidMessage = command
			break
		}
	}
	if invalidMessage == nil {
		t.Fatal("invalid-message command not found")
	}
	for _, flag := range invalidMessage.Flags {
		if flag.Names()[0] == "direction" {
			return
		}
	}
	t.Fatal("invalid-message command missing --direction flag")
}

func TestInvalidDirections(t *testing.T) {
	for _, tc := range []struct {
		name      string
		direction string
		wantNames []string
		wantErr   string
	}{
		{
			name:      "defaults to every ordered pair",
			wantNames: []string{"A->B", "A->C", "B->A", "B->C", "C->A", "C->B"},
		},
		{
			name:      "selects one direction",
			direction: "A->B",
			wantNames: []string{"A->B"},
		},
		{
			name:      "rejects malformed selector",
			direction: "A-B",
			wantErr:   "must be in the form A->B",
		},
		{
			name:      "rejects unknown selector",
			direction: "A->D",
			wantErr:   "unknown",
		},
		{
			name:      "rejects same chain selector",
			direction: "A->A",
			wantErr:   "must differ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := &smokeEnv{
				invalidDirection: tc.direction,
				users: []*remoteUser{
					{chain: &remoteChain{name: "L2A"}},
					{chain: &remoteChain{name: "L2B"}},
					{chain: &remoteChain{name: "L2C"}},
				},
			}
			dirs, err := invalidDirections(env)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
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
