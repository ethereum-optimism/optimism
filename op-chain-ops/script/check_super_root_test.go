package script

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var (
	opSepoliaExpectedBlock       = uint64(27114605)
	opSepoliaExpectedBlockHash   = common.HexToHash("0xbc2ac87e80a134b12e0eb971a7ed5046c2f0574ce457d2926f8ba2422c8e5c82")
	baseSepoliaExpectedBlock     = uint64(25131731)
	baseSepoliaExpectedBlockHash = common.HexToHash("0xdee49b02220c4d70f071cddae3276dd6db437d79c46960e5c26b3318739923ac")
	expectedAnchorTimestamp      = uint64(1745748550) // Apr-30-2025 04:49:10 PM +UTC
)

const (
	opSepoliaRPC   = "https://sepolia.optimism.io"
	baseSepoliaRPC = "https://sepolia.base.org"
)

var (
	testRPCs = []string{
		opSepoliaRPC,
		baseSepoliaRPC,
	}

	testTimestamp = uint64(1745748551)
)

func TestNewSuperRootMigrator(t *testing.T) {
	t.Run("ValidInput", func(t *testing.T) {
		migrator, _ := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
		if len(migrator.rpcEndpoints) != len(testRPCs) {
			t.Errorf("expected %d endpoints, got %d", len(testRPCs), len(migrator.rpcEndpoints))
		}
	})

	t.Run("EmptyEndpoints", func(t *testing.T) {
		_, err := NewSuperRootMigrator(nil, []string{}, nil)
		if err == nil {
			t.Fatal("expected error for empty endpoints")
		}
	})

	t.Run("NilTimestamp", func(t *testing.T) {
		_, err := NewSuperRootMigrator(nil, testRPCs, nil)
		if err == nil {
			t.Fatal("expected error for nil timestamp")
		}
	})
}

func TestInitClientsAndFetchIDs(t *testing.T) {
	migrator, err := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	err = migrator.initClientsAndFetchIDs(context.Background())
	if err != nil {
		t.Fatalf("initClientsAndFetchIDs failed: %v", err)
	}

	if len(migrator.ethClients) != len(testRPCs) {
		t.Errorf("expected %d clients, got %d", len(testRPCs), len(migrator.ethClients))
	}

	for _, endpoint := range testRPCs {
		if _, ok := migrator.ethClients[endpoint]; !ok {
			t.Errorf("client for endpoint %s not found", endpoint)
		}
		if _, ok := migrator.chainSettings[endpoint]; !ok {
			t.Errorf("chain settings for endpoint %s not found", endpoint)
		}
	}
}

func TestCalculateTargetBlockNumbers(t *testing.T) {
	migrator, err := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()
	err = migrator.initClientsAndFetchIDs(ctx)
	if err != nil {
		t.Fatalf("initClientsAndFetchIDs failed: %v", err)
	}

	err = migrator.calculateTargetBlockNumbers(ctx)
	if err != nil {
		t.Fatalf("calculateTargetBlockNumbers failed: %v", err)
	}

	for endpoint, settings := range migrator.chainSettings {
		if settings.TargetBlockNumber == nil {
			t.Errorf("expected non-nil TargetBlockNumber for endpoint %s", endpoint)
		}
		if settings.BlockTime == 0 {
			t.Errorf("expected non-zero BlockTime for endpoint %s", endpoint)
		}
		t.Logf("Endpoint: %s, TargetBlockNumber: %v, BlockTime: %d", endpoint, settings.TargetBlockNumber, settings.BlockTime)
	}
}

