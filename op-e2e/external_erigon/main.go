package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	var configPath string
	flag.StringVar(&configPath, "config", "", "Execute based on the config in this file")
	flag.Parse()
	err := run(configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func run(configPath string) error {
	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("could not open config: %w", err)
	}

	var config external.Config
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
	if err := external.AtomicEncode(config.EndpointsReadyPath, sess.endpoints); err != nil {
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

func initialize(binPath string, config external.Config) error {
	cmd := exec.Command(
		binPath,
		"--datadir", config.DataDir,
		"init", config.GenesisPath,
	)
	if err := cmd.Run(); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("could not kill process: %w", err)
		}
		return err
	}
	return nil
}

type erigonSession struct {
	session   *gexec.Session
	endpoints *external.Endpoints
}

func (es *erigonSession) Close() {
	es.session.Terminate()
	select {
	case <-time.After(5 * time.Second):
		es.session.Kill()
	case <-es.session.Exited:
	}
}

func execute(binPath string, config external.Config) (*erigonSession, error) {
	if config.Verbosity < 3 {
		// Note, we could manually filter the logging further, if this is
		// really problematic.
		return nil, fmt.Errorf("verbosity of at least 2 is required to scrape for logs")
	}
	cmd := exec.Command(
		binPath,
		"--chain", "dev",
		"--datadir", config.DataDir,
		"--db.size.limit", "8TB",
		"--ws",
		"--ws.port", "0",
		"--mine",
		"--miner.gaslimit", strconv.FormatUint(config.GasCeil, 10),
		"--http=true",
		"--http.port", "0",
		"--http.addr", "127.0.0.1",
		"--http.api", "eth,debug,net,engine,erigon,web3,txpool",
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
		"--log.console.verbosity", strconv.FormatUint(config.Verbosity, 10),
	)
	// The order of messages for engine vs vanilla http API is inconsistent.  A
	// quick hack is to simply write to two gbytes buffers
	engineBuffer := gbytes.NewBuffer()
	sess, err := gexec.Start(cmd, os.Stdout, io.MultiWriter(os.Stderr, engineBuffer))
	gm := gomega.NewGomega(func(msg string, _ ...int) {
		err = errors.New(msg)
	})
	gm.Expect(err).NotTo(gomega.HaveOccurred())

	var enginePort, httpPort int
	gm.Eventually(engineBuffer, time.Minute).Should(gbytes.Say("JsonRpc endpoint opened\\s*.*http.url=127.0.0.1:"))
	if err != nil {
		return nil, fmt.Errorf("http endpoint never opened")
	}
	fmt.Fscanf(engineBuffer, "%d", &httpPort)
	fmt.Printf("==================    op-erigon shim got http port %d  ==========================\n", httpPort)

	gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("HTTP endpoint opened for Engine API\\s*url=127.0.0.1:"))
	if err != nil {
		return nil, fmt.Errorf("http engine endpoint never opened")
	}
	fmt.Fscanf(sess.Err, "%d", &enginePort)
	fmt.Printf("==================    op-erigon shim got engine port %d  ==========================\n", enginePort)
	gm.Eventually(sess.Err, time.Minute).Should(gbytes.Say("Regeneration ended"))
	if err != nil {
		return nil, fmt.Errorf("started did not finish in time")
	}

	return &erigonSession{
		session: sess,
		endpoints: &external.Endpoints{
			HTTPEndpoint:     fmt.Sprintf("http://127.0.0.1:%d/", httpPort),
			WSEndpoint:       fmt.Sprintf("ws://127.0.0.1:%d/", httpPort),
			HTTPAuthEndpoint: fmt.Sprintf("http://127.0.0.1:%d/", enginePort),
			WSAuthEndpoint:   fmt.Sprintf("ws://127.0.0.1:%d/", enginePort),
		},
	}, nil
}
