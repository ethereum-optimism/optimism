package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultPresetName = "minimal"

	devstackL2ELKindEnvVar       = "DEVSTACK_L2EL_KIND"
	devstackL2CLKindEnvVar       = "DEVSTACK_L2CL_KIND"
	sysgoMetricsEnabledEnvVar    = "SYSGO_METRICS_ENABLED"
	defaultInteropActivationSecs = uint64(2)
)

type opUpConfig struct {
	Dir           string
	Preset        string
	LegacyInterop bool
	L2ELKind      string
	L2CLKind      string
	Metrics       bool
}

type devnetPreset struct {
	Name        string
	Aliases     []string
	Summary     string
	Description string
	Build       func(*testingT) (*runningDevnet, error)
}

type runningDevnet struct {
	L1EL               *devnetEndpoint
	L1CL               *devnetEndpoint
	L2Networks         []*namedL2Network
	L2ELs              []*devnetEndpoint
	L2CLs              []*devnetEndpoint
	Services           []*devnetEndpoint
	ControlledServices []*controlledService
	Contracts          []*contractSet
	ExportDepset       bool
	BackgroundLoggers  []func(context.Context, io.Writer)
	Notes              []string
}

type devnetEndpoint struct {
	Name       string
	Layer      string
	ChainID    eth.ChainID
	ChainLabel string
	RPC        opclient.RPC
	BackendURL string
	DirectURL  string
}

type namedL2Network struct {
	Name    string
	Network *dsl.L2Network
}

type contractSet struct {
	Network   string
	ChainID   eth.ChainID
	Contracts []contractAddress
}

type contractAddress struct {
	Name    string
	Address common.Address
}

type localEndpoint struct {
	*devnetEndpoint
	LocalURL string
	Listener net.Listener
}

type localEndpointRequest struct {
	Endpoint *devnetEndpoint
}

var devnetPresets = []*devnetPreset{
	{
		Name:        defaultPresetName,
		Aliases:     []string{"single", "single-chain"},
		Summary:     "single L2 with one sequencer, batcher, proposer, and dev faucets",
		Description: "The fastest general-purpose OP Stack devnet. This is the default.",
		Build:       buildMinimalDevnet,
	},
	{
		Name:        "interop",
		Aliases:     []string{"two-l2-interop", "supernode-interop"},
		Summary:     "two L2s with interop enabled and a shared op-supernode",
		Description: "Use this when testing cross-chain messages or op-supernode behavior.",
		Build:       buildInteropDevnet,
	},
	{
		Name:        "two-l2",
		Aliases:     []string{"supernode", "multi-chain"},
		Summary:     "two L2s backed by a shared op-supernode, without interop activation",
		Description: "Useful for tooling that needs multiple L2 RPCs without interop semantics.",
		Build:       buildTwoL2Devnet,
	},
	{
		Name:        "multinode",
		Aliases:     []string{"multi-node", "singlechain-multinode"},
		Summary:     "single L2 with a sequencer and verifier node",
		Description: "Exercises a small sequencer/verifier topology on one L2.",
		Build:       buildMultinodeDevnet,
	},
	{
		Name:        "two-verifiers",
		Aliases:     []string{"singlechain-twoverifiers"},
		Summary:     "single L2 with a sequencer and two verifier nodes",
		Description: "A larger single-chain topology for verifier and P2P behavior.",
		Build:       buildTwoVerifiersDevnet,
	},
	{
		Name:        "conductors",
		Aliases:     []string{"conductor", "failover"},
		Summary:     "single L2 with an op-conductor cluster around the sequencer",
		Description: "Use this for local sequencer failover and conductor workflows.",
		Build:       buildConductorsDevnet,
	},
	{
		Name:        "flashblocks",
		Aliases:     []string{"flashblock"},
		Summary:     "single L2 with rollup-boost and op-rbuilder flashblocks components",
		Description: "Requires the local flashblocks-capable binaries expected by devstack.",
		Build:       buildFlashblocksDevnet,
	},
}

