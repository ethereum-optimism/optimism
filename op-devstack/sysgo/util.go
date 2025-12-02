package sysgo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
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

// getAvailableLocalPort searches for and returns a currently unused local port.
// Note: this function is threadsafe.
func getAvailableLocalPort() (string, error) {
	availableLocalPortMutex.Lock()
	defer availableLocalPortMutex.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("could not listen on ephemeral port: %w", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return "", errors.New("listener did not return a TCP addr")
	}
	return strconv.Itoa(addr.Port), nil
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

// waitWSReady attempts an actual WebSocket handshake to confirm readiness using EventuallyWithT.
func waitWSReady(p devtest.P, rawURL string, timeout time.Duration) {
	p.Helper()
	waitWSMsg := fmt.Sprintf("WebSocket endpoint %s not ready within %v", rawURL, timeout)
	p.Require().EventuallyWithT(func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
		conn, resp, err := websocket.Dial(ctx, rawURL, nil)
		cancel()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if conn != nil {
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}
		assert.NoError(c, err, "WebSocket handshake to %s should succeed", rawURL)
	}, timeout, 100*time.Millisecond, waitWSMsg)
}
