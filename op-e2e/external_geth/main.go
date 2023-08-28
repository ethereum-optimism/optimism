package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/external"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Execute based on the config in this file")
	flag.Parse()
	if err := run(configPath); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func run(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("must supply a '--config <path>' flag")
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("could not open config: %w", err)
	}

	var config external.Config
	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		return fmt.Errorf("could not decode config file: %w", err)
	}

	binPath, err := filepath.Abs("op-geth")
	if err != nil {
		return fmt.Errorf("could not get absolute path of op-geth")
	}
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("could not locate op-geth in working directory, did you forget to run '--init'?")
	}

	fmt.Printf("================== op-geth shim initializing chain config ==========================\n")
	if err := initialize(binPath, config); err != nil {
		return fmt.Errorf("could not initialize datadir: %s %w", binPath, err)
	}

	fmt.Printf("==================    op-geth shim executing op-geth     ==========================\n")
	sess, err := execute(binPath, config)
	if err != nil {
		return fmt.Errorf("could not execute geth: %w", err)
	}
	defer sess.Close()

	fmt.Printf("==================    op-geth shim encoding ready-file   ==========================\n")
	if err := external.AtomicEncode(config.EndpointsReadyPath, sess.endpoints); err != nil {
		return fmt.Errorf("could not encode endpoints")
	}

	fmt.Printf("==================    op-geth shim awaiting termination  ==========================\n")
	select {
	case <-sess.session.Exited:
		return fmt.Errorf("geth exited")
	case <-time.After(30 * time.Minute):
		return fmt.Errorf("exiting after 30 minute timeout")
	}
}

func initialize(binPath string, config external.Config) error {
	cmd := exec.Command(
		binPath,
		"--datadir", config.DataDir,
		"init", config.GenesisPath,
	)
	return cmd.Run()
}

type gethSession struct {
	session   *gexec.Session
	endpoints *external.Endpoints
}

func (es *gethSession) Close() {
	es.session.Terminate()
	select {
	case <-time.After(5 * time.Second):
		es.session.Kill()
	case <-es.session.Exited:
	}
}

func execute(binPath string, config external.Config) (*gethSession, error) {
	if config.Verbosity < 2 {
		return nil, fmt.Errorf("a minimum configured verbosity of 2 is required")
	}
	cmd := exec.Command(
		binPath,
		"--datadir", config.DataDir,
		"--http",
		"--http.addr", "127.0.0.1",
		"--http.port", "0",
		"--http.api", "web3,debug,eth,txpool,net,engine",
		"--ws",
		"--ws.addr", "127.0.0.1",
		"--ws.port", "0",
		"--ws.api", "debug,eth,txpool,net,engine",
		"--syncmode=full",
		"--nodiscover",
		"--port", "0",
		"--maxpeers", "0",
		"--networkid", strconv.FormatUint(config.ChainID, 10),
		"--authrpc.addr", "127.0.0.1",
		"--authrpc.port", "0",
		"--authrpc.jwtsecret", config.JWTPath,
		"--gcmode=archive",
		"--verbosity", strconv.FormatUint(config.Verbosity, 10),
	)
	sess, err := gexec.Start(cmd, os.Stdout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not start op-geth session: %w", err)
	}
	matcher := gbytes.Say("HTTP server started\\s*endpoint=127.0.0.1:")
	var enginePort, httpPort int
	for enginePort == 0 || httpPort == 0 {
		match, err := matcher.Match(sess.Err)
		if err != nil {
			return nil, fmt.Errorf("could not execute matcher")
		}
		if !match {
			if sess.Err.Closed() {
				return nil, fmt.Errorf("op-geth exited before announcing http ports")
			}
			// Wait for a bit more output, then try again
			time.Sleep(10 * time.Millisecond)
			continue
		}
		var authString string
		var port int
		fmt.Fscanf(sess.Err, "%d %s", &port, &authString)
		switch authString {
		case "auth=true":
			enginePort = port
		case "auth=false":
			httpPort = port
		default:
			return nil, fmt.Errorf("unexpected auth string %q", authString)
		}
	}

	return &gethSession{
		session: sess,
		endpoints: &external.Endpoints{
			HTTPEndpoint:     fmt.Sprintf("http://127.0.0.1:%d/", httpPort),
			WSEndpoint:       fmt.Sprintf("ws://127.0.0.1:%d/", httpPort),
			HTTPAuthEndpoint: fmt.Sprintf("http://127.0.0.1:%d/", enginePort),
			WSAuthEndpoint:   fmt.Sprintf("ws://127.0.0.1:%d/", enginePort),
		},
	}, nil
}
