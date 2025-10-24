package sysgo

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// GetEnvVarOrDefault returns the value of the provided env var or the provided default value if unset or empty.
func GetEnvVarOrDefault(envVarName string, defaultValue string) string {
	val := os.Getenv(envVarName)
	if val == "" {
		val = defaultValue
	}
	return val
}

// PropagateEnvVarOrDefault returns a string in the format "ENV_VAR_NAME=VALUE", with the ENV_VAR_NAME being
// the provided env var name and the value being the value of that env var, or the provided default
// value if that env var is unset or empty
func PropagateEnvVarOrDefault(envVarName string, defaultValue string) string {
	if val := GetEnvVarOrDefault(envVarName, defaultValue); val == "" {
		return ""
	} else {
		return fmt.Sprintf("%s=%s", envVarName, val)
	}
}

// NB: arbitrary start port with a low probability of conflict
var availableLocalPortStart = 20_000
var availableLocalPortMutex sync.Mutex

// GetAvailableLocalPort searches for and returns a currently unused local port.
// Note: this function is threadsafe.
func GetAvailableLocalPort() string {
	availableLocalPortMutex.Lock()
	defer availableLocalPortMutex.Unlock()

	for port := availableLocalPortStart; port < 65_535; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		availableLocalPortStart = port + 1
		return fmt.Sprintf("%d", port)
	}

	// NB: this is a hack, but it should never happen, so don't return & check for error on every call.
	return "[should never happen] no available port found... this will err when you try to use it"
}
