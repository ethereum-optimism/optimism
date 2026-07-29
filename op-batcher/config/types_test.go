package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUmarshalPIDConfig(t *testing.T) {
	rawJson := `{"kp": 0.3, "ki": 0.3, "kd": 0.008, "integral_max": 1000.0, "output_max": 1.0, "sample_time": "200ms"}`
	var pidConfig PIDConfig
	err := json.Unmarshal([]byte(rawJson), &pidConfig)
	require.NoError(t, err)
	require.Equal(t, 0.3, pidConfig.Kp)
	require.Equal(t, 0.3, pidConfig.Ki)
	require.Equal(t, 0.008, pidConfig.Kd)
	require.Equal(t, 1000.0, pidConfig.IntegralMax)
	require.Equal(t, 1.0, pidConfig.OutputMax)
	require.Equal(t, 200*time.Millisecond, pidConfig.SampleTime)

	rawJson = `{"sample_time": "1s"}`
	err = json.Unmarshal([]byte(rawJson), &pidConfig)
	require.NoError(t, err)
	require.Equal(t, 1*time.Second, pidConfig.SampleTime)
}
