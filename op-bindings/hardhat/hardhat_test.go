package hardhat_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"

	"github.com/stretchr/testify/require"
)

func TestGetFullyQualifiedName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fqn    hardhat.QualifiedName
		expect string
	}{
		{
			fqn:    hardhat.QualifiedName{"contract.sol", "C"},
			expect: "contract.sol:C",
		},
		{
			fqn:    hardhat.QualifiedName{"folder/contract.sol", "C"},
			expect: "folder/contract.sol:C",
		},
		{
			fqn:    hardhat.QualifiedName{"folder/a:b/contract.sol", "C"},
			expect: "folder/a:b/contract.sol:C",
		},
	}

	for _, test := range cases {
		got := hardhat.GetFullyQualifiedName(test.fqn.SourceName, test.fqn.ContractName)
		require.Equal(t, got, test.expect)
	}
}

func TestParseFullyQualifiedName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fqn    string
		expect hardhat.QualifiedName
	}{
		{
			fqn:    "contract.sol:C",
			expect: hardhat.QualifiedName{"contract.sol", "C"},
		},
		{
			fqn:    "folder/contract.sol:C",
			expect: hardhat.QualifiedName{"folder/contract.sol", "C"},
		},
		{
			fqn:    "folder/a:b/contract.sol:C",
			expect: hardhat.QualifiedName{"folder/a:b/contract.sol", "C"},
		},
	}

	for _, test := range cases {
		got := hardhat.ParseFullyQualifiedName(test.fqn)
		require.Equal(t, got, test.expect)
	}
}

func TestIsFullyQualifiedName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fqn    string
		expect bool
	}{
		{
			fqn:    "contract.sol:C",
			expect: true,
		},
		{
			fqn:    "folder/contract.sol:C",
			expect: true,
		},
		{
			fqn:    "folder/a:b/contract.sol:C",
			expect: true,
		},
		{
			fqn:    "C",
			expect: false,
		},
		{
			fqn:    "contract.sol",
			expect: false,
		},
		{
			fqn:    "folder/contract.sol",
			expect: false,
		},
	}

	for _, test := range cases {
		got := hardhat.IsFullyQualifiedName(test.fqn)
		require.Equal(t, got, test.expect)
	}
}

func TestHardhatGetArtifact(t *testing.T) {
	t.Parallel()

	hh, err := hardhat.New(
		"goerli",
		[]string{"testdata/artifacts"},
		[]string{"testdata/deployments"},
	)
	require.Nil(t, err)

	artifact, err := hh.GetArtifact("HelloWorld")
	require.Nil(t, err)
	require.NotNil(t, artifact)
}

func TestHardhatGetBuildInfo(t *testing.T) {
	t.Parallel()

	hh, err := hardhat.New(
		"goerli",
		[]string{"testdata/artifacts"},
		[]string{"testdata/deployments"},
	)
	require.Nil(t, err)

	buildInfo, err := hh.GetBuildInfo("HelloWorld")
	require.Nil(t, err)
	require.NotNil(t, buildInfo)
}

func TestHardhatGetDeployments(t *testing.T) {
	t.Parallel()

	hh, err := hardhat.New(
		"goerli",
		[]string{"testdata/artifacts"},
		[]string{"testdata/deployments"},
	)
	require.Nil(t, err)

	deployment, err := hh.GetDeployment("OptimismPortal")
	require.Nil(t, err)
	require.NotNil(t, deployment)
}

func TestHardhatGetDeploymentsDuplicates(t *testing.T) {
	t.Parallel()

	// Set the network to an empty string to simulate
	// an invalid network name.
	_, err := hardhat.New(
		"",
		[]string{"testdata/artifacts"},
		[]string{"testdata/deployments"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate deployment")
}

func TestHardhatGetStorageLayout(t *testing.T) {
	t.Parallel()

	hh, err := hardhat.New(
		"goerli",
		[]string{"testdata/artifacts"},
		[]string{"testdata/deployments"},
	)
	require.Nil(t, err)

	storageLayout, err := hh.GetStorageLayout("HelloWorld")
	require.Nil(t, err)
	require.NotNil(t, storageLayout)
}
