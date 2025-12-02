package types

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const testMaxDepth = 3

func createTestClaims() (Claim, Claim, Claim, Claim) {
	// root & middle are from the trace "abcdexyz"
	// top & bottom are from the trace  "abcdefgh"
	root := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: NewPosition(0, common.Big0),
		},
		// Root claim has no parent
	}
	top := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: NewPosition(1, common.Big0),
		},
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	middle := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: NewPosition(2, big.NewInt(2)),
		},
		ContractIndex:       2,
		ParentContractIndex: 1,
	}

	bottom := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(3, big.NewInt(4)),
		},
		ContractIndex:       3,
		ParentContractIndex: 2,
	}

	return root, top, middle, bottom
}

func TestIsDuplicate(t *testing.T) {
	root, top, middle, bottom := createTestClaims()
	g := NewGameState([]Claim{root, top}, testMaxDepth)

	// Root + Top should be duplicates
	require.True(t, g.IsDuplicate(root))
	require.True(t, g.IsDuplicate(top))

	// Middle + Bottom should not be a duplicate
	require.False(t, g.IsDuplicate(middle))
	require.False(t, g.IsDuplicate(bottom))
}

func TestGame_Claims(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	expected := []Claim{root, top, middle, bottom}
	g := NewGameState(expected, testMaxDepth)

	// Validate claim pairs.
	actual := g.Claims()
	require.ElementsMatch(t, expected, actual)
}

func TestGame_DefendsParent(t *testing.T) {
	tests := []struct {
		name     string
		game     *gameState
		expected bool
	}{
		{
			name:     "LeftChildAttacks",
			game:     buildGameWithClaim(big.NewInt(2), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(3), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(4), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubSecondChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(5), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightLeftChildDefendsParent",
			game:     buildGameWithClaim(big.NewInt(6), big.NewInt(1)),
			expected: true,
		},
		{
			name:     "SubThirdChildDefends",
			game:     buildGameWithClaim(big.NewInt(7), big.NewInt(1)),
			expected: true,
		},
		{
			name: "RootDoesntDefend",
			game: NewGameState([]Claim{
				{
					ClaimData: ClaimData{
						Position: NewPositionFromGIndex(big.NewInt(0)),
					},
					ContractIndex: 0,
				},
			}, testMaxDepth),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := test.game.Claims()
			require.Equal(t, test.expected, test.game.DefendsParent(claims[len(claims)-1]))
		})
	}
}

func TestAncestorWithTraceIndex(t *testing.T) {
	depth := Depth(4)
	claims := []Claim{
		{
			ClaimData: ClaimData{
				Position: NewPositionFromGIndex(big.NewInt(0)),
			},
			ContractIndex:       0,
			ParentContractIndex: 0,
		},
	}
	addClaimAtPos := func(parent Claim, pos Position) Claim {
		claim := Claim{
			ClaimData: ClaimData{
				Position: pos,
			},
			ParentContractIndex: parent.ContractIndex,
			ContractIndex:       len(claims),
		}
		claims = append(claims, claim)
		return claim
	}
	attack := func(claim Claim) Claim {
		return addClaimAtPos(claim, claim.Position.Attack())
	}
	defend := func(claim Claim) Claim {
		return addClaimAtPos(claim, claim.Position.Defend())
	}
	// Create a variety of paths to leaf nodes
	attack(attack(attack(attack(claims[0]))))
	defend(defend(defend(defend(claims[0]))))
	defend(attack(defend(attack(claims[0]))))
	attack(defend(attack(defend(claims[0]))))
	attack(attack(defend(defend(claims[0]))))
	defend(defend(attack(attack(claims[0]))))

	game := NewGameState(claims, depth)
	// Every claim should be able to find the root's trace index
	for _, claim := range claims {
		actual, ok := game.AncestorWithTraceIndex(claim, claims[0].TraceIndex(depth))
		require.True(t, ok)
		require.Equal(t, claims[0], actual)
	}

	// Leaf claims should be able to find the trace index before and after
	for _, claim := range game.Claims() {
		if claim.Depth() != depth {
			// Only leaf nodes are guaranteed to have the pre and post states available
			continue
		}
		claimIdx := claim.TraceIndex(depth)

		actual, ok := game.AncestorWithTraceIndex(claim, claimIdx)
		require.True(t, ok)
		require.Equal(t, claim, actual, "Should get leaf claim for its own trace index")

		// The right most claim doesn't have
		if claim.IndexAtDepth().Cmp(big.NewInt(30)) < 0 {
			idx := new(big.Int).Add(claimIdx, big.NewInt(1))
			actual, ok = game.AncestorWithTraceIndex(claim, idx)
			require.Truef(t, ok, "Should find claim with next trace index for claim %v index at depth %v", claim.ContractIndex, claim.IndexAtDepth())
			require.Equalf(t, idx, actual.TraceIndex(depth), "Should find claim with next trace index for claim %v index at depth %v", claim.ContractIndex, claim.IndexAtDepth())
		}

		if claimIdx.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		idx := new(big.Int).Sub(claimIdx, big.NewInt(1))
		actual, ok = game.AncestorWithTraceIndex(claim, idx)
		require.True(t, ok)
		require.Equal(t, idx, actual.TraceIndex(depth), "Should find claim with previous trace index")
	}

	actual, ok := game.AncestorWithTraceIndex(claims[0], big.NewInt(0))
	require.False(t, ok)
	require.Equal(t, Claim{}, actual)

	actual, ok = game.AncestorWithTraceIndex(claims[1], big.NewInt(1))
	require.False(t, ok)
	require.Equal(t, Claim{}, actual)

	actual, ok = game.AncestorWithTraceIndex(claims[3], big.NewInt(1))
	require.True(t, ok)
	require.Equal(t, claims[3], actual)
}

