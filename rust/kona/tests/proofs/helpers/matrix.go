package helpers

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum/go-ethereum/common"
)

type RunTest[cfg any] func(t *testing.T, testCfg *TestCfg[cfg])

type TestCfg[cfg any] struct {
	Hardfork    *Hardfork
	CheckResult CheckResult
	InputParams []FixtureInputParam
	Custom      cfg
	Allocs      *e2eutils.AllocParams
}

type TestCase[cfg any] struct {
	Name        string
	Cfg         cfg
	ForkMatrix  ForkMatrix
	RunTest     RunTest[cfg]
	InputParams []FixtureInputParam
	CheckResult CheckResult
}

type TestMatrix[cfg any] struct {
	CommonInputParams []FixtureInputParam
	TestCases         []TestCase[cfg]
}

func (suite *TestMatrix[cfg]) Run(t *testing.T) {
	for _, tc := range suite.TestCases {
		for _, fork := range tc.ForkMatrix {
			t.Run(fmt.Sprintf("%s-%s", tc.Name, fork.Name), func(t *testing.T) {
				SkipKonaProofActionTest(t)

				testCfg := &TestCfg[cfg]{
					Hardfork:    fork,
					CheckResult: tc.CheckResult,
					InputParams: append(suite.CommonInputParams, tc.InputParams...),
					Custom:      tc.Cfg,
				}
				tc.RunTest(t, testCfg)
			})
		}
	}
}

const konaProofActionTestSkipMessage = "TODO: port these op-geth action tests to op-acceptance-tests/op-reth; kona proof hosts no longer support legacy op-geth witness RPC assumptions"

// SkipKonaProofActionTest skips kona proof action tests before they build an op-geth action-test
// chain that no longer matches the witness RPCs required by the kona proof hosts.
func SkipKonaProofActionTest(t *testing.T) {
	t.Helper()

	if os.Getenv("KONA_HOST_PATH") != "" || os.Getenv("KONA_SP1_RANGE_EXECUTOR_PATH") != "" {
		t.Skip(konaProofActionTestSkipMessage)
	}
}

func NewMatrix[cfg any]() *TestMatrix[cfg] {
	return &TestMatrix[cfg]{}
}

func (ts *TestMatrix[cfg]) WithCommonInputParams(params ...FixtureInputParam) *TestMatrix[cfg] {
	ts.CommonInputParams = params
	return ts
}

func (ts *TestMatrix[cfg]) AddTestCase(
	name string,
	testCfg cfg,
	forkMatrix ForkMatrix,
	runTest RunTest[cfg],
	checkResult CheckResult,
	inputParams ...FixtureInputParam,
) *TestMatrix[cfg] {
	ts.TestCases = append(ts.TestCases, TestCase[cfg]{
		Name:        name,
		Cfg:         testCfg,
		ForkMatrix:  forkMatrix,
		RunTest:     runTest,
		InputParams: inputParams,
		CheckResult: checkResult,
	})
	return ts
}

func (ts *TestMatrix[cfg]) AddDefaultTestCases(
	testCfg cfg,
	forkMatrix ForkMatrix,
	runTest RunTest[cfg],
) *TestMatrix[cfg] {
	return ts.AddDefaultTestCasesWithName("", testCfg, forkMatrix, runTest)
}

func (ts *TestMatrix[cfg]) AddDefaultTestCasesWithName(
	name string,
	testCfg cfg,
	forkMatrix ForkMatrix,
	runTest RunTest[cfg],
) *TestMatrix[cfg] {
	return ts.AddTestCase(
		"HonestClaim-"+name,
		testCfg,
		forkMatrix,
		runTest,
		ExpectNoError(),
	).AddTestCase(
		"JunkClaim-"+name,
		testCfg,
		forkMatrix,
		runTest,
		ExpectError(ErrClaimNotValid),
		WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}

type Hardfork struct {
	Name       string
	Precedence int
}

type ForkMatrix = []*Hardfork

// Hardfork definitions
var (
	Regolith = &Hardfork{Name: string(forks.Regolith), Precedence: 1}
	Canyon   = &Hardfork{Name: string(forks.Canyon), Precedence: 2}
	Delta    = &Hardfork{Name: string(forks.Delta), Precedence: 3}
	Ecotone  = &Hardfork{Name: string(forks.Ecotone), Precedence: 4}
	Fjord    = &Hardfork{Name: string(forks.Fjord), Precedence: 5}
	Granite  = &Hardfork{Name: string(forks.Granite), Precedence: 6}
	Holocene = &Hardfork{Name: string(forks.Holocene), Precedence: 7}
	Isthmus  = &Hardfork{Name: string(forks.Isthmus), Precedence: 8}
	Jovian   = &Hardfork{Name: string(forks.Jovian), Precedence: 9}
	Karst    = &Hardfork{Name: string(forks.Karst), Precedence: 10}
)

var (
	Hardforks      = ForkMatrix{Regolith, Canyon, Delta, Ecotone, Fjord, Granite, Holocene, Isthmus, Jovian, Karst}
	LatestFork     = Hardforks[len(Hardforks)-1]
	LatestForkOnly = ForkMatrix{LatestFork}
)

func NewForkMatrix(forks ...*Hardfork) ForkMatrix {
	return append(ForkMatrix{}, forks...)
}

func FaultProofForks() ForkMatrix {
	var forks ForkMatrix
	for _, hf := range Hardforks {
		if hf == Regolith || hf == Canyon || hf == Delta {
			continue
		}
		forks = append(forks, hf)
	}
	return forks
}
