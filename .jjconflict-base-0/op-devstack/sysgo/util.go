package sysgo

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
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

var availableLocalPortMutex sync.Mutex
var recentlyAllocatedPorts = make(map[int]struct{})

// getAvailableLocalPort searches for and returns a currently unused local port.
// Tracks recently allocated ports to avoid returning the same port twice
// (the OS may recycle a port immediately after we release it).
// Note: this function is threadsafe.
func getAvailableLocalPort() (string, error) {
	availableLocalPortMutex.Lock()
	defer availableLocalPortMutex.Unlock()

	// Keep listeners open while looping so the OS won't return the same port twice
	var heldListeners []net.Listener
	defer func() {
		for _, ln := range heldListeners {
			ln.Close()
		}
	}()

	const maxAttempts = 100
	for range maxAttempts {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("could not listen on ephemeral port: %w", err)
		}
		heldListeners = append(heldListeners, ln)

		addr, ok := ln.Addr().(*net.TCPAddr)
		if !ok {
			return "", errors.New("listener did not return a TCP addr")
		}
		port := addr.Port

		if _, used := recentlyAllocatedPorts[port]; used {
			continue
		}
		recentlyAllocatedPorts[port] = struct{}{}
		return strconv.Itoa(port), nil
	}
	return "", errors.New("failed to allocate unique port after max attempts")
}

// waitTCPReady parses a URL and waits for its TCP endpoint to become ready using EventuallyWithT.
func waitTCPReady(p devtest.P, rawURL string, timeout time.Duration) {
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
// This is used to parse addresses from process (e.g. op-rbuilder) log output.
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
