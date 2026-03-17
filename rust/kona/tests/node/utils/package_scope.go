package node_utils

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// packageScopeT adapts a package-scoped devtest.P so sysgo runtimes can be
// initialized once in TestMain and shared across the package.
type packageScopeT struct {
	p    devtest.P
	gate *testreq.Assertions
}

type packageGateAdapter struct {
	inner interface {
		Helper()
		Skipf(format string, args ...any)
		SkipNow()
	}
}

func (g *packageGateAdapter) Errorf(format string, args ...interface{}) {
	g.inner.Helper()
	g.inner.Skipf(format, args...)
}

func (g *packageGateAdapter) FailNow() {
	g.inner.Helper()
	g.inner.SkipNow()
}

func (g *packageGateAdapter) Helper() {
	g.inner.Helper()
}

func newPackageScopeT(p devtest.P) *packageScopeT {
	t := &packageScopeT{p: p}
	t.gate = testreq.New(&packageGateAdapter{inner: t})
	return t
}

func (t *packageScopeT) Error(args ...any) {
	t.p.Error(args...)
}

func (t *packageScopeT) Errorf(format string, args ...any) {
	t.p.Errorf(format, args...)
}

func (t *packageScopeT) Fail() {
	t.p.Fail()
}

func (t *packageScopeT) FailNow() {
	t.p.FailNow()
}

func (t *packageScopeT) TempDir() string {
	return t.p.TempDir()
}

func (t *packageScopeT) Cleanup(fn func()) {
	t.p.Cleanup(fn)
}

func (t *packageScopeT) Run(_ string, fn func(devtest.T)) {
	fn(newPackageScopeT(t.p))
}

func (t *packageScopeT) Ctx() context.Context {
	return t.p.Ctx()
}

func (t *packageScopeT) WithCtx(ctx context.Context) devtest.T {
	return newPackageScopeT(t.p.WithCtx(ctx))
}

func (t *packageScopeT) Parallel() {
}

func (t *packageScopeT) Skip(args ...any) {
	t.Helper()
	t.Log(args...)
	t.SkipNow()
}

func (t *packageScopeT) Skipped() bool {
	return false
}

func (t *packageScopeT) Skipf(format string, args ...any) {
	t.Helper()
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *packageScopeT) SkipNow() {
	t.p.SkipNow()
}

func (t *packageScopeT) Log(args ...any) {
	t.p.Log(args...)
}

func (t *packageScopeT) Logf(format string, args ...any) {
	t.p.Logf(format, args...)
}

func (t *packageScopeT) Helper() {
	t.p.Helper()
}

func (t *packageScopeT) Name() string {
	return t.p.Name()
}

func (t *packageScopeT) Logger() log.Logger {
	return t.p.Logger()
}

func (t *packageScopeT) Tracer() trace.Tracer {
	return t.p.Tracer()
}

func (t *packageScopeT) Require() *testreq.Assertions {
	return t.p.Require()
}

func (t *packageScopeT) Gate() *testreq.Assertions {
	return t.gate
}

func (t *packageScopeT) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (t *packageScopeT) TestOnly() {
}

var _ devtest.T = (*packageScopeT)(nil)

func NewSharedMixedOpKona(pkg devtest.P) *MixedOpKonaPreset {
	return NewSharedMixedOpKonaForConfig(pkg, ParseL2NodeConfigFromEnv())
}

func NewSharedMixedOpKonaForConfig(pkg devtest.P, cfg L2NodeConfig) *MixedOpKonaPreset {
	return NewMixedOpKonaFromRuntime(newPackageScopeT(pkg), NewSharedMixedOpKonaRuntimeForConfig(pkg, cfg))
}

func NewSharedMixedOpKonaRuntime(pkg devtest.P) *sysgo.MixedSingleChainRuntime {
	return NewSharedMixedOpKonaRuntimeForConfig(pkg, ParseL2NodeConfigFromEnv())
}

func NewSharedMixedOpKonaRuntimeForConfig(pkg devtest.P, cfg L2NodeConfig) *sysgo.MixedSingleChainRuntime {
	return sysgo.NewMixedSingleChainRuntime(newPackageScopeT(pkg), sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: mixedOpKonaNodeSpecs(cfg),
	})
}
