package config

import "time"

// ThrottleControllerType represents the type of throttle controller
type ThrottleControllerType string

const (
	StepControllerType      ThrottleControllerType = "step"
	QuadraticControllerType ThrottleControllerType = "quadratic"
	PIDControllerType       ThrottleControllerType = "pid"
)

// String returns the string representation of ThrottleControllerType
func (t ThrottleControllerType) String() string {
	return string(t)
}

// ThrottleControllerInfo represents throttle controller information
type ThrottleControllerInfo struct {
	Type         string  `json:"type"`
	Threshold    uint64  `json:"threshold"`
	CurrentLoad  uint64  `json:"current_load"`
	Intensity    float64 `json:"intensity"`
	MaxTxSize    uint64  `json:"max_tx_size"`
	MaxBlockSize uint64  `json:"max_block_size"`
}

// PIDConfig represents PID controller configuration for RPC
type PIDConfig struct {
	Kp          float64       `json:"kp"`
	Ki          float64       `json:"ki"`
	Kd          float64       `json:"kd"`
	IntegralMax float64       `json:"integral_max"`
	OutputMax   float64       `json:"output_max"`
	SampleTime  time.Duration `json:"sample_time"`
}
