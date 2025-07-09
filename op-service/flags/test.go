package flags

import (
	"flag"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

var flLoadTest = flag.Bool("loadtest", false, "Enable load tests during test run")

type TestConfig struct {
	LogConfig       log.CLIConfig
	EnableLoadTests bool
}

func ReadTestConfig() TestConfig {
	flag.Parse()

	loadTest := *flLoadTest
	if v := os.Getenv("NAT_LOADTEST"); v != "" {
		loadTest = v == "true"
	}
	cfg := log.ReadTestCLIConfig()

	return TestConfig{
		EnableLoadTests: loadTest,
		LogConfig:       cfg,
	}
}
