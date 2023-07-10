package devnet

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// WriteFile writes a file.
func WriteFile(path string, data []byte) {
	err := ioutil.WriteFile(path, data, 0644)
	if err != nil {
		log.Crit("Failed to write json file", "err", err)
	}
}

// ReadFile reads a file.
func ReadFile(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Crit("Failed to read json file", "err", err)
	}
	return content
}

// RunCommand executes a command and logs the output.
func RunCommand(cmds []string, envs []string, cwd string) error {
	cmd := exec.Command("docker-compose", "build", "--progress", "plain")
	cmd.Env = append(os.Environ(), envs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type WaitOpts struct {
	retries int
	timeout time.Duration
}

// WaitUp waits for a service to be up.
// opts are optional parameters and do not need to be provided.
func WaitUp(port int, opts WaitOpts) {
	retries := 10
	if opts.retries != 0 {
		retries = opts.retries
	}
	timeout := 1 * time.Second
	if opts.timeout != 0 {
		timeout = opts.timeout
	}
	host := "127.0.0.1"
	for i := 0; i < retries; i++ {
		log.Info(fmt.Sprintf("Trying %s:%d", host, port))
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, string(port)), timeout)
		if err == nil {
			log.Info(fmt.Sprintf("Connected to %s:%d", host, port))
			if conn != nil {
				conn.Close()
			}
			break
		}
	}
}

// WaitForRpcServer waits for an RPC server to be up.
func WaitForRpcServer(addr string) {
	log.Info(fmt.Sprintf("Waiting for RPC server at %s", addr))

	body := []byte(`{
		"id": 1,
		"jsonrpc": "2.0",
		"method": "eth_chainId",
		"params": []
	}`)
	r, err := http.NewRequest("POST", addr, bytes.NewBuffer(body))
	if err != nil {
		log.Crit("Failed to create rpc request", "err", err)
	}
	r.Header.Add("Content-Type", "application/json")

	for {
		client := &http.Client{}
		res, err := client.Do(r)
		if err != nil {
			continue
		}
		if res.StatusCode < 300 {
			log.Info(fmt.Sprintf("RPC server at %s is ready", addr))
			break
		}
		log.Info(fmt.Sprintf("Waiting for RPC server at %s", addr))
		time.Sleep(1 * time.Second)
	}
}