func TestChessClock(t *testing.T) {
	rootTime := time.UnixMilli(42978249)
	defenderRootClaim, challengerFirstClaim, defenderSecondClaim, challengerSecondClaim := createTestClaims()
	defenderRootClaim.Clock = Clock{Timestamp: rootTime, Duration: 0}
	challengerFirstClaim.Clock = Clock{Timestamp: rootTime.Add(5 * time.Minute), Duration: 5 * time.Minute}
	defenderSecondClaim.Clock = Clock{Timestamp: challengerFirstClaim.Clock.Timestamp.Add(2 * time.Minute), Duration: 2 * time.Minute}
	challengerSecondClaim.Clock = Clock{Timestamp: defenderSecondClaim.Clock.Timestamp.Add(3 * time.Minute), Duration: 8 * time.Minute}
	claims := []Claim{defenderRootClaim, challengerFirstClaim, defenderSecondClaim, challengerSecondClaim}
	game := NewGameState(claims, 10)

	// At the time the root claim is posted, both defender and challenger have no time on their chess clock
	// The root claim starts the chess clock for the challenger
	require.Equal(t, time.Duration(0), game.ChessClock(rootTime, game.Claims()[0]))
	// As time progresses, the challenger's chess clock increases
	require.Equal(t, 2*time.Minute, game.ChessClock(rootTime.Add(2*time.Minute), game.Claims()[0]))

	// The challenger's first claim arrives 5 minutes after the root claim and starts the clock for the defender
	// This is the defender's first turn so at the time the claim is posted, the defender's chess clock is 0
	require.Equal(t, time.Duration(0), game.ChessClock(challengerFirstClaim.Clock.Timestamp, challengerFirstClaim))
	// As time progresses, the defender's chess clock increases
	require.Equal(t, 3*time.Minute, game.ChessClock(challengerFirstClaim.Clock.Timestamp.Add(3*time.Minute), challengerFirstClaim))

	// The defender's second claim arrives 2 minutes after the challenger's first claim.
	// This starts the challenger's clock again. At the time of the claim it already has 5 minutes on the clock
	// from the challenger's previous turn
	require.Equal(t, 5*time.Minute, game.ChessClock(defenderSecondClaim.Clock.Timestamp, defenderSecondClaim))
	// As time progresses the challenger's chess clock increases
	require.Equal(t, 5*time.Minute+30*time.Second, game.ChessClock(defenderSecondClaim.Clock.Timestamp.Add(30*time.Second), defenderSecondClaim))

	// The challenger's second claim arrives 3 minutes after the defender's second claim.
	// This starts the defender's clock again. At the time of the claim it already has 2 minutes on the clock
	// from the defenders previous turn
	require.Equal(t, 2*time.Minute, game.ChessClock(challengerSecondClaim.Clock.Timestamp, challengerSecondClaim))
	// As time progresses, the defender's chess clock increases
	require.Equal(t, 2*time.Minute+45*time.Minute, game.ChessClock(challengerSecondClaim.Clock.Timestamp.Add(45*time.Minute), challengerSecondClaim))
}

func buildGameWithClaim(claimGIndex *big.Int, parentGIndex *big.Int) *gameState {
	parentClaim := Claim{
		ClaimData: ClaimData{
			Position: NewPositionFromGIndex(parentGIndex),
		},
		ContractIndex: 0,
	}
	claim := Claim{
		ClaimData: ClaimData{
			Position: NewPositionFromGIndex(claimGIndex),
		},
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	return NewGameState([]Claim{parentClaim, claim}, testMaxDepth)
}
