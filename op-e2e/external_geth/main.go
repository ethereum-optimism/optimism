package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/external"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func main() {
	var init bool
	var configPath string
	flag.BoolVar(&init, "init", false, "Do one time setup for all executions")
	flag.StringVar(&configPath, "config", "", "Execute based on the config in this file")
	flag.Parse()
	err := run(init, configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func build() error {
	cmd := exec.Command("go", "build", "-o", "op-geth", "github.com/ethereum/go-ethereum/cmd/geth")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run(init bool, configPath string) error {
	if !init && configPath == "" {
		return fmt.Errorf("must supply a '--config <path>' or '--init' flag")
	}

	if init {
		if err := build(); err != nil {
			return fmt.Errorf("could not build op-geth: %w", err)
		}
		fmt.Printf("Successfully built op-geth!\n")

		if configPath == "" {
			return nil
		}
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
	gm := gomega.NewGomega(func(msg string, _ ...int) {
		err = errors.New(msg)
	})
	gm.Expect(err).NotTo(gomega.HaveOccurred())
	var enginePort, httpPort int
	for i := 0; i < 2; i++ {
		gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("HTTP server started\\s*endpoint=127.0.0.1:"))
		if err != nil {
			return nil, fmt.Errorf("http endpoint never opened")
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