func resolvePreset(name string) (*devnetPreset, error) {
	normalized := normalizePresetName(name)
	for _, spec := range devnetPresets {
		if normalized == spec.Name {
			return spec, nil
		}
		for _, alias := range spec.Aliases {
			if normalized == alias {
				return spec, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown preset %q (run `op-up presets` to list available presets)", name)
}

func normalizePresetName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func printPresetList(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "PRESET\tALIASES\tDESCRIPTION"); err != nil {
		return err
	}
	for _, spec := range devnetPresets {
		aliases := "-"
		if len(spec.Aliases) > 0 {
			aliases = strings.Join(spec.Aliases, ", ")
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", spec.Name, aliases, spec.Summary); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "\nStart one with `op-up --preset <name>` or `op-up up <name>`.")
	return err
}

func runSelectedDevnet(ctx context.Context, stdout io.Writer, stderr io.Writer, cfg opUpConfig, spec *devnetPreset, tempRoot string, devnet *runningDevnet, runID string, logFilePath string) error {
	fmt.Fprintf(stderr, "\nPreset: %s\n", spec.Name)
	fmt.Fprintf(stderr, "%s\n\n", spec.Description)

	if err := printAccountInfo(stderr); err != nil {
		return err
	}

	endpoints, err := devnet.localEndpoints()
	if err != nil {
		return err
	}
	defer closeLocalEndpointListeners(endpoints)
	devnet.applyControlEndpointURLs(endpoints)

	configExport, err := exportDevnetConfigs(ctx, cfg, spec, tempRoot, devnet, endpoints)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Config files: %s\n", configExport.Dir)
	fmt.Fprintf(stderr, "\nConfig files: %s\n", configExport.Dir)
	if logFilePath != "" {
		fmt.Fprintf(stdout, "Logs: %s\n", logFilePath)
	}
	controlServer, err := newControlServer(ctx, cfg, spec, devnet.ControlledServices, runID, logFilePath)
	if err != nil {
		return err
	}
	defer controlServer.Close()

	if len(endpoints) > 0 {
		fmt.Fprintln(stderr, "\nServices / Endpoints:")
		controlIDs := devnet.controlIDsByEndpoint()
		for _, endpoint := range endpoints {
			fmt.Fprintf(stderr, "  %-14s %-4s control=%-18s chain=%-12s %s\n", endpoint.Name, endpoint.Layer, controlIDs[endpoint.devnetEndpoint], endpoint.Chain(), endpoint.LocalURL)
		}
	}
	if controlServer != nil {
		fmt.Fprintf(stderr, "\nControl: op-up control services --session %s\n", controlServer.session.ID)
	}
	if len(devnet.Contracts) > 0 {
		fmt.Fprintln(stderr, "\nDeployed Contracts:")
		for _, contracts := range devnet.Contracts {
			fmt.Fprintf(stderr, "  %s chain=%s\n", contracts.Network, contracts.ChainID)
			for _, contract := range contracts.Contracts {
				fmt.Fprintf(stderr, "    %-34s %s\n", contract.Name, contract.Address)
			}
		}
	}

	if cfg.Metrics {
		fmt.Fprintln(stderr, "\nMetrics: Grafana should be available at http://localhost:3000 when sysgo metrics finish starting.")
	}
	for _, note := range devnet.Notes {
		fmt.Fprintf(stderr, "\n%s\n", note)
	}
	fmt.Fprintln(stderr, "\nPress Ctrl+C to stop the devnet and clean up resources.")

	errCh := make(chan error, len(endpoints))
	controlledByEndpoint := devnet.controlledServiceByEndpoint()
	for _, endpoint := range endpoints {
		if endpoint.Listener == nil || endpoint.RPC == nil {
			continue
		}
		endpoint := endpoint
		controlled := controlledByEndpoint[endpoint.devnetEndpoint]
		go func() {
			errCh <- proxyEL(ctx, stderr, endpoint.Listener, endpoint.RPC, endpoint.BackendURL, controlled)
		}()
	}
	for _, logger := range devnet.BackgroundLoggers {
		go logger(ctx, stderr)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (d *runningDevnet) localEndpoints() ([]*localEndpoint, error) {
	var requests []localEndpointRequest
	if d.L1EL != nil {
		requests = appendLocalEndpointRequests(requests, []*devnetEndpoint{d.L1EL})
	}
	if d.L1CL != nil {
		requests = appendLocalEndpointRequests(requests, []*devnetEndpoint{d.L1CL})
	}
	if len(d.L2ELs) > 0 {
		requests = appendLocalEndpointRequests(requests, d.L2ELs)
	}
	if len(d.L2CLs) > 0 {
		requests = appendLocalEndpointRequests(requests, d.L2CLs)
	}
	if len(d.Services) > 0 {
		requests = appendLocalEndpointRequests(requests, d.Services)
	}
	return bindLocalEndpoints(requests)
}

func (d *runningDevnet) applyControlEndpointURLs(endpoints []*localEndpoint) {
	for _, svc := range d.ControlledServices {
		if svc == nil || svc.Endpoint == nil {
			continue
		}
		for _, endpoint := range endpoints {
			if endpoint != nil && endpoint.devnetEndpoint == svc.Endpoint {
				svc.EndpointURL = endpoint.LocalURL
				break
			}
		}
	}
}

func (d *runningDevnet) controlIDsByEndpoint() map[*devnetEndpoint]string {
	out := make(map[*devnetEndpoint]string)
	for _, svc := range d.ControlledServices {
		if svc != nil && svc.Endpoint != nil {
			out[svc.Endpoint] = svc.ID
		}
	}
	return out
}

func (d *runningDevnet) controlledServiceByEndpoint() map[*devnetEndpoint]*controlledService {
	out := make(map[*devnetEndpoint]*controlledService)
	for _, svc := range d.ControlledServices {
		if svc != nil && svc.Endpoint != nil {
			out[svc.Endpoint] = svc
		}
	}
	return out
}

func appendLocalEndpointRequests(out []localEndpointRequest, endpoints []*devnetEndpoint) []localEndpointRequest {
	for _, endpoint := range endpoints {
		out = append(out, localEndpointRequest{Endpoint: endpoint})
	}
	return out
}

func bindLocalEndpoints(requests []localEndpointRequest) ([]*localEndpoint, error) {
	out := make([]*localEndpoint, len(requests))
	for i, request := range requests {
		if request.Endpoint.DirectURL != "" {
			out[i] = &localEndpoint{
				devnetEndpoint: request.Endpoint,
				LocalURL:       request.Endpoint.DirectURL,
			}
			continue
		}
		listener, port, err := listenOnRandomLocalPort()
		if err != nil {
			closeLocalEndpointListeners(out)
			return nil, err
		}
		out[i] = newLocalEndpoint(request.Endpoint, port, listener)
	}

	return out, nil
}

func newLocalEndpoint(endpoint *devnetEndpoint, port uint, listener net.Listener) *localEndpoint {
	return &localEndpoint{
		devnetEndpoint: endpoint,
		LocalURL:       localURL(port),
		Listener:       listener,
	}
}

func listenOnRandomLocalPort() (net.Listener, uint, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, 0, fmt.Errorf("listen on a random localhost port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return listener, uint(port), nil
}

func closeLocalEndpointListeners(endpoints []*localEndpoint) {
	for _, endpoint := range endpoints {
		if endpoint != nil && endpoint.Listener != nil {
			_ = endpoint.Listener.Close()
		}
	}
}

func localURL(port uint) string {
	return fmt.Sprintf("http://localhost:%d", port)
}

func applyRuntimeOverrides(cfg opUpConfig) (func(), error) {
	restore := make([]func(), 0, 3)
	set := func(key, val string) {
		old, ok := os.LookupEnv(key)
		restore = append(restore, func() {
			if ok {
				_ = os.Setenv(key, old)
			} else {
				_ = os.Unsetenv(key)
			}
		})
		_ = os.Setenv(key, val)
	}

	if cfg.L2ELKind != "" {
		switch cfg.L2ELKind {
		case string(sysgo.MixedL2ELOpGeth), string(sysgo.MixedL2ELOpReth), string(sysgo.MixedL2ELOpRethV2):
			set(devstackL2ELKindEnvVar, cfg.L2ELKind)
		default:
			return nil, fmt.Errorf("unsupported L2 EL kind %q (expected op-geth, op-reth, or op-reth-proof-v2)", cfg.L2ELKind)
		}
	}
	if cfg.L2CLKind != "" {
		switch cfg.L2CLKind {
		case string(sysgo.MixedL2CLOpNode), string(sysgo.MixedL2CLKona):
			set(devstackL2CLKindEnvVar, cfg.L2CLKind)
		default:
			return nil, fmt.Errorf("unsupported L2 CL kind %q (expected op-node or kona-node)", cfg.L2CLKind)
		}
	}
	if cfg.Metrics {
		set(sysgoMetricsEnabledEnvVar, "true")
	}

	return func() {
		for _, fn := range slices.Backward(restore) {
			fn()
		}
	}, nil
}

func buildMinimalDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := newMinimalSystem(t)
	if err != nil {
		return nil, err
	}
	return singleChainDevnet(sys.L1EL, sys.L1CL, sys.L2Chain, sys.L2EL, sys.L2CL), nil
}

func buildInteropDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := newSupernodeInteropSystem(t)
	if err != nil {
		return nil, err
	}
	l2AEp := l2ELEndpoint("L2A", sys.L2ELA)
	l2BEp := l2ELEndpoint("L2B", sys.L2ELB)
	superEp := supernodeEndpoint("Supernode", sys.Supernode)
	return &runningDevnet{
		L1EL:       l1ELEndpoint("L1", sys.L1EL),
		L1CL:       l1CLEndpoint("L1 Beacon", sys.L1CL),
		L2Networks: []*namedL2Network{l2Network("L2A", sys.L2A), l2Network("L2B", sys.L2B)},
		L2ELs:      []*devnetEndpoint{l2AEp, l2BEp},
		Services:   []*devnetEndpoint{superEp},
		ControlledServices: []*controlledService{
			controlledL2EL("l2a-el", "L2A EL", sys.L2ELA, l2AEp),
			controlledL2EL("l2b-el", "L2B EL", sys.L2ELB, l2BEp),
			controlledSupernode("supernode", "Supernode", sys.Supernode, superEp),
		},
		Contracts: []*contractSet{
			contractsForL2("L2A", sys.L2A),
			contractsForL2("L2B", sys.L2B),
		},
		ExportDepset: true,
		BackgroundLoggers: []func(context.Context, io.Writer){
			func(ctx context.Context, stderr io.Writer) {
				logInterop(ctx, stderr, sys)
			},
		},
		Notes: []string{
			fmt.Sprintf("Interop activates %d seconds after genesis; run `op-up smoke-interop all --l2a-rpc <printed L2A EL URL> --l2b-rpc <printed L2B EL URL> --private-key <printed Test Account Private Key>` in another terminal for a cross-chain smoke test.", defaultInteropActivationSecs),
			"Supernode per-chain rollup RPCs are available at <printed Supernode RPC>/<chain-id>, for example /901 and /902.",
		},
	}, nil
}

func buildTwoL2Devnet(t *testingT) (*runningDevnet, error) {
	sys, err := buildPreset(func() *presets.TwoL2 {
		return presets.NewTwoL2Supernode(t)
	})
	if err != nil {
		return nil, err
	}
	l2AEL := sys.L2A.PublicRPC()
	l2BEL := sys.L2B.PublicRPC()
	l2AEp := l2ELEndpoint("L2A", l2AEL)
	l2BEp := l2ELEndpoint("L2B", l2BEL)
	l2ACLEp := l2CLEndpoint("L2A CL", sys.L2ACL)
	l2BCLEp := l2CLEndpoint("L2B CL", sys.L2BCL)
	return &runningDevnet{
		L1EL:       l1ELEndpoint("L1", sys.L1EL),
		L1CL:       l1CLEndpoint("L1 Beacon", sys.L1CL),
		L2Networks: []*namedL2Network{l2Network("L2A", sys.L2A), l2Network("L2B", sys.L2B)},
		L2ELs:      []*devnetEndpoint{l2AEp, l2BEp},
		L2CLs:      []*devnetEndpoint{l2ACLEp, l2BCLEp},
		ControlledServices: []*controlledService{
			controlledL2EL("l2a-el", "L2A EL", l2AEL, l2AEp),
			controlledL2EL("l2b-el", "L2B EL", l2BEL, l2BEp),
			controlledL2CL("l2a-cl", "L2A CL", sys.L2ACL, l2ACLEp),
			controlledL2CL("l2b-cl", "L2B CL", sys.L2BCL, l2BCLEp),
		},
		Contracts: []*contractSet{
			contractsForL2("L2A", sys.L2A),
			contractsForL2("L2B", sys.L2B),
		},
	}, nil
}

func buildMultinodeDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := buildPreset(func() *presets.SingleChainMultiNode {
		return presets.NewSingleChainMultiNodeWithoutCheck(t)
	})
	if err != nil {
		return nil, err
	}
	seqELEp := l2ELEndpoint("L2 sequencer", sys.L2EL)
	verELEp := l2ELEndpoint("L2 verifier", sys.L2ELB)
	seqCLEp := l2CLEndpoint("L2 sequencer CL", sys.L2CL)
	verCLEp := l2CLEndpoint("L2 verifier CL", sys.L2CLB)
	return &runningDevnet{
		L1EL:       l1ELEndpoint("L1", sys.L1EL),
		L1CL:       l1CLEndpoint("L1 Beacon", sys.L1CL),
		L2Networks: []*namedL2Network{l2Network("L2", sys.L2Chain)},
		L2ELs:      []*devnetEndpoint{seqELEp, verELEp},
		L2CLs:      []*devnetEndpoint{seqCLEp, verCLEp},
		ControlledServices: []*controlledService{
			controlledL2EL("l2-sequencer-el", "L2 sequencer EL", sys.L2EL, seqELEp),
			controlledL2EL("l2-verifier-el", "L2 verifier EL", sys.L2ELB, verELEp),
			controlledL2CL("l2-sequencer-cl", "L2 sequencer CL", sys.L2CL, seqCLEp),
			controlledL2CL("l2-verifier-cl", "L2 verifier CL", sys.L2CLB, verCLEp),
		},
		Contracts: []*contractSet{contractsForL2("L2", sys.L2Chain)},
	}, nil
}

func buildTwoVerifiersDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := buildPreset(func() *presets.SingleChainTwoVerifiers {
		return presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	})
	if err != nil {
		return nil, err
	}
	seqELEp := l2ELEndpoint("L2 sequencer", sys.L2EL)
	verBEL := l2ELEndpoint("L2 verifier B", sys.L2ELB)
	verCEL := l2ELEndpoint("L2 verifier C", sys.L2ELC)
	seqCLEp := l2CLEndpoint("L2 sequencer CL", sys.L2CL)
	verBCL := l2CLEndpoint("L2 verifier B CL", sys.L2CLB)
	verCCL := l2CLEndpoint("L2 verifier C CL", sys.L2CLC)
	return &runningDevnet{
		L1EL:       l1ELEndpoint("L1", sys.L1EL),
		L1CL:       l1CLEndpoint("L1 Beacon", sys.L1CL),
		L2Networks: []*namedL2Network{l2Network("L2", sys.L2Chain)},
		L2ELs:      []*devnetEndpoint{seqELEp, verBEL, verCEL},
		L2CLs:      []*devnetEndpoint{seqCLEp, verBCL, verCCL},
		ControlledServices: []*controlledService{
			controlledL2EL("l2-sequencer-el", "L2 sequencer EL", sys.L2EL, seqELEp),
			controlledL2EL("l2-verifier-b-el", "L2 verifier B EL", sys.L2ELB, verBEL),
			controlledL2EL("l2-verifier-c-el", "L2 verifier C EL", sys.L2ELC, verCEL),
			controlledL2CL("l2-sequencer-cl", "L2 sequencer CL", sys.L2CL, seqCLEp),
			controlledL2CL("l2-verifier-b-cl", "L2 verifier B CL", sys.L2CLB, verBCL),
			controlledL2CL("l2-verifier-c-cl", "L2 verifier C CL", sys.L2CLC, verCCL),
		},
		Contracts: []*contractSet{contractsForL2("L2", sys.L2Chain)},
	}, nil
}

func buildConductorsDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := buildPreset(func() *presets.MinimalWithConductors {
		return presets.NewMinimalWithConductors(t)
	})
	if err != nil {
		return nil, err
	}
	devnet := singleChainDevnet(sys.L1EL, sys.L1CL, sys.L2Chain, sys.L2EL, sys.L2CL)
	devnet.Notes = append(devnet.Notes, "op-conductor is running inside the preset; the stable L2 EL and CL proxies are exposed for transactions and rollup RPC calls.")
	return devnet, nil
}

func buildFlashblocksDevnet(t *testingT) (*runningDevnet, error) {
	sys, err := buildPreset(func() *presets.SingleChainWithFlashblocks {
		return presets.NewSingleChainWithFlashblocks(t)
	})
	if err != nil {
		return nil, err
	}
	devnet := singleChainDevnet(sys.L1EL, sys.L1CL, sys.L2Chain, sys.L2EL, sys.L2CL)
	devnet.Notes = append(devnet.Notes, "flashblocks support is running through rollup-boost and op-rbuilder in the selected preset.")
	return devnet, nil
}

func buildPreset[T any](fn func() T) (out T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			var failure testingFailure
			if errors.As(asError(recovered), &failure) {
				err = failure.err
				return
			}
			panic(recovered)
		}
	}()
	return fn(), nil
}

