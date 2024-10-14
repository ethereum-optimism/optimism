//go:build cannon64
// +build cannon64

package tests

import (
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func FuzzStateDmultInsn(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, rs arch.Word, rt arch.Word, seed int64) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(seed), testutil.WithPC(0), testutil.WithNextPC(4))
				state := goVm.GetState()
				var baseReg uint32 = 17
				var rtReg uint32 = 18
				// dmult s1, s2
				insn := baseReg<<21 | rtReg<<16 | 0x1c
				state.GetRegistersRef()[baseReg] = rs
				state.GetRegistersRef()[rtReg] = rt
				state.GetMemory().SetUint32(0, insn)
				step := state.GetStep()

				// Setup expectations
				sanity := new(big.Int).Mul(big.NewInt(int64(rs)), big.NewInt(int64(rt)))
				mask := new(big.Int).Lsh(big.NewInt(1), 64)
				mask.Sub(mask, big.NewInt(1))
				expectLo := new(big.Int).And(sanity, mask).Uint64()
				expectHi := new(big.Int).Rsh(sanity, 64).Uint64()

				expected := testutil.NewExpectedState(state)
				expected.ExpectStep()
				expected.LO = expectLo
				expected.HI = expectHi

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, nil)
			})
		}
	})
}
