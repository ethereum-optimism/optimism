package flags

import (
	"flag"

	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

type TestConfig struct {
	LogConfig logcli.CLIConfig
}

func ReadTestConfig() TestConfig {
	flag.Parse()

	cfg := logcli.ReadTestCLIConfig()

	return TestConfig{
		LogConfig: cfg,
	}
}
