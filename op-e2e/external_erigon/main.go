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

	e2e "github.com/ethereum-optimism/optimism/op-e2e"
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
	outFile, err := filepath.Abs("op-erigon")
	if err != nil {
		return err
	}
	workDir, err := filepath.Abs(filepath.Join("..", "..", "op-erigon"))
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", outFile, "github.com/ledgerwatch/erigon/cmd/erigon")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running build in %s to create %s\n", cmd.Dir, outFile)
	return cmd.Run()
}

func run(init bool, configPath string) error {
	if !init && configPath == "" {
		return fmt.Errorf("must supply a '--config <path>' or '--init' flag")
	}

	if init {
		if err := build(); err != nil {
			return fmt.Errorf("could not build op-erigon: %w", err)
		}
		fmt.Printf("Successfully built op-erigon!\n")

		if configPath == "" {
			return nil
		}
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("could not open config: %w", err)
	}

	var config e2e.ExternalConfig
	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		return fmt.Errorf("could not decode config file: %w", err)
	}

	binPath, err := filepath.Abs("op-erigon")
	if err != nil {
		return fmt.Errorf("could not get absolute path of op-erigon")
	}
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("could not locate op-erigon in working directory, did you forget to run '--init'?")
	}

	fmt.Printf("================== op-erigon shim initializing chain config ==========================\n")
	if err := initialize(binPath, config); err != nil {
		return fmt.Errorf("could not initialize datadir: %s %w", binPath, err)
	}

	fmt.Printf("==================    op-erigon shim executing op-erigon     ==========================\n")
	sess, err := execute(binPath, config)
	if err != nil {
		return fmt.Errorf("could not execute erigon: %w", err)
	}
	defer sess.Close()

	fmt.Printf("==================    op-erigon shim encoding ready-file   ==========================\n")
	if err := e2e.AtomicEncode(config.EndpointsReadyPath, sess.endpoints); err != nil {
		return fmt.Errorf("could not encode endpoints")
	}

	fmt.Printf("==================    op-erigon shim awaiting termination  ==========================\n")
	select {
	case <-sess.session.Exited:
		return fmt.Errorf("erigon exited")
	case <-time.After(30 * time.Minute):
		return fmt.Errorf("exiting after 30 minute timeout")
	}
}

func initialize(binPath string, config e2e.ExternalConfig) error {
	cmd := exec.Command(
		binPath,
		"--datadir", config.DataDir,
		"init", config.GenesisPath,
	)
	return cmd.Run()
}

type erigonSession struct {
	session   *gexec.Session
	endpoints *e2e.ExternalEndpoints
}

func (es *erigonSession) Close() {
	es.session.Terminate()
	select {
	case <-time.After(5 * time.Second):
		es.session.Kill()
	case <-es.session.Exited:
	}
}

func execute(binPath string, config e2e.ExternalConfig) (*erigonSession, error) {
	cmd := exec.Command(
		binPath,
		"--chain", "dev",
		"--datadir", config.DataDir,
		"--log.console.verbosity", "dbug",
		"--ws",
		"--mine",
		"--miner.gaslimit", strconv.FormatUint(config.GasCeil, 10),
		"--http.port", "0",
		"--http.addr", "127.0.0.1",
		"--http.api", "eth,debug,net,engine,erigon,web3",
		"--private.api.addr=127.0.0.1:0",
		"--allow-insecure-unlock",
		"--authrpc.addr=127.0.0.1",
		"--nat", "none",
		"--p2p.allowed-ports", "0",
		"--authrpc.port=0",
		"--authrpc.vhosts=*",
		"--authrpc.jwtsecret", config.JWTPath,
		"--networkid", strconv.FormatUint(config.ChainID, 10),
		"--torrent.port", "0", // There doesn't seem to be an obvious way to disable torrent listening
	)
	sess, err := gexec.Start(cmd, os.Stdout, os.Stderr)
	gm := gomega.NewGomega(func(msg string, _ ...int) {
		err = errors.New(msg)
	})
	gm.Expect(err).NotTo(gomega.HaveOccurred())

	var enginePort, httpPort int
	gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("HTTP endpoint opened for Engine API\\s*url=127.0.0.1:"))
	if err != nil {
		return nil, fmt.Errorf("http engine endpoint never opened")
	}
	fmt.Fscanf(sess.Err, "%d", &enginePort)
	gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("HTTP endpoint opened\\s*url=127.0.0.1:"))
	if err != nil {
		return nil, fmt.Errorf("http endpoint never opened")
	}
	fmt.Fscanf(sess.Err, "%d", &httpPort)
	gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("\\[1/15 Snapshots\\] DONE"))
	if err != nil {
		return nil, fmt.Errorf("started did not finish in time")
	}

	return &erigonSession{
		session: sess,
		endpoints: &e2e.ExternalEndpoints{
			HTTPEndpoint:     fmt.Sprintf("http://127.0.0.1:%d/", httpPort),
			WSEndpoint:       fmt.Sprintf("ws://127.0.0.1:%d/", httpPort),
			HTTPAuthEndpoint: fmt.Sprintf("http://127.0.0.1:%d/", enginePort),
			WSAuthEndpoint:   fmt.Sprintf("ws://127.0.0.1:%d/", enginePort),
		},
	}, nil
}
