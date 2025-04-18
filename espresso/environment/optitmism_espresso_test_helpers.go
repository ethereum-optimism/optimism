package environment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network-go/client"
	espressoCommon "github.com/EspressoSystems/espresso-network-go/types"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/faultproofs"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	gethNode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

type EspressoAllocAccount struct {
	State types.Account `json:"state"`
	Name  string        `json:"name"`
}

//go:embed allocs.json
var ESPRESSO_ALLOCS_RAW string
var ESPRESSO_ALLOCS map[common.Address]EspressoAllocAccount

func init() {
	// Unmarshal allocs to set up the dockerConfig environment variables
	ESPRESSO_ALLOCS = make(map[common.Address]EspressoAllocAccount)

	if err := json.Unmarshal([]byte(ESPRESSO_ALLOCS_RAW), &ESPRESSO_ALLOCS); err != nil {
		panic(fmt.Sprintf("failed to unmarshal ESPRESSO_ALLOCS: %v", err))
	}
}

const ESPRESSO_LIGHT_CLIENT_ADDRESS = "0x703848f4c85f18e3acd8196c8ec91eb0b7bd0797"

const ESPRESSO_DEV_NODE_DOCKER_IMAGE = "ghcr.io/espressosystems/espresso-sequencer/espresso-dev-node:release-colorful-snake"

// This is the mnemonic that we use to create the private key for deploying
// contacts on the L1
const ESPRESSO_MNEMONIC = "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete"

// This is the Mnemonic Index that we use to create the private key for deploying
// contracts on the L1
const ESPRESSO_MNEMONIC_INDEX = "0"

const ESPRESSO_TESTING_BATCHER_KEY = "0xfad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"

// This is address that corresponds to the menmonic we pass to the espresso-dev-node
var ESPRESSO_CONTRACT_ACCOUNT = common.HexToAddress("0x8943545177806ed17b9f23f0a21ee5948ecaa776")

const ESPRESSO_BUILDER_PORT = "31003"
const ESPRESSO_SEQUENCER_API_PORT = "24000"
const ESPRESSO_DEV_NODE_PORT = "24002"

// ErrEspressoBlockHeightDidNotIncrease is a sentinel error that occurs when
// the Espresso Block Height does not increase within the alloted context
// allowance.
var ErrEspressoBlockHeightDidNotIncrease = errors.New("espresso block height did not increase")

// ErrFailedToParseNumber is a sentinel error that occurs when we are unable
// to parse a number from a string
var ErrFailedToParseNumber = errors.New("failed to parse number from string")

// WaitForEspressoBlockHeightToBePositive waits for the Espresso Block Height to
// increase beyond 0.
func WaitForEspressoBlockHeightToBePositive(ctx context.Context, url string) error {
	for {
		select {
		case <-ctx.Done():
			// We've timed out
			return ErrEspressoBlockHeightDidNotIncrease
		default:
		}

		time.Sleep(time.Millisecond * 10)

		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			// Service may not yet be available?
			continue
		}

		if response.StatusCode != http.StatusOK {
			// Service may not yet be available?
			continue
		}

		// Alright, presumably, we have a block height

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, response.Body); err != nil {
			return err
		}
		if err := response.Body.Close(); err != nil {
			return err
		}

		blockHeight, ok := new(big.Int).SetString(buf.String(), 10)
		if !ok {
			return ErrFailedToParseNumber
		}

		if blockHeight.Cmp(big.NewInt(0)) > 0 {
			// We have a positive block height! That means we're
			// committing blocks, and we're progressing.  We
			// **SHOULD** be good to continue"
			return nil
		}
	}
}

// EspressoDevNodeLauncherDocker is an implementation of EspressoDevNodeLauncher
// that uses Docker to launch the Espresso Dev Node
type EspressoDevNodeLauncherDocker struct {
	// Whether to run batcher in enclave.
	EnclaveBatcher bool
	// Whether to enable AltDa
	AltDa bool
}

var _ EspressoDevNetLauncher = (*EspressoDevNodeLauncherDocker)(nil)

// FailedToDetermineL1RPCURL represents a class of errors that occur when we
// are unable to correctly form our L1 RPC URL
type FailedToDetermineL1RPCURL struct {
	Cause error
}

// Error implements error
func (f FailedToDetermineL1RPCURL) Error() string {
	return fmt.Sprintf("failed to determine the L1 RPC URL: %v", f.Cause)
}

