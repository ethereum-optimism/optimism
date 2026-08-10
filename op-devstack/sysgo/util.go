package sysgo

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
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

// parseAndValidateAddr ensures the address has a scheme and is a valid URL.
// Returns the validated URL string or empty string if invalid.
// This is used to parse addresses from child-process log output.
func parseAndValidateAddr(addr, defaultScheme string) string {
	if addr == "" {
		return ""
	}
	// Add scheme if not present
	if !strings.Contains(addr, "://") {
		addr = defaultScheme + "://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return ""
	}
	return u.String()
}

func parseBoundAddressLog(message, prefix, defaultScheme string) (string, bool) {
	addr, found := strings.CutPrefix(message, prefix)
	if !found {
		return "", false
	}
	validURL := parseAndValidateAddr(addr, defaultScheme)
	if validURL == "" {
		return "", false
	}
	u, err := url.Parse(validURL)
	if err != nil {
		return "", false
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil || port == 0 {
		return "", false
	}
	return validURL, true
}