func singleChainDevnet(l1 *dsl.L1ELNode, l1CL *dsl.L1CLNode, l2 *dsl.L2Network, l2EL *dsl.L2ELNode, l2CL *dsl.L2CLNode) *runningDevnet {
	l2ELEp := l2ELEndpoint("L2", l2EL)
	l2CLEp := l2CLEndpoint("L2 CL", l2CL)
	return &runningDevnet{
		L1EL:       l1ELEndpoint("L1", l1),
		L1CL:       l1CLEndpoint("L1 Beacon", l1CL),
		L2Networks: []*namedL2Network{l2Network("L2", l2)},
		L2ELs:      []*devnetEndpoint{l2ELEp},
		L2CLs:      []*devnetEndpoint{l2CLEp},
		ControlledServices: []*controlledService{
			controlledL2EL("l2-el", "L2 EL", l2EL, l2ELEp),
			controlledL2CL("l2-cl", "L2 CL", l2CL, l2CLEp),
		},
		Contracts: []*contractSet{contractsForL2("L2", l2)},
	}
}

func l2Network(name string, l2 *dsl.L2Network) *namedL2Network {
	return &namedL2Network{Name: name, Network: l2}
}

func contractsForL2(name string, l2 *dsl.L2Network) *contractSet {
	deployment := l2.Escape().Deployment()
	contracts := []contractAddress{
		{Name: "OptimismPortalProxy", Address: l2.DepositContractAddr()},
		{Name: "SystemConfigProxy", Address: deployment.SystemConfigProxyAddr()},
		{Name: "L1StandardBridgeProxy", Address: deployment.L1StandardBridgeProxyAddr()},
		{Name: "DisputeGameFactoryProxy", Address: deployment.DisputeGameFactoryProxyAddr()},
	}
	if withProxyAdmin, ok := deployment.(interface{ ProxyAdminAddr() common.Address }); ok {
		contracts = appendNonZeroContract(contracts, "ProxyAdmin", withProxyAdmin.ProxyAdminAddr())
	}
	if withDelayedWETH, ok := deployment.(interface{ PermissionlessDelayedWETHProxyAddr() common.Address }); ok {
		contracts = appendNonZeroContract(contracts, "PermissionlessDelayedWETHProxy", withDelayedWETH.PermissionlessDelayedWETHProxyAddr())
	}
	return &contractSet{
		Network:   name,
		ChainID:   l2.ChainID(),
		Contracts: contracts,
	}
}