// FailedToLoadEspressoAccount represents a class of errors that occur when we
// are unable to load the espresso account
type FailedToLoadEspressoAccount struct {
	Cause error
}

// Error implements error
func (f FailedToLoadEspressoAccount) Error() string {
	return fmt.Sprintf("failed to load the espresso account: %v", f.Cause)
}

// FailedToLaunchDockerContainer represents a class of errors that occur when
// we are unable to launch a docker container
type FailedToLaunchDockerContainer struct {
	Cause error
}

// Error implements error
func (f FailedToLaunchDockerContainer) Error() string {
	return fmt.Sprintf("failed to launch docker container: %v", f.Cause)
}

// EspressoNodeFailedToBecomeReady represents a class of errors that indicate
// that the espresso-dev-node failed to become ready.
type EspressoNodeFailedToBecomeReady struct {
	Cause error
}

// Error implements error
func (e EspressoNodeFailedToBecomeReady) Error() string {
	return fmt.Sprintf("espresso node failed to become ready: %v", e.Cause)
}

type EspressoDevNodeContainerInfo struct {
	ContainerInfo DockerContainerInfo
	espressoUrls  []string
}

// EspressoUrl returns the URL of the Espresso node
func (e *EspressoDevNodeContainerInfo) EspressoUrls() []string {
	return e.espressoUrls
}

var _ EspressoDevNode = (*EspressoDevNodeContainerInfo)(nil)

// getPort is a helper function that takes the original port and returns
// the remapped port that the container is listening on.
func (e EspressoDevNodeContainerInfo) getPort(originalPort string) string {
	hosts := e.ContainerInfo.PortMap[originalPort]

	if len(hosts) == 0 {
		return ""
	}

	_, port, err := net.SplitHostPort(hosts[0])
	if err != nil {
		return ""
	}

	return port
}

// SequencerPort implements EspressoDevNode, by returning the relevant
// port for the sequencer API in the Espresso dev node
func (e EspressoDevNodeContainerInfo) SequencerPort() string {
	return e.getPort(ESPRESSO_SEQUENCER_API_PORT)
}

// BuilderPort implements EspressoDevNode, by returning the relevant
// port for the builder API in the Espresso dev node
func (e EspressoDevNodeContainerInfo) BuilderPort() string {
	return e.getPort(ESPRESSO_BUILDER_PORT)
}

// Stop implements EspressoDevNode, and is a convenience method to stop the
// running container.
//
// This is mostly unnecessary as the context that the container was launched
// in will govern the lifecycle of the container automatically, assuming that
// the context is following the recommended context usage patterns.
func (e EspressoDevNodeContainerInfo) Stop() error {
	cli := new(DockerCli)
	return cli.StopContainer(context.Background(), e.ContainerInfo.ContainerID)
}

// ErrUnableToDetermineEspressoDevNodeSequencerHost is a sentinel error that
// indicates that we were unable to determine what the Sequencer API host
// is meant to be.
var ErrUnableToDetermineEspressoDevNodeSequencerHost = errors.New("unable to determine the host for the espresso-dev-node sequencer api")

// GetDevNetConfig returns a configuration for a devnet
func (l *EspressoDevNodeLauncherDocker) GetDevNetSysConfig(ctx context.Context, t *testing.T, options ...DevNetLauncherOption) e2esys.SystemConfig {

	var allocOpt e2esys.SystemConfigOpt
	if l.EnclaveBatcher {
		allocOpt = e2esys.WithAllocType(config.AllocTypeEspressoWithEnclave)
	} else {
		allocOpt = e2esys.WithAllocType(config.AllocTypeEspressoWithoutEnclave)
	}

	sysConfig := e2esys.DefaultSystemConfig(t, allocOpt)

	if l.AltDa {
		sysConfig.DeployConfig.UseAltDA = true
		sysConfig.DeployConfig.DACommitmentType = "KeccakCommitment"
		sysConfig.DeployConfig.DAChallengeWindow = 16
		sysConfig.DeployConfig.DAResolveWindow = 16
		sysConfig.DeployConfig.DABondSize = 1000000
		sysConfig.DeployConfig.DAResolverRefundPercentage = 0
		sysConfig.BatcherMaxPendingTransactions = 0
		sysConfig.BatcherBatchType = 0
		sysConfig.DataAvailabilityType = flags.CalldataType
	}

	// Set a short L1 block time and finalized distance to make tests faster and reach finality sooner
	sysConfig.DeployConfig.L1BlockTime = 2

	sysConfig.DeployConfig.DeployCeloContracts = true

	// Ensure that we fund the dev accounts
	sysConfig.DeployConfig.FundDevAccounts = true

	espressoPremine := new(big.Int).Mul(new(big.Int).SetUint64(1_000_000), new(big.Int).SetUint64(params.Ether))
	sysConfig.L1Allocs[ESPRESSO_CONTRACT_ACCOUNT] = types.Account{
		Nonce:   100000,          // Set the nonce to avoid collisions with predeployed contracts
		Balance: espressoPremine, // Pre-fund Espresso deployer acount with 1M Ether
	}

	//Set up the L1Allocs in the system config
	for address, account := range ESPRESSO_ALLOCS {
		sysConfig.L1Allocs[address] = account.State
	}

	return sysConfig
}

