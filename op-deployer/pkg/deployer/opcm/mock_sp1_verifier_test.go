package opcm

import (
	"testing"
	"testing/fstest"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func mockSP1VerifierArtifacts(artifact string) fstest.MapFS {
	return fstest.MapFS{MockSP1VerifierArtifact + "/" + MockSP1VerifierContract + ".json": {Data: []byte(artifact)}}
}

// TestDeployMockSP1VerifierGenesis checks that the genesis gets the runtime code of the verifier at
// the next CREATE address of the sender, whether the artifact leaves out its empty immutable
// references or not.
func TestDeployMockSP1VerifierGenesis(t *testing.T) {
	for name, immutables := range map[string]string{"absent": "", "empty": `,"immutableReferences":{}`, "null": `,"immutableReferences":null`} {
		t.Run(name, func(t *testing.T) {
			sender := common.Address{0x5e}
			genesis := NewGenesisState(sender, false)
			verifier, err := DeployMockSP1VerifierGenesis(&ForgeEnv{
				Genesis: genesis,
				Artifacts: mockSP1VerifierArtifacts(`{"abi":[],"bytecode":{"object":"0x6002"},` +
					`"deployedBytecode":{"object":"0x6001"` + immutables + `}}`),
			})
			require.NoError(t, err)
			require.Equal(t, crypto.CreateAddress(sender, 0), verifier)
			require.Equal(t, []byte{0x60, 0x01}, []byte(genesis.Allocs.Accounts[verifier].Code))
			require.Equal(t, uint64(1), genesis.Allocs.Accounts[sender].Nonce)
		})
	}
}

// TestDeployMockSP1VerifierGenesisRefusesInitCode checks that the genesis refuses a verifier whose
// deployment leaves more than its runtime code, since the genesis does not run the init code.
func TestDeployMockSP1VerifierGenesisRefusesInitCode(t *testing.T) {
	for name, artifact := range map[string]string{
		"constructor": `{"abi":[{"type":"constructor","inputs":[],"stateMutability":"nonpayable"}],` +
			`"bytecode":{"object":"0x6002"},"deployedBytecode":{"object":"0x6001"}}`,
		"immutables": `{"abi":[],"bytecode":{"object":"0x6002"},` +
			`"deployedBytecode":{"object":"0x6001","immutableReferences":{"7":[{"start":0,"length":32}]}}}`,
		"storage": `{"abi":[],"bytecode":{"object":"0x6002"},"deployedBytecode":{"object":"0x6001"},` +
			`"storageLayout":{"storage":[{"astId":7,"contract":"MockSP1Verifier","label":"x","offset":0,"slot":"0","type":"t_uint256"}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			genesis := NewGenesisState(common.Address{0x5e}, false)
			_, err := DeployMockSP1VerifierGenesis(&ForgeEnv{Genesis: genesis, Artifacts: mockSP1VerifierArtifacts(artifact)})
			require.ErrorContains(t, err, "MockSP1Verifier needs its init code")
			require.NotContains(t, genesis.Allocs.Accounts, common.Address{0x5e}, "the refused deployment used a nonce")
		})
	}
}