func appendNonZeroContract(contracts []contractAddress, name string, addr common.Address) []contractAddress {
	if addr == (common.Address{}) {
		return contracts
	}
	return append(contracts, contractAddress{Name: name, Address: addr})
}

func (e *devnetEndpoint) Chain() string {
	if e.ChainLabel != "" {
		return e.ChainLabel
	}
	return e.ChainID.String()
}

func l1ELEndpoint(name string, node *dsl.L1ELNode) *devnetEndpoint {
	return &devnetEndpoint{
		Name:    name,
		Layer:   "EL",
		ChainID: node.Escape().ChainID(),
		RPC:     node.EthClient().RPC(),
	}
}

func l1CLEndpoint(name string, node *dsl.L1CLNode) *devnetEndpoint {
	return &devnetEndpoint{
		Name:      name,
		Layer:     "HTTP",
		ChainID:   node.Escape().ChainID(),
		DirectURL: node.BeaconHTTPAddr(),
	}
}

func l2ELEndpoint(name string, node *dsl.L2ELNode) *devnetEndpoint {
	return &devnetEndpoint{
		Name:    name,
		Layer:   "EL",
		ChainID: node.Escape().ChainID(),
		RPC:     node.Escape().L2EthClient().RPC(),
	}
}

func controlledL2EL(id string, name string, node *dsl.L2ELNode, endpoint *devnetEndpoint) *controlledService {
	return controlledLifecycleService(id, name, "EL", endpoint.ChainID, endpoint, endpoint.RPC, node.Escape())
}

