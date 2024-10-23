package metrics

import "time"

type VmMetricer interface {
	RecordVmExecutionTime(vmType string, t time.Duration)
	RecordVmMemoryUsed(vmType string, memoryUsed uint64)
}

type VmMetrics struct {
	m      VmMetricer
	vmType string
}

func NewVmMetrics(m VmMetricer, vmType string) *VmMetrics {
	return &VmMetrics{
		m:      m,
		vmType: vmType,
	}
}

func (m *VmMetrics) RecordExecutionTime(dur time.Duration) {
	m.m.RecordVmExecutionTime(m.vmType, dur)
}

func (m *VmMetrics) RecordMemoryUsed(memoryUsed uint64) {
	m.m.RecordVmMemoryUsed(m.vmType, memoryUsed)
}
