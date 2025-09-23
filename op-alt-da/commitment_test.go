package altda

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/stretchr/testify/require"
)

// TestCommitmentData tests the CommitmentData type and its implementations,
// by encoding and decoding the commitment data and verifying the input data.
func TestCommitmentData(t *testing.T) {

	type tcase struct {
		name        string
		commType    CommitmentType
		commData    []byte
		expectedErr error
	}

	testCases := []tcase{
		{
			name:        "valid keccak256 commitment",
			commType:    Keccak256CommitmentType,
			commData:    []byte("abcdefghijklmnopqrstuvwxyz012345"),
			expectedErr: ErrInvalidCommitment,
		},
		{
			name:        "invalid keccak256 commitment",
			commType:    Keccak256CommitmentType,
			commData:    []byte("ab_baddata_yz012345"),
			expectedErr: ErrInvalidCommitment,
		},
		{
			name:        "valid generic commitment",
			commType:    GenericCommitmentType,
			commData:    []byte("any length of data! wow, that's so generic!"),
			expectedErr: ErrInvalidCommitment,
		},
		{
			name:        "invalid commitment type",
			commType:    9,
			commData:    []byte("abcdefghijklmnopqrstuvwxyz012345"),
			expectedErr: ErrInvalidCommitment,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comm, err := DecodeCommitmentData(tc.commData)
			require.ErrorIs(t, err, tc.expectedErr)
			if err == nil {
				// Test that the commitment type is correct
				require.Equal(t, tc.commType, comm.CommitmentType())
				// Test that reencoding the commitment returns the same data
				require.Equal(t, tc.commData, comm.Encode())
				// Test that TxData() returns the same data as the original, prepended with a version byte
				require.Equal(t, append([]byte{params.DerivationVersion1}, tc.commData...), comm.TxData())

				// Test that Verify() returns no error for the correct data
				require.NoError(t, comm.Verify(tc.commData))
				// Test that Verify() returns error for the incorrect data
				// don't do this for GenericCommitmentType, which does not do any verification
				if tc.commType != GenericCommitmentType {
					require.ErrorIs(t, ErrCommitmentMismatch, comm.Verify([]byte("wrong data")))
				}
			}
		})
	}
}

// TestNewCommitmentData tests the NewCommitmentData function specifically,
// focusing on the core safety issue: preventing nil interface returns.
func TestNewCommitmentData(t *testing.T) {
	tests := []struct {
		name        string
		commitType  CommitmentType
		input       []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid keccak256 commitment with data",
			commitType:  Keccak256CommitmentType,
			input:       []byte("test data"),
			expectError: false,
		},
		{
			name:        "valid keccak256 commitment with empty input",
			commitType:  Keccak256CommitmentType,
			input:       []byte{}, // Empty input is valid for Keccak256
			expectError: false,
		},
		{
			name:        "valid generic commitment with data",
			commitType:  GenericCommitmentType,
			input:       []byte("any data here"),
			expectError: false,
		},
		{
			name:        "valid generic commitment with empty input",
			commitType:  GenericCommitmentType,
			input:       []byte{}, // Empty input is valid for Generic too
			expectError: false,
		},
		{
			name:        "unsupported commitment type should fail",
			commitType:  CommitmentType(99), // Invalid type
			input:       []byte("test data"),
			expectError: true,
			errorMsg:    "unsupported commitment type: 99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitment, err := NewCommitmentData(tt.commitType, tt.input)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, commitment)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, commitment)
				require.Equal(t, tt.commitType, commitment.CommitmentType())
			}
		})
	}
}
