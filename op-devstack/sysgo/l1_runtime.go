package sysgo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

const DevstackL1ELKindEnvVar = "DEVSTACK_L1EL_KIND"

const GethExecPathEnvVar = "SYSGO_GETH_EXEC_PATH"

func writeJWTSecret(t devtest.T) (string, [32]byte) {
	jwtPath := filepath.Join(t.TempDir(), "jwt_secret")
	jwtSecret := [32]byte{123}
	err := os.WriteFile(jwtPath, []byte(hexutil.Encode(jwtSecret[:])), 0o600)
	t.Require().NoError(err, "failed to write jwt secret")
	return jwtPath, jwtSecret
}

func startInProcessL1(t devtest.T, l1Net *L1Network, jwtPath string) (*L1Geth, *L1CLNode) {
	return startInProcessL1WithClock(t, l1Net, jwtPath, clock.SystemClock)
}

func startInProcessL1WithClock(t devtest.T, l1Net *L1Network, jwtPath string, l1Clock clock.Clock) (*L1Geth, *L1CLNode) {
	if os.Getenv(DevstackL1ELKindEnvVar) == "geth" {
		return startSubprocessL1WithClock(t, l1Net, jwtPath, l1Clock)
	}

	require := t.Require()
	l1ChainID := l1Net.ChainID()

	blobPath := t.TempDir()
	bcn := fakebeacon.NewBeacon(t.Logger().New("component", "l1cl"), blobstore.New(), l1Net.genesis.Timestamp, l1Net.blockTime)
	t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(bcn.Start("127.0.0.1:0"))
	beaconAddr := bcn.BeaconAddr()
	require.NotEmpty(beaconAddr, "beacon API listener must be up")

	l1Geth, fp, err := geth.InitL1(
		l1Net.blockTime,
		20,
		l1Net.genesis,
		l1Clock,
		filepath.Join(blobPath, "l1_el"),
		bcn,
		geth.WithAuth(jwtPath),
	)
	require.NoError(err)
	require.NoError(l1Geth.Node.Start())
	t.Cleanup(func() {
		t.Logger().Info("Closing L1 geth")
		_ = l1Geth.Close()
	})

	l1EL := &L1Geth{
		name:     "l1",
		chainID:  l1ChainID,
		userRPC:  l1Geth.Node.HTTPEndpoint(),
		authRPC:  l1Geth.Node.HTTPAuthEndpoint(),
		l1Geth:   l1Geth,
		blobPath: blobPath,
	}
	l1CL := &L1CLNode{
		name:           "l1",
		chainID:        l1ChainID,
		beaconHTTPAddr: beaconAddr,
		beacon:         bcn,
		fakepos:        &FakePoS{fakepos: fp, p: t},
	}
	return l1EL, l1CL
}

