package sysgo

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
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

// NB: arbitrary start port with a low probability of conflict
var availableLocalPortStart = 20_000
var availableLocalPortMutex sync.Mutex

// getAvailableLocalPort searches for and returns a currently unused local port.
// Note: this function is threadsafe.
func getAvailableLocalPort() (string, error) {
	availableLocalPortMutex.Lock()
	defer availableLocalPortMutex.Unlock()

	for port := availableLocalPortStart; port < 65_535; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		availableLocalPortStart = port + 1
		return fmt.Sprintf("%d", port), nil
	}

	return "", errors.New("could not find open port")
}