func l2CLEndpoint(name string, node *dsl.L2CLNode) *devnetEndpoint {
	return &devnetEndpoint{
		Name:    name,
		Layer:   "CL",
		ChainID: node.ChainID(),
		RPC:     node.Escape().ClientRPC(),
	}
}

func controlledL2CL(id string, name string, node *dsl.L2CLNode, endpoint *devnetEndpoint) *controlledService {
	return controlledLifecycleService(id, name, "CL", endpoint.ChainID, endpoint, endpoint.RPC, node.Escape())
}

func supernodeEndpoint(name string, node *dsl.Supernode) *devnetEndpoint {
	return &devnetEndpoint{
		Name:       name,
		Layer:      "RPC",
		ChainLabel: "shared",
		RPC:        node.ClientRPC(),
		BackendURL: node.UserRPC(),
	}
}

func controlledSupernode(id string, name string, node *dsl.Supernode, endpoint *devnetEndpoint) *controlledService {
	return controlledLifecycleService(id, name, "RPC", eth.ChainID{}, endpoint, endpoint.RPC, node.Escape())
}

func controlledLifecycleService(id string, name string, kind string, chainID eth.ChainID, endpoint *devnetEndpoint, rpc opclient.RPC, candidate any) *controlledService {
	control, _ := candidate.(stack.ControlledLifecycle)
	return &controlledService{
		ID:       id,
		Name:     name,
		Kind:     kind,
		ChainID:  chainID,
		Endpoint: endpoint,
		RPC:      rpc,
		Control:  control,
	}
}

func logRuntimeOverrides(stderr io.Writer, cfg opUpConfig) {
	var settings []string
	if cfg.L2ELKind != "" {
		settings = append(settings, fmt.Sprintf("%s=%s", devstackL2ELKindEnvVar, cfg.L2ELKind))
	}
	if cfg.L2CLKind != "" {
		settings = append(settings, fmt.Sprintf("%s=%s", devstackL2CLKindEnvVar, cfg.L2CLKind))
	}
	if cfg.Metrics {
		settings = append(settings, fmt.Sprintf("%s=true", sysgoMetricsEnabledEnvVar))
	}
	if len(settings) > 0 {
		fmt.Fprintf(stderr, "Runtime overrides: %s\n", strings.Join(settings, " "))
	}
}
