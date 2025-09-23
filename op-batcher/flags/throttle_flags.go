package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	// Block-builder side
	DefaultThrottleTxSizeLowerLimit    = 150
	DefaultThrottleTxSizeUpperLimit    = 20_000
	DefaultThrottleBlockSizeLowerLimit = 2_000
	DefaultThrottleBlockSizeUpperLimit = 130_000

	// Controller side
	DefaultThrottleControllerType = "quadratic"
	DefaultThrottleLowerThreshold = 3_200_000 // allows for 4x 6-blob-tx channels at ~131KB per blob
	DefaultThrottleUpperThreshold = DefaultThrottleLowerThreshold * 4
	DefaultPIDSampleTime          = 2 * time.Second
	DefaultPIDKp                  = 0.33
	DefaultPIDKi                  = 0.01
	DefaultPIDKd                  = 0.05
	DefaultPIDIntegralMax         = 1000.0
	DefaultPIDOutputMax           = 1.0
)

var (
	// Block-builder side
	AdditionalThrottlingEndpointsFlag = &cli.StringSliceFlag{
		Name:    "throttle.additional-endpoints",
		Usage:   "Comma-separated list of endpoints to distribute throttling configuration to (in addition to the L2 endpoints specified with --l2-eth-rpc).",
		EnvVars: prefixEnvVars("THROTTLE_ADDITIONAL_ENDPOINTS"),
	}

	// Builder-side Tx-size limits
	ThrottleTxSizeLowerLimitFlag = &cli.Uint64Flag{
		Name:    "throttle.tx-size-lower-limit",
		Usage:   "The limit on the DA size of transactions when we are at maximum throttle intensity. 0 means no limits will ever be applied, so consider 1 the smallest effective limit.",
		Value:   DefaultThrottleTxSizeLowerLimit,
		EnvVars: prefixEnvVars("THROTTLE_TX_SIZE_LOWER_LIMIT"),
	}
	ThrottleTxSizeUpperLimitFlag = &cli.Uint64Flag{
		Name:    "throttle.tx-size-upper-limit",
		Usage:   "The limit on the DA size of transactions when we are at 0+ throttle intensity (limit of the intensity as it approaches 0 from positive values). Not applied when throttling is inactive.",
		Value:   DefaultThrottleTxSizeUpperLimit,
		EnvVars: prefixEnvVars("THROTTLE_TX_SIZE_UPPER_LIMIT"),
	}

	// Builder-side block-size limits
	ThrottleBlockSizeLowerLimitFlag = &cli.Uint64Flag{
		Name:    "throttle.block-size-lower-limit",
		Usage:   "The limit on the DA size of blocks when we are at maximum throttle intensity (linear and quadratic controllers only). 0 means no limits will ever be applied, so consider 1 the smallest effective limit.",
		Value:   DefaultThrottleBlockSizeLowerLimit,
		EnvVars: prefixEnvVars("THROTTLE_BLOCK_SIZE_LOWER_LIMIT"),
	}
	ThrottleBlockSizeUpperLimitFlag = &cli.Uint64Flag{
		Name:    "throttle.block-size-upper-limit",
		Usage:   "The limit on the DA size of blocks when we are at 0 throttle intensity (applied when throttling is inactive)",
		Value:   DefaultThrottleBlockSizeUpperLimit,
		EnvVars: prefixEnvVars("THROTTLE_BLOCK_SIZE_UPPER_LIMIT"),
	}

	// // Controller side
	ThrottleControllerTypeFlag = &cli.StringFlag{
		Name:    "throttle.controller-type",
		Usage:   "Type of throttle controller to use: 'step', 'linear', 'quadratic' (default) or 'pid' (EXPERIMENTAL - use with caution)",
		Value:   DefaultThrottleControllerType,
		EnvVars: prefixEnvVars("THROTTLE_CONTROLLER_TYPE"),
		Action: func(ctx *cli.Context, value string) error {
			validTypes := []string{"step", "linear", "quadratic", "pid"}
			for _, validType := range validTypes {
				if value == validType {
					return nil
				}
			}
			return fmt.Errorf("throttle.controller-type must be one of %v, got %s", validTypes, value)
		},
	}
	ThrottleUsafeDABytesLowerThresholdFlag = &cli.Uint64Flag{
		Name:    "throttle.unsafe-da-bytes-lower-threshold",
		Usage:   "The threshold on unsafe_da_bytes beyond which the batcher will start to throttle the block builder. Zero disables throttling.",
		Value:   DefaultThrottleLowerThreshold,
		EnvVars: prefixEnvVars("THROTTLE_UNSAFE_DA_BYTES_LOWER_THRESHOLD"),
	}
	ThrottleUsafeDABytesUpperThresholdFlag = &cli.Uint64Flag{
		Name:    "throttle.unsafe-da-bytes-upper-threshold",
		Usage:   "Threshold on unsafe_da_bytes at which throttling has the maximum intensity (linear and quadratic controllers only)",
		Value:   DefaultThrottleUpperThreshold,
		EnvVars: prefixEnvVars("THROTTLE_UNSAFE_DA_BYTES_UPPER_THRESHOLD"),
	}

	// Controller side (EXPERIMENTAL PID Controller only)
	ThrottlePidKpFlag = &cli.Float64Flag{
		Name:    "throttle.pid-kp",
		Usage:   "EXPERIMENTAL: PID controller proportional gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKp,
		EnvVars: prefixEnvVars("THROTTLE_PID_KP"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-kp must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidKiFlag = &cli.Float64Flag{
		Name:    "throttle.pid-ki",
		Usage:   "EXPERIMENTAL: PID controller integral gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKi,
		EnvVars: prefixEnvVars("THROTTLE_PID_KI"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-ki must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidKdFlag = &cli.Float64Flag{
		Name:    "throttle.pid-kd",
		Usage:   "EXPERIMENTAL: PID controller derivative gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKd,
		EnvVars: prefixEnvVars("THROTTLE_PID_KD"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-kd must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidIntegralMaxFlag = &cli.Float64Flag{
		Name:    "throttle.pid-integral-max",
		Usage:   "EXPERIMENTAL: PID controller maximum integral windup. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDIntegralMax,
		EnvVars: prefixEnvVars("THROTTLE_PID_INTEGRAL_MAX"),
		Action: func(ctx *cli.Context, value float64) error {
			if value <= 0 {
				return fmt.Errorf("throttle-pid-integral-max must be > 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidOutputMaxFlag = &cli.Float64Flag{
		Name:    "throttle.pid-output-max",
		Usage:   "EXPERIMENTAL: PID controller maximum output. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDOutputMax,
		EnvVars: prefixEnvVars("THROTTLE_PID_OUTPUT_MAX"),
		Action: func(ctx *cli.Context, value float64) error {
			if value <= 0 || value > 1.0 {
				return fmt.Errorf("throttle-pid-output-max must be between 0 and 1, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidSampleTimeFlag = &cli.DurationFlag{
		Name:    "throttle.pid-sample-time",
		Usage:   "EXPERIMENTAL: PID controller sample time interval, default is " + DefaultPIDSampleTime.String(),
		Value:   DefaultPIDSampleTime,
		EnvVars: prefixEnvVars("THROTTLE_PID_SAMPLE_TIME"),
	}
)

var ThrottleFlags = []cli.Flag{
	AdditionalThrottlingEndpointsFlag,
	ThrottleTxSizeLowerLimitFlag,
	ThrottleTxSizeUpperLimitFlag,
	ThrottleBlockSizeLowerLimitFlag,
	ThrottleBlockSizeUpperLimitFlag,
	ThrottleControllerTypeFlag,
	ThrottleUsafeDABytesLowerThresholdFlag,
	ThrottleUsafeDABytesUpperThresholdFlag,
	// PID controller flags (only used when controller type is 'pid')
	ThrottlePidKpFlag,
	ThrottlePidKiFlag,
	ThrottlePidKdFlag,
	ThrottlePidIntegralMaxFlag,
	ThrottlePidOutputMaxFlag,
	ThrottlePidSampleTimeFlag,
}