func TestFindAnchorTimestamp(t *testing.T) {
	t.Run("SingleChainCase", func(t *testing.T) {
		migrator, _ := NewSuperRootMigrator(nil, testRPCs[:1], &testTimestamp)
		if migrator.rpcEndpoints[0] != testRPCs[0] {
			t.Errorf("expected endpoint %s, got %s", testRPCs[0], migrator.rpcEndpoints[0])
		}
		var err error
		ctx := context.Background()
		err = migrator.initClientsAndFetchIDs(ctx)
		if err != nil {
			t.Fatalf("initClientsAndFetchIDs failed: %v", err)
		}

		err = migrator.calculateTargetBlockNumbers(ctx)
		if err != nil {
			t.Fatalf("calculateTargetBlockNumbers failed: %v", err)
		}

		err = migrator.findAnchorTimestamp(context.Background())
		if err != nil {
			t.Fatalf("failed to find anchor timestamp: %v", err)
		}
		if migrator.anchorTimestamp == 0 {
			t.Error("expected non-zero anchor timestamp")
		}

		t.Logf("Found anchor timestamp: %d", migrator.anchorTimestamp)
	})

	t.Run("MultipleChainsCase", func(t *testing.T) {
		migrator, err := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
		if err != nil {
			t.Fatalf("failed to create migrator: %v", err)
		}

		ctx := context.Background()
		err = migrator.initClientsAndFetchIDs(ctx)
		if err != nil {
			t.Fatalf("initClientsAndFetchIDs failed: %v", err)
		}

		err = migrator.calculateTargetBlockNumbers(ctx)
		if err != nil {
			t.Fatalf("calculateTargetBlockNumbers failed: %v", err)
		}

		err = migrator.findAnchorTimestamp(ctx)
		if err != nil {
			t.Fatalf("failed to find anchor timestamp: %v", err)
		}
		if migrator.anchorTimestamp == 0 {
			t.Error("expected non-zero anchor timestamp")
		}

		t.Logf("Found anchor timestamp for multiple chains: %d", migrator.anchorTimestamp)
	})
}

func TestCalculateOutputRoots(t *testing.T) {
	migrator, err := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()
	err = migrator.initClientsAndFetchIDs(ctx)
	if err != nil {
		t.Fatalf("initClientsAndFetchIDs failed: %v", err)
	}

	err = migrator.calculateTargetBlockNumbers(ctx)
	if err != nil {
		t.Fatalf("calculateTargetBlockNumbers failed: %v", err)
	}

	err = migrator.findAnchorTimestamp(ctx)
	if err != nil {
		t.Fatalf("findAnchorTimestamp failed: %v", err)
	}

	err = migrator.calculateOutputRoots(ctx)
	if err != nil {
		t.Fatalf("calculateOutputRoots failed: %v", err)
	}

	for i, output := range migrator.chainOutputs {
		if output.Output == (eth.Bytes32{}) {
			t.Errorf("empty output root for chain %d (ID: %v)", i, output.ChainID)
		}
		t.Logf("ChainOutput[%d]: ChainID=%v OutputRoot=%v", i, output.ChainID, output.Output)
	}
}

func TestCalculateSuperRoot(t *testing.T) {
	migrator, err := NewSuperRootMigrator(nil, testRPCs, &testTimestamp)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()
	err = migrator.initClientsAndFetchIDs(ctx)
	if err != nil {
		t.Fatalf("initClientsAndFetchIDs failed: %v", err)
	}

	err = migrator.calculateTargetBlockNumbers(ctx)
	if err != nil {
		t.Fatalf("calculateTargetBlockNumbers failed: %v", err)
	}

	err = migrator.findAnchorTimestamp(ctx)
	if err != nil {
		t.Fatalf("findAnchorTimestamp failed: %v", err)
	}

	err = migrator.calculateOutputRoots(ctx)
	if err != nil {
		t.Fatalf("calculateOutputRoots failed: %v", err)
	}

	err = migrator.calculateSuperRoot()
	if err != nil {
		t.Fatalf("calculateSuperRoot failed: %v", err)
	}

	if migrator.superRoot == (common.Hash{}) {
		t.Error("expected non-zero super root")
	}

	t.Logf("Super root: %v", migrator.superRoot.Hex())
}