// GetDevNetWithFaultDisputeSysConfig returns a configuration for a devnet with a Fault Dispute System
func (l *EspressoDevNodeLauncherDocker) GetDevNetWithFaultDisputeSysConfig(ctx context.Context, t *testing.T, options ...DevNetLauncherOption) e2esys.SystemConfig {
	var allocOpt e2esys.SystemConfigOpt
	if l.EnclaveBatcher {
		allocOpt = e2esys.WithAllocType(config.AllocTypeEspressoWithEnclave)
	} else {
		allocOpt = e2esys.WithAllocType(config.AllocTypeEspressoWithoutEnclave)
	}

	// Get a Fault Dispute System configuration with Espresso Dev Node allocation
	sysConfig := faultproofs.GetFaultDisputeSystemConfigForEspresso(t, []e2esys.SystemConfigOpt{allocOpt})

	if l.AltDa {
		sysConfig.DeployConfig.UseAltDA = true
		sysConfig.DeployConfig.DACommitmentType = "KeccakCommitment"
		sysConfig.DeployConfig.DAChallengeWindow = 16
		sysConfig.DeployConfig.DAResolveWindow = 16
		sysConfig.DeployConfig.DABondSize = 1000000
		sysConfig.DeployConfig.DAResolverRefundPercentage = 0
		sysConfig.BatcherMaxPendingTransactions = 0
		sysConfig.BatcherBatchType = 0
		sysConfig.DataAvailabilityType = flags.CalldataType
	}

	// Set a short L1 block time and finalized distance to make tests faster and reach finality sooner
	sysConfig.DeployConfig.L1BlockTime = 2

	sysConfig.DeployConfig.DeployCeloContracts = true

	// Ensure that we fund the dev accounts
	sysConfig.DeployConfig.FundDevAccounts = true

	espressoPremine := new(big.Int).Mul(new(big.Int).SetUint64(1_000_000), new(big.Int).SetUint64(params.Ether))
	sysConfig.L1Allocs[ESPRESSO_CONTRACT_ACCOUNT] = types.Account{
		Nonce:   100000,          // Set the nonce to avoid collisions with predeployed contracts
		Balance: espressoPremine, // Pre-fund Espresso deployer acount with 1M Ether
	}

	//Set up the L1Allocs in the system config
	for address, account := range ESPRESSO_ALLOCS {
		sysConfig.L1Allocs[address] = account.State
	}

	return sysConfig
}

// GetDevNetStartOptions returns the start options for the devnet
func (l *EspressoDevNodeLauncherDocker) GetDevNetStartOptions(originalCtx context.Context, t *testing.T, sysConfig *e2esys.SystemConfig, options ...DevNetLauncherOption) ([]e2esys.StartOption, *DevNetLauncherContext) {
	initialOptions := []DevNetLauncherOption{
		allowHostDockerInternalVirtualHost(),
		launchEspressoDevNodeDocker(),
	}

	if l.EnclaveBatcher {
		initialOptions = append(initialOptions, LaunchBatcherInEnclave())
	}

	launchContext := DevNetLauncherContext{
		Ctx:       originalCtx,
		SystemCfg: sysConfig,
	}

	allOptions := append(initialOptions, options...)

	startOptions := []e2esys.StartOption{}

	for _, opt := range allOptions {
		options := opt(&launchContext)

		if gethOption := options.GethOptions; gethOption != nil {
			for k, v := range gethOption {
				sysConfig.GethOptions[k] = append(sysConfig.GethOptions[k], v...)
			}
		}

		if startOption := options.StartOptions; startOption != nil {
			startOptions = append(startOptions, startOption...)
		}

		if sysConfigOption := options.SysConfigOption; sysConfigOption != nil {
			sysConfigOption(sysConfig)
		}
	}

	return startOptions, &launchContext
}

