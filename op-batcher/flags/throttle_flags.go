package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	// Block-builder side
	AdditionalThrottlingEndpointsFlag = &cli.StringSliceFlag{
		Name:    "throttle.additional-endpoints",
		Usage:   "Comma-separated list of endpoints to distribute throttling configuration to (in addition to the L2 endpoints specified with --l2-eth-rpc).",
		EnvVars: prefixEnvVars("THROTTLE_ADDITIONAL_ENDPOINTS"),
	}

	// Builder-side Tx-size limits
	ThrottleTxSizeLowerLimitFlag = &cli.IntFlag{
		Name:    "throttle.tx-size-lower-limit",
		Usage:   "The limit on the DA size of transactions when we are at maximum throttle intensity",
		Value:   5000, // less than 1% of all transactions should be affected by this limit
		EnvVars: prefixEnvVars("THROTTLE_TX_SIZE_LOWER_LIMIT"),
	}
	ThrottleTxSizeUpperLimitFlag = &cli.IntFlag{
		Name:    "throttle.tx-size-upper-limit",
		Usage:   "The limit on the DA size of transactions when we are at 0+ throttle intensity (limit of the intensity as it approaches 0 from positive values)",
		Value:   5000, // less than 1% of all transactions should be affected by this limit
		EnvVars: prefixEnvVars("THROTTLE_TX_SIZE_UPPER_LIMIT"),
	}

	// Builder-side block-size limits
	ThrottleBlockSizeLowerLimitFlag = &cli.IntFlag{
		Name:    "throttle.block-size-lower-limit",
		Usage:   "The limit on the DA size of blocks when we are at maximum throttle intensity (linear and quadratic controllers only)",
		Value:   21_000, // at least 70 transactions per block of up to 300 compressed bytes each.
		EnvVars: prefixEnvVars("THROTTLE_BLOCK_SIZE_LOWER_LIMIT"),
	}
	ThrottleBlockSizeUpperLimitFlag = &cli.IntFlag{
		Name:    "throttle.block-size-upper-limit",
		Usage:   "The limit on the DA size of blocks when we are at 0 throttle intensity",
		Value:   21_000, // at least 70 transactions per block of up to 300 compressed bytes each.
		EnvVars: prefixEnvVars("THROTTLE_BLOCK_SIZE_UPPER_LIMIT"),
	}

	// // Controller side
	ThrottleControllerTypeFlag = &cli.StringFlag{
		Name:    "throttle.controller-type",
		Usage:   "Type of throttle controller to use: 'step', 'linear', 'quadratic' (default) or 'pid' (EXPERIMENTAL - use with caution)",
		Value:   "quadratic",
		EnvVars: prefixEnvVars("THROTTLE_CONTROLLER_TYPE"),
		Action: func(ctx *cli.Context, value string) error {
			validTypes := []string{"step", "linear", "quadratic", "pid"}
			for _, validType := range validTypes {
				if value == validType {
					return nil
				}
			}
			return fmt.Errorf("throttle-controller-type must be one of %v, got %s", validTypes, value)
		},
	}
	ThrottleUsafeDABytesLowerThresholdFlag = &cli.IntFlag{
		Name:    "throttle.unsafe-da-bytes-lower-threshold",
		Usage:   "The threshold on unsafe_da_bytes beyond which the batcher will start to throttle the block builder. Zero disables throttling.",
		Value:   1_000_000,
		EnvVars: prefixEnvVars("THROTTLE_LOWER_THRESHOLD"),
	}
	ThrottleUsafeDABytesUpperThresholdFlag = &cli.Uint64Flag{
		Name:    "throttle.unsafe-da-bytes-upper-threshold",
		Usage:   "Threshold on unsafe_da_bytes at which throttling has the maximum intensity (linear and quadratic controllers only)",
		Value:   DefaultThrottleMaxThreshold,
		EnvVars: prefixEnvVars("THROTTLE_UPPER_THRESHOLD"),
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
	ThrottleUsafeDABytesLowerThresholdFlag,
	ThrottleUsafeDABytesUpperThresholdFlag,
}
