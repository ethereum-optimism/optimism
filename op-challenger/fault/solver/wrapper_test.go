package solver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
)

func TestAgreeWithClaim(t *testing.T) {
	maxDepth := 4
	alphabetProvider := test.NewFullAlphabetProvider(t, maxDepth)
	tests := []struct {
		name          string
		claim         types.ClaimData
		gameDepth     int
		expectedErr   error
		expectedAgree bool
	}{
		{
			name:          "AgreeWithLevel_CorrectRoot",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).CreateRootClaim(true).ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: true,
		},
		{
			name:          "AgreeWithLevel_IncorrectRoot",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).CreateRootClaim(false).ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: false,
		},
		{
			name:          "AgreeWithLevel_EvenDepth",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).Seq(false).Attack(true).Get().ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: true,
		},
		{
			name:          "DisagreeWithLevel_EvenDepth",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).Seq(false).Attack(false).Get().ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: false,
		},
		{
			name:          "AgreeWithLevel_OddDepth",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).Seq(false).Attack(false).Defend(true).Get().ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: true,
		},
		{
			name:          "DisagreeWithLevel_OddDepth",
			claim:         test.NewAlphabetClaimBuilder(t, maxDepth).Seq(false).Attack(false).Defend(false).Get().ClaimData,
			gameDepth:     maxDepth,
			expectedAgree: false,
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			wrapper := NewProviderWrapper(alphabetProvider)
			agree, err := wrapper.AgreeWithClaim(tableTest.claim, tableTest.gameDepth)
			if tableTest.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tableTest.expectedErr)
			}
			require.Equal(t, tableTest.expectedAgree, agree)
		})
	}
}