func (l *EspressoDevNodeLauncherDocker) StartDevNet(ctx context.Context, t *testing.T, options ...DevNetLauncherOption) (*e2esys.System, EspressoDevNode, error) {

	sysConfig := l.GetDevNetSysConfig(ctx, t, options...)

	originalCtx := ctx
	startOptions, launchContext := l.GetDevNetStartOptions(originalCtx, t, &sysConfig, options...)

	// We want to run the espresso-dev-node.  But we need it to be able to
	// access the L1 node.

	system, err := sysConfig.Start(
		t,

		startOptions...,
	)

	if err != nil {
		if system != nil {
			// We don't want the system running in a partial / incomplete
			// state. So we'll tell it to stop here, just in case.
			system.Close()
		}

		return system, nil, err
	}

	// Auto System Cleanup tied to the passed in context.
	{
		// We want to ensure that the lifecycle of the system node is tied to
		// the context we were given, just like the espresso-dev-node.  So if
		// the context is canceled, or otherwise closed, it will automatically
		// clean up the system.
		go (func(ctx context.Context) {
			<-ctx.Done()

			// The system is guaranteed to not be null here.
			system.Close()
		})(originalCtx)
	}

	return system, launchContext.EspressoDevNode, launchContext.Error
}

// StartDevNetWithFaultDisputeSystem starts a Fault Dispute System with an Espresso Dev Node
func (l *EspressoDevNodeLauncherDocker) StartDevNetWithFaultDisputeSystem(ctx context.Context, t *testing.T, options ...DevNetLauncherOption) (*e2esys.System, EspressoDevNode, error) {

	sysConfig := l.GetDevNetWithFaultDisputeSysConfig(ctx, t, options...)

	originalCtx := ctx
	startOptions, launchContext := l.GetDevNetStartOptions(originalCtx, t, &sysConfig, options...)

	system, err := sysConfig.Start(
		t,

		startOptions...,
	)

	if err != nil {
		if system != nil {
			// We don't want the system running in a partial / incomplete
			// state. So we'll tell it to stop here, just in case.
			system.Close()
		}

		return system, nil, err
	}

	// Auto System Cleanup tied to the passed in context.
	{
		// We want to ensure that the lifecycle of the system node is tied to
		// the context we were given, just like the espresso-dev-node.  So if
		// the context is canceled, or otherwise closed, it will automatically
		// clean up the system.
		go (func(ctx context.Context) {
			<-ctx.Done()

			// The system is guaranteed to not be null here.
			system.Close()
		})(originalCtx)
	}

	return system, launchContext.EspressoDevNode, launchContext.Error
}

// EspressoDevNodeDockerContainerInfo is an implementation of
// EspressoDevNode that uses a Docker container to run the Espresso Dev Node
// and provides the relevant port information for the sequencer API and
type EspressoDevNodeDockerContainerInfo struct {
	DockerContainerInfo
	espressoUrls []string
}

// EspressoUrl returns the URL of the Espresso node
func (e *EspressoDevNodeDockerContainerInfo) EspressoUrls() []string {
	return e.espressoUrls
}

var _ EspressoDevNode = (*EspressoDevNodeDockerContainerInfo)(nil)

// SequencerPort implements EspressoDevNode
func (e EspressoDevNodeDockerContainerInfo) SequencerPort() string {
	ports := e.PortMap[ESPRESSO_SEQUENCER_API_PORT]
	if len(ports) <= 0 {
		return ""
	}

	return ports[0]
}

// BuilderPort implements EspressoDevNode
func (e EspressoDevNodeDockerContainerInfo) BuilderPort() string {
	ports := e.PortMap[ESPRESSO_BUILDER_PORT]
	if len(ports) <= 0 {
		return ""
	}

	return ports[0]
}

// ContainerID implements EspressoDevNode
func (e EspressoDevNodeDockerContainerInfo) Stop() error {
	cli := new(DockerCli)
	return cli.StopContainer(context.Background(), e.ContainerID)
}

