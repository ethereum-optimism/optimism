package bindgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var generator BindGenGeneratorRemote = BindGenGeneratorRemote{}

func TestRemoveDeploymentSalt(t *testing.T) {
	for _, tt := range removeDeploymentSaltTests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := generator.removeDeploymentSalt(tt.deploymentData, tt.deploymentSalt)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestRemoveDeploymentSaltFailures(t *testing.T) {
	for _, tt := range removeDeploymentSaltTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.removeDeploymentSalt(tt.deploymentData, tt.deploymentSalt)
			require.Equal(t, err.Error(), tt.expectedError)
		})
	}
}
