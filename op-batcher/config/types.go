package config

import (
	"encoding/json"
	"slices"
	"time"
)

// ThrottleControllerType represents the type of throttle controller
type ThrottleControllerType string

const (
	StepControllerType      ThrottleControllerType = "step"
	LinearControllerType    ThrottleControllerType = "linear"
	QuadraticControllerType ThrottleControllerType = "quadratic"
	PIDControllerType       ThrottleControllerType = "pid"
)

var ThrottleControllerTypes = []ThrottleControllerType{
	StepControllerType,
	LinearControllerType,
	QuadraticControllerType,
	PIDControllerType,
}

func ValidThrottleControllerType(value ThrottleControllerType) bool {
	return slices.Contains(ThrottleControllerTypes, value)
}

// String returns the string representation of ThrottleControllerType
func (t ThrottleControllerType) String() string {
	return string(t)
}

// ThrottleControllerInfo represents throttle controller information
type ThrottleControllerInfo struct {
	Type           string  `json:"type"`
	LowerThreshold uint64  `json:"lower_threshold"`
	UpperThreshold uint64  `json:"upper_threshold"`
	CurrentLoad    uint64  `json:"current_load"`
	Intensity      float64 `json:"intensity"`
	MaxTxSize      uint64  `json:"max_tx_size"`
	MaxBlockSize   uint64  `json:"max_block_size"`
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

// UnmarshalJSON implements custom JSON unmarshaling for PIDConfig to handle duration strings
func (p *PIDConfig) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with SampleTime as string to handle the duration parsing
	type pidConfigAlias struct {
		Kp          float64 `json:"kp"`
		Ki          float64 `json:"ki"`
		Kd          float64 `json:"kd"`
		IntegralMax float64 `json:"integral_max"`
		OutputMax   float64 `json:"output_max"`
		SampleTime  string  `json:"sample_time"`
	}

	var alias pidConfigAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Parse the duration string
	duration, err := time.ParseDuration(alias.SampleTime)
	if err != nil {
		return err
	}

	// Assign values to the actual struct
	p.Kp = alias.Kp
	p.Ki = alias.Ki
	p.Kd = alias.Kd
	p.IntegralMax = alias.IntegralMax
	p.OutputMax = alias.OutputMax
	p.SampleTime = duration

	return nil
}

type ThrottleParams struct {
	LowerThreshold      uint64
	UpperThreshold      uint64
	TxSizeLowerLimit    uint64
	TxSizeUpperLimit    uint64
	BlockSizeLowerLimit uint64
	BlockSizeUpperLimit uint64
	PIDConfig           *PIDConfig
	ControllerType      ThrottleControllerType
	Endpoints           []string
}