// allowHostDockerInternalVirtualHost is a convenience method that configures
// Geth instance to allow communication from a virtual host of
// "host.docker.internal".
//
// host.docker.internal is a special DNS name that allows docker containers
// to speak to ports hosted on the host node.
func allowHostDockerInternalVirtualHost() DevNetLauncherOption {
	return func(c *DevNetLauncherContext) E2eSystemOption {
		return E2eSystemOption{
			GethOptions: map[string][]geth.GethOption{
				e2esys.RoleL1: {
					func(thCfg *ethconfig.Config, nodeCfg *gethNode.Config) error {
						// We append the host machine address to the list of virtual hosts, so
						// that we do not get denied when attempting to access the host machine's
						// RPC API.
						nodeCfg.HTTPVirtualHosts = append(nodeCfg.HTTPVirtualHosts, "host.docker.internal", "localhost")

						return nil
					},
				},
			},
		}
	}
}

// This code is adapted from a gist file:
// https://gist.github.com/sevkin/96bdae9274465b2d09191384f86ef39d
func determineFreePort() (port int, err error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() {
		err = listener.Close()
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func SetBatcherKey(privateKey ecdsa.PrivateKey) DevNetLauncherOption {
	return func(ct *DevNetLauncherContext) E2eSystemOption {
		return E2eSystemOption{
			StartOptions: []e2esys.StartOption{
				{
					Role: "set-batcher-key",
					BatcherMod: func(c *batcher.CLIConfig, sys *e2esys.System) {
						c.TestingEspressoBatcherPrivateKey = hexutil.Encode(crypto.FromECDSA(&privateKey))
					},
				},
			},
		}
	}
}

// SetEspressoUrls allows to set the list of urls for the Espresso client in such a way that N of them are "good" and M of them are "bad".
// Good urls are the urls defined by this test framework repeated M times. The bad url is provided to the function
// This function is introduced for testing purposes. It allows to check the enforcement of the majority rule (Test 12)
func SetEspressoUrls(numGood int, numBad int, badServerUrl string) DevNetLauncherOption {
	return func(ct *DevNetLauncherContext) E2eSystemOption {

		return E2eSystemOption{
			StartOptions: []e2esys.StartOption{
				{
					BatcherMod: func(c *batcher.CLIConfig, sys *e2esys.System) {

						goodUrl := c.EspressoUrls[0]
						var urls []string

						for i := 0; i < numGood; i++ {
							urls = append(urls, goodUrl)
						}

						for i := 0; i < numBad; i++ {
							urls = append(urls, badServerUrl)
						}
						c.EspressoUrls = urls
					},
				},
			},
		}
	}
}

func Config(fn func(*e2esys.SystemConfig)) DevNetLauncherOption {
	return func(ct *DevNetLauncherContext) E2eSystemOption {
		return E2eSystemOption{
			SysConfigOption: fn,
		}
	}
}

// getContainerRemappedHostPort is a helper function that takes the
// containerListeningHostPort and returns the remapped host port
// that the container is listening on.
//
// By default the mapped hosts and ports are in the form of
// - 0.0.0.0:<port> for IPv4
// - [::]:<port> for IPv6
//
// So this function will replace the host with "localhost" to allow
// for communication with the host system.
func getContainerRemappedHostPort(containerListeningHostPort string) (string, error) {
	_, port, err := net.SplitHostPort(containerListeningHostPort)
	if err != nil {
		return "", ErrUnableToDetermineEspressoDevNodeSequencerHost
	}

	hostPort := net.JoinHostPort("localhost", port)

	return hostPort, nil
}

// waitForEspressoToFinishSpinningUp is a helper function that waits for the
// espresso dev node to finish spinning up.
// It checks the portMap of the DockerContainerInfo to retrieve the
// Espresso Dev Node Sequencer API port, and then waits for the block height
// to be greater than 0.
func waitForEspressoToFinishSpinningUp(ct *DevNetLauncherContext, espressoDevNodeContainerInfo DockerContainerInfo) error {
	// We have all of our ports.
	// Let's return all of the relevant port mapping information
	// for easy reference, and cancellation

	hosts := espressoDevNodeContainerInfo.PortMap[ESPRESSO_SEQUENCER_API_PORT]

	if len(hosts) == 0 {
		return ErrUnableToDetermineEspressoDevNodeSequencerHost
	}

	// We may have more than a single host, but we'll make do.
	hostPort, err := getContainerRemappedHostPort(hosts[0])
	if err != nil {
		return err
	}

	currentBlockHeightURLString := "http://" + hostPort + "/status/block-height"

	// Wait for Espresso to be ready
	timeoutCtx, cancel := context.WithTimeout(ct.Ctx, 3*time.Minute)
	defer cancel()
	return WaitForEspressoBlockHeightToBePositive(timeoutCtx, currentBlockHeightURLString)
}

// translateContainerToNodeURL is a helper function that translates the the
// given URL to be used by a container to a form that can be communicated with
// the host system.
//
//	Note:
//		if the network passed in is determined to be "host" we will assume that
//		the host machine can be accessed via "localhost".
//
//	Note:
//
//		The default way we assume this will work is with the Docker for X
//		platform, in which the reserved "host.docker.internal" domain name
//		will allow communication with the host system.  This does **NOT**
//		work on a native Linux platform.
func translateContainerToNodeURL(parsedURL url.URL, network string) (url.URL, error) {
	// We need to know the port, so we can configure docker to
	// communicate with the L1 RPC node running on the host machine.
	_, port, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		return url.URL{}, FailedToDetermineL1RPCURL{Cause: err}
	}

	// We replace the host with host.docker.internal to inform
	// docker to communicate with the host system.
	if network == "host" {
		parsedURL.Host = net.JoinHostPort("localhost", port)
	} else {
		parsedURL.Host = net.JoinHostPort("host.docker.internal", port)
	}

	return parsedURL, nil
}

// determineEspressoDevNodeDockerContainerConfig will return an initial
// configuration for the docker cli command to launch the espresso-dev-node.
// It will also return a port mapping that will contain any remapped ports,
// should they be necessary.
func determineEspressoDevNodeDockerContainerConfig(l1EthRpcURL url.URL, network string) (containerConfig DockerContainerConfig, portMapping map[string]string, err error) {
	// These are the expected initial mappings for the ports.  This will
	// be fine when running in an isolated container, and these ports cannot
	// possibly overlap.
	portRemapping := map[string]string{
		ESPRESSO_BUILDER_PORT:       ESPRESSO_BUILDER_PORT,
		ESPRESSO_SEQUENCER_API_PORT: ESPRESSO_SEQUENCER_API_PORT,
		ESPRESSO_DEV_NODE_PORT:      ESPRESSO_DEV_NODE_PORT,
	}

	if network == "host" {
		// If we're running in host mode, we will can potentially have overlapping
		// port definitions, as we spin up nodes in parallel.
		// So we need to determine the free ports on the host system
		// to bind the espresso-dev-node to.
		for portKey := range portRemapping {
			// We need to determine a free port on the host system
			// to bind the espresso-dev-node to.
			freePort, err := determineFreePort()
			if err != nil {
				return DockerContainerConfig{}, nil, FailedToDetermineL1RPCURL{Cause: err}
			}
			portRemapping[portKey] = strconv.FormatInt(int64(freePort), 10)
		}
	}

	l1EthRpcURL.Scheme = "http"

	dockerConfig := DockerContainerConfig{
		Image:   ESPRESSO_DEV_NODE_DOCKER_IMAGE,
		Network: network,
		Environment: map[string]string{
			"ESPRESSO_DEPLOYER_ACCOUNT_INDEX":             ESPRESSO_MNEMONIC_INDEX,
			"ESPRESSO_SEQUENCER_ETH_MNEMONIC":             ESPRESSO_MNEMONIC,
			"ESPRESSO_SEQUENCER_L1_PROVIDER":              l1EthRpcURL.String(),
			"ESPRESSO_SEQUENCER_L1_POLLING_INTERVAL":      "30ms",
			"ESPRESSO_SEQUENCER_DATABASE_MAX_CONNECTIONS": "25",
			"ESPRESSO_SEQUENCER_STORAGE_PATH":             "/data/espresso",
			"RUST_LOG":                                    "info",

			"ESPRESSO_BUILDER_PORT":       portRemapping[ESPRESSO_BUILDER_PORT],
			"ESPRESSO_SEQUENCER_API_PORT": portRemapping[ESPRESSO_SEQUENCER_API_PORT],
			"ESPRESSO_DEV_NODE_PORT":      portRemapping[ESPRESSO_DEV_NODE_PORT],

			// We preallocate L1 deployments
			"ESPRESSO_DEV_NODE_L1_DEPLOYMENT": "skip",
			// This is a workaround for devnode not picking up stake table
			// initial state when it's baked into the genesis block. This
			// results in HotShot stalling when transitioning to epoch 3,
			// where staking reward distribution starts. Setting epoch
			// height to a very big number ensures we don't run into this
			// stalling problem during our tests, as we'll never reach
			// epoch 3.
			"ESPRESSO_DEV_NODE_EPOCH_HEIGHT": fmt.Sprint(uint64(math.MaxUint64)),
		},
		Ports: []string{
			portRemapping[ESPRESSO_BUILDER_PORT],
			portRemapping[ESPRESSO_SEQUENCER_API_PORT],
			portRemapping[ESPRESSO_DEV_NODE_PORT],
		},
	}

	// Add name:address pairs to dockerConfig environment
	for address, account := range ESPRESSO_ALLOCS {
		if account.Name != "" {
			dockerConfig.Environment[account.Name] = hexutil.Encode(address[:])
		}
	}

	return dockerConfig, portRemapping, nil
}

// determineDockerNetworkMode is a helper function that determines the
// docker network mode to use for the container.
//
// We launch in network mode host on linux, otherwise the container is not able
// to communicate with the host system. We use host.docker.internal to do this
// on platforms that are not running natively on linux, as this special address
// achieves the same result.  But on linux, this does not work, and we need to
// run on the host instead.
func determineDockerNetworkMode() string {
	if isRunningOnLinux {
		return "host"
	}

	return ""
}

// ensureHardCodedPortsAreMappedFromTheirOriginalValues is a convenience
// function that makes sure that hard coded ports are associated with their
// remapped port values.  This is done for convenience in order to ensure that
// we can still reference the hard coded ports, even if they've been remapped
// from their original values.
func ensureHardCodedPortsAreMappedFromTheirOriginalValues(containerInfo *DockerContainerInfo, portRemapping map[string]string) {
	if _, ok := portRemapping[ESPRESSO_SEQUENCER_API_PORT]; !ok {
		// If we don't have the original port mapping for the hard
		// coded port, we will need to back fill them in, just
		// to make life easier for consumers.

		for portKey, portValue := range portRemapping {
			// We copy the port mapping information
			// so we know the original mapping again,
			// since we're hard-coding the ports to use.
			// This should allow us to run multiple
			// e2e test environments in parallel on
			// linux as well.
			containerInfo.PortMap[portKey] = containerInfo.PortMap[portValue]
		}
	}
}

// launchEspressoDevNodeDocker is DevNetLauncherOption that launches th
// Espresso Dev Node within a Docker container.  It also ensures that the
// Espresso Dev Node is actively producing blocks before returning.
func launchEspressoDevNodeStartOption(ct *DevNetLauncherContext) e2esys.StartOption {
	return e2esys.StartOption{
		Role: "launch-espresso-dev-node",
		BatcherMod: func(c *batcher.CLIConfig, sys *e2esys.System) {
			if ct.Error != nil {
				// Early Return if we already have an Error set
				return
			}

			l1EthRpcURLPtr, err := url.Parse(c.L1EthRpc)
			if err != nil {
				ct.Error = FailedToDetermineL1RPCURL{Cause: err}
				return
			}

			network := determineDockerNetworkMode()

			// Let's spin up the espresso-dev-node
			l1EthRpcURL, err := translateContainerToNodeURL(*l1EthRpcURLPtr, network)
			if err != nil {
				ct.Error = err
				return
			}

			dockerConfig, portRemapping, err := determineEspressoDevNodeDockerContainerConfig(l1EthRpcURL, network)
			if err != nil {
				ct.Error = err
				return
			}

			containerCli := new(DockerCli)

			espressoDevNodeContainerInfo, err := containerCli.LaunchContainer(ct.Ctx, dockerConfig)

			if err != nil {
				ct.Error = FailedToLaunchDockerContainer{Cause: err}
				return
			}

			ensureHardCodedPortsAreMappedFromTheirOriginalValues(&espressoDevNodeContainerInfo, portRemapping)

			if err := waitForEspressoToFinishSpinningUp(ct, espressoDevNodeContainerInfo); err != nil {
				ct.Error = err
				return
			}

			// This skip on error check **SHOULD** be safe as this was
			// already performed inside the `waitForEspressoToFinishSpinningUp`
			// call.
			hostPort, _ := getContainerRemappedHostPort(espressoDevNodeContainerInfo.PortMap[ESPRESSO_SEQUENCER_API_PORT][0])

			espressoDevNode := &EspressoDevNodeDockerContainerInfo{
				DockerContainerInfo: espressoDevNodeContainerInfo,
				espressoUrls:        []string{"http://" + hostPort},
			}
			ct.EspressoDevNode = espressoDevNode

			c.EspressoUrls = espressoDevNode.espressoUrls
			c.LogConfig.Level = slog.LevelDebug
			c.TestingEspressoBatcherPrivateKey = "0x" + config.ESPRESSO_PRE_APPROVED_BATCHER_PRIVATE_KEY
			c.EspressoLightClientAddr = ESPRESSO_LIGHT_CLIENT_ADDRESS
		},
	}

}

// launchEspressoDevNodeDocker is DevNetLauncherOption that launches th
// Espresso Dev Node within a Docker container.  It also ensures that the
// Espresso Dev Node is actively producing blocks before returning.
func launchEspressoDevNodeDocker() DevNetLauncherOption {
	return func(ct *DevNetLauncherContext) E2eSystemOption {
		return E2eSystemOption{
			StartOptions: []e2esys.StartOption{
				launchEspressoDevNodeStartOption(ct),
			},
		}
	}
}

// StopConfig represents the configuration options for the Stop function.
// The configuration options help to define how the Stop function should
// to failure types.
type StopConfig struct {
	IgnoreErrors bool
	Ctx          context.Context
}

// StopOption is a functional option that allows for the modification of the
// Stop Config
type StopOption func(*StopConfig)

// IgnoreStopErrors is a functional option that ignores errors encountered
// by the stop function, so that they do not cause test failure
func IgnoreStopErrors(c *StopConfig) {
	c.IgnoreErrors = true
}

// Stop is a convenience method to handle the graceful shutdown, and the errors
// thereof of any node that should be stopped on test exit.
// There are different type signatures for the shutdown methods, and this
// aims to handle each of them as gracefully as possible while still ensuring
// that any returned errors are handled accordingly.
func Stop(t *testing.T, toStop any, options ...StopOption) {
	config := StopConfig{
		Ctx: context.Background(),
	}

	for _, opt := range options {
		opt(&config)
	}

	ctx := config.Ctx
	if cast, castOk := toStop.(interface{ Stop() error }); castOk {
		if have, want := cast.Stop(), error(nil); have != want && !config.IgnoreErrors {
			t.Fatalf("failed to stop node:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"\n", have, want)
		}

		return

	}

	if cast, castOk := toStop.(interface{ Stop(context.Context) error }); castOk {
		if have, want := cast.Stop(ctx), error(nil); have != want && !config.IgnoreErrors {
			t.Fatalf("failed to stop node:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"\n", have, want)
		}

		return
	}

	if cast, castOk := toStop.(interface{ Close() }); castOk {
		cast.Close()
		return
	}

	if cast, castOk := toStop.(interface{ Close(context.Context) }); castOk {
		cast.Close(ctx)
		return
	}

	if cast, castOk := toStop.(interface{ Close(context.Context) error }); castOk {
		if have, want := cast.Close(ctx), error(nil); have != want && !config.IgnoreErrors {
			t.Fatalf("failed to stop node:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"\n", have, want)
		}

		return
	}

	t.Fatalf("unable to determine how to stop the given node")
}

// Waits for an Espresso transaction to be confirmed using its hash.
func WaitForEspressoTx(ctx context.Context, txHash *espressoCommon.TaggedBase64, espressoClient *espressoClient.MultipleNodesClient) error {

	const transactionFetchTimeout = 4 * time.Second
	const transactionFetchInterval = 100 * time.Millisecond

	timer := time.NewTimer(transactionFetchTimeout)
	defer timer.Stop()

	ticker := time.NewTicker(transactionFetchInterval)
	defer ticker.Stop()

	var err error
	for {
		select {
		case <-ticker.C:
			_, err := espressoClient.FetchTransactionByHash(ctx, txHash)
			if err == nil {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("failed to fetch transaction by hash: %w", err)
		case <-ctx.Done():
			return nil
		}
	}
}
