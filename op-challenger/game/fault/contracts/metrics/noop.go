package metrics

type NoopMetrics struct {
}

func (n *NoopMetrics) StartContractRequest(_ string) EndTimer {
	return func() {}
}

var _ ContractMetricer = (*NoopMetrics)(nil)

var NoopContractMetrics = &NoopMetrics{}
