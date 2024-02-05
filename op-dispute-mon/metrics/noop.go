package metrics

type NoopMetricsImpl struct{}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string) {}
func (*NoopMetricsImpl) RecordUp()                 {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}