func startSubprocessL1WithClock(t devtest.T, l1Net *L1Network, jwtPath string, l1Clock clock.Clock) (*L1Geth, *L1CLNode) {
	require := t.Require()
	l1ChainID := l1Net.ChainID()

	execPath, ok := os.LookupEnv(GethExecPathEnvVar)
	require.True(ok, "%s must be set when %s=geth", GethExecPathEnvVar, DevstackL1ELKindEnvVar)
	_, err := os.Stat(execPath)
	require.NotErrorIs(err, os.ErrNotExist, "geth executable must exist")

	tempDir := t.TempDir()
	data, err := json.Marshal(l1Net.genesis)
	require.NoError(err, "must json-encode genesis")
	chainConfigPath := filepath.Join(tempDir, "genesis.json")
	require.NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write genesis file")

	dataDirPath := filepath.Join(tempDir, "data")
	require.NoError(os.MkdirAll(dataDirPath, 0o755), "must create datadir")

	initCmd := exec.Command(execPath, "--datadir", dataDirPath, "init", chainConfigPath)
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	require.NoError(initCmd.Run(), "initialize geth datadir")

	userProxy := tcpproxy.New(t.Logger().New("component", "l1el-user-proxy"))
	require.NoError(userProxy.Start())
	t.Cleanup(func() {
		userProxy.Close()
	})
	authProxy := tcpproxy.New(t.Logger().New("component", "l1el-auth-proxy"))
	require.NoError(authProxy.Start())
	t.Cleanup(func() {
		authProxy.Close()
	})

	userRPC := "ws://" + userProxy.Addr()
	authRPC := "ws://" + authProxy.Addr()
	userRPCUpstream := make(chan string, 1)
	authRPCUpstream := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "WebSocket enabled":
			select {
			case userRPCUpstream <- e.FieldValue("url").(string):
			default:
			}
		case "HTTP server started":
			if e.FieldValue("auth").(bool) {
				select {
				case authRPCUpstream <- "http://" + e.FieldValue("endpoint").(string):
				default:
				}
			}
		}
	}
	logOut := logpipe.ToLogger(t.Logger().New("component", "l1el", "src", "stdout"))
	logErr := logpipe.ToLogger(t.Logger().New("component", "l1el", "src", "stderr"))
	stdOutLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseGoStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseGoStructuredLogs(line)
		logErr(e)
		onLogEntry(e)
	})
	sub := NewSubProcess(t, stdOutLogs, stdErrLogs)
	args := []string{
		"--log.format", "json",
		"--datadir", dataDirPath,
		"--ws", "--ws.addr", "127.0.0.1", "--ws.port", "0", "--ws.origins", "*", "--ws.api", "admin,debug,eth,net,txpool",
		"--authrpc.addr", "127.0.0.1", "--authrpc.port", "0", "--authrpc.jwtsecret", jwtPath,
		"--ipcdisable",
		"--port", "0",
		"--nodiscover",
		"--verbosity", "5",
		"--miner.recommit", "2s",
		"--gcmode", "archive",
	}
	require.NoError(sub.Start(execPath, args, nil), "must start geth subprocess")

	var userRPCAddr string
	var authRPCAddr string
	require.NoError(tasks.Await(t.Ctx(), userRPCUpstream, &userRPCAddr), "need geth user RPC")
	require.NoError(tasks.Await(t.Ctx(), authRPCUpstream, &authRPCAddr), "need geth auth RPC")
	userProxy.SetUpstream(ProxyAddr(require, userRPCAddr))
	authProxy.SetUpstream(ProxyAddr(require, authRPCAddr))

	backend, err := ethclient.DialContext(t.Ctx(), userRPC)
	require.NoError(err, "failed to dial geth user RPC")
	t.Cleanup(backend.Close)

	jwtSecret := readJWTSecret(t, jwtPath)
	engineCl, err := dialEngine(t.Ctx(), authRPC, jwtSecret)
	require.NoError(err, "failed to dial geth engine API")
	t.Cleanup(func() {
		engineCl.inner.Close()
	})

	bcn := fakebeacon.NewBeacon(t.Logger().New("component", "l1cl"), blobstore.New(), l1Net.genesis.Timestamp, l1Net.blockTime)
	t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(bcn.Start("127.0.0.1:0"))
	beaconAddr := bcn.BeaconAddr()
	require.NotEmpty(beaconAddr, "beacon API listener must be up")

	fp := &FakePoS{
		p:       t,
		fakepos: geth.NewFakePoS(backend, engineCl, l1Clock, t.Logger().New("component", "l1cl"), l1Net.blockTime, 20, bcn, l1Net.genesis.Config),
	}
	fp.Start()
	t.Cleanup(fp.Stop)

	l1EL := &L1Geth{
		name:     "l1",
		chainID:  l1ChainID,
		userRPC:  userRPC,
		authRPC:  authRPC,
		blobPath: tempDir,
	}
	l1CL := &L1CLNode{
		name:           "l1",
		chainID:        l1ChainID,
		beaconHTTPAddr: beaconAddr,
		beacon:         bcn,
		fakepos:        fp,
	}
	return l1EL, l1CL
}

func readJWTSecret(t devtest.T, jwtPath string) [32]byte {
	data, err := os.ReadFile(jwtPath)
	t.Require().NoError(err, "failed to read jwt secret file")
	decoded, err := hexutil.Decode(strings.TrimSpace(string(data)))
	t.Require().NoError(err, "failed to decode jwt secret file")
	var jwtSecret [32]byte
	copy(jwtSecret[:], decoded)
	t.Require().Len(decoded, len(jwtSecret), "jwt secret must be 32 bytes")
	return jwtSecret
}
