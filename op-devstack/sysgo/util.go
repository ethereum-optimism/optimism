package sysgo

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/stretchr/testify/assert"
)

// getEnvVarOrDefault returns the value of the provided env var or the provided default value if unset.
func getEnvVarOrDefault(envVarName string, defaultValue string) string {
	val, found := os.LookupEnv(envVarName)
	if !found {
		val = defaultValue
	}
	return val
}

// propagateEnvVarOrDefault returns a string in the format "ENV_VAR_NAME=VALUE", with the ENV_VAR_NAME being
// the provided env var name and the value being the value of that env var, or the provided default
// value if that env var is unset.
func propagateEnvVarOrDefault(envVarName string, defaultValue string) string {
	if val := getEnvVarOrDefault(envVarName, defaultValue); val == "" {
		return ""
	} else {
		return fmt.Sprintf("%s=%s", envVarName, val)
	}
}

// waitTCPReady parses a URL and waits for its TCP endpoint to become ready using EventuallyWithT.
func waitTCPReady(p devtest.CommonT, rawURL string, timeout time.Duration) {
	p.Helper()
	u, err := url.Parse(rawURL)
	p.Require().NoError(err, "parse URL: %s", rawURL)
	p.Require().NotEmpty(u.Host, "URL has no host: %s", rawURL)
	waitMsg := fmt.Sprintf("TCP endpoint %s not ready within %v", u.Host, timeout)
	p.Require().EventuallyWithT(func(c *assert.CollectT) {
		conn, err := net.DialTimeout("tcp", u.Host, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
		}
		assert.NoError(c, err, "TCP connection to %s should succeed", u.Host)
	}, timeout, 100*time.Millisecond, waitMsg)
}
