package utils

import (
	"net"
	"time"
	"fmt"
	"bytes"
	"strings"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

// PrefixIfMissing adds a prefix to a string if it is missing.
func PrefixIfMissing(s string, prefix string) string {
	if !strings.HasPrefix(s, prefix) {
		return prefix + s
	}
	return s
}

// WaitUp waits until the endpoint is up.
func WaitUp(endpoint string, retries int, delay time.Duration) error {
	log.Info("Waiting for endpoint", "endpoint", endpoint, "retries", retries)

	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", endpoint, delay)
		if err == nil && conn != nil {
			conn.Close()
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("failed to connect to %s after %d retries", endpoint, retries)
}

// WaitForRPC waits until an RPC server is up at the specified endpoint.
func WaitForRPC(endpoint string, retries int, delay time.Duration) error {
	log.Info("Waiting for RPC server", "endpoint", endpoint, "retries", retries)

	for i := 0; i < retries; i++ {
		client := &http.Client{Timeout: 10 * time.Second}
		body := []byte(`{"id":"1","jsonrpc":"2.0","method":"eth_chainId","params":[]}`)
		req, err := http.NewRequest("POST", PrefixIfMissing(endpoint, "http://"), bytes.NewReader(body))
		if err != nil {
			time.Sleep(delay)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			time.Sleep(delay)
			continue
		}

		if res.StatusCode < 300 {
			return nil
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("failed to connect to %s after %d retries", endpoint, retries)
}
