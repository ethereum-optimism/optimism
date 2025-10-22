package sysgo

import (
	"fmt"
	"net"
	"os"
)

func GetEnvVarOrDefault(envVarName string, defaultValue string) string {
	val := os.Getenv(envVarName)
	if val == "" {
		val = defaultValue
	}
	return val
}

func PropagateEnvVarOrDefault(envVarName string, defaultValue string) string {
	if val := GetEnvVarOrDefault(envVarName, defaultValue); val == "" {
		return ""
	} else {
		return fmt.Sprintf("%s=%s", envVarName, val)
	}
}

// NB: arbitrary start port with a low probability of conflict
var availableLocalPortStart = 20_000

func GetAvailableLocalPort() string {
	port := availableLocalPortStart
	for {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			port++
		}
		_ = ln.Close()
		availableLocalPortStart = port + 1
		return fmt.Sprintf("%d", port)
	}

}
