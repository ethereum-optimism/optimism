package kurtosis

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	devnetTypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	apiInterfaces "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/run"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/deployer"
	srcInterfaces "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	autofixTypes "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/types"
)

const (
	DefaultPackageName = "github.com/ethpandaops/optimism-package"
	DefaultEnclave     = "devnet"

	// static URL for kurtosis reverse proxy
	defaultKurtosisReverseProxyURL = "http://127.0.0.1:9730"
)

// KurtosisEnvironment represents the output of a Kurtosis deployment
type KurtosisEnvironment struct {
	*descriptors.DevnetEnvironment
}

// KurtosisDeployer handles deploying packages using Kurtosis
type KurtosisDeployer struct {
	// Base directory where the deployment commands should be executed
	baseDir string
	// Package name to deploy
	packageName string
	// Dry run mode
	dryRun bool
	// Enclave name
	enclave string

	// interfaces for kurtosis sources
	enclaveSpec      srcInterfaces.EnclaveSpecifier
	enclaveInspecter srcInterfaces.EnclaveInspecter
	enclaveObserver  srcInterfaces.EnclaveObserver
	jwtExtractor     srcInterfaces.JWTExtractor
	depsetExtractor  srcInterfaces.DepsetExtractor

	// interface for kurtosis interactions
	kurtosisCtx apiInterfaces.KurtosisContextInterface

	// autofix mode
	autofixMode autofixTypes.AutofixMode
}

type KurtosisDeployerOptions func(*KurtosisDeployer)

func WithKurtosisBaseDir(baseDir string) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.baseDir = baseDir
	}
}

func WithKurtosisPackageName(packageName string) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.packageName = packageName
	}
}

func WithKurtosisDryRun(dryRun bool) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.dryRun = dryRun
	}
}

func WithKurtosisEnclave(enclave string) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.enclave = enclave
	}
}

func WithKurtosisEnclaveSpec(enclaveSpec srcInterfaces.EnclaveSpecifier) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.enclaveSpec = enclaveSpec
	}
}

func WithKurtosisEnclaveInspecter(enclaveInspecter srcInterfaces.EnclaveInspecter) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.enclaveInspecter = enclaveInspecter
	}
}

func WithKurtosisEnclaveObserver(enclaveObserver srcInterfaces.EnclaveObserver) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.enclaveObserver = enclaveObserver
	}
}

func WithKurtosisJWTExtractor(extractor srcInterfaces.JWTExtractor) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.jwtExtractor = extractor
	}
}

func WithKurtosisDepsetExtractor(extractor srcInterfaces.DepsetExtractor) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.depsetExtractor = extractor
	}
}

func WithKurtosisKurtosisContext(kurtosisCtx apiInterfaces.KurtosisContextInterface) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.kurtosisCtx = kurtosisCtx
	}
}

func WithKurtosisAutofixMode(autofixMode autofixTypes.AutofixMode) KurtosisDeployerOptions {
	return func(d *KurtosisDeployer) {
		d.autofixMode = autofixMode
	}
}

// NewKurtosisDeployer creates a new KurtosisDeployer instance
func NewKurtosisDeployer(opts ...KurtosisDeployerOptions) (*KurtosisDeployer, error) {
	d := &KurtosisDeployer{
		baseDir:     ".",
		packageName: DefaultPackageName,
		dryRun:      false,
		enclave:     DefaultEnclave,

		enclaveSpec:      &enclaveSpecAdapter{},
		enclaveInspecter: &enclaveInspectAdapter{},
		enclaveObserver:  &enclaveDeployerAdapter{},
		jwtExtractor:     &enclaveJWTAdapter{},
		depsetExtractor:  &enclaveDepsetAdapter{},
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.kurtosisCtx == nil {
		var err error
		d.kurtosisCtx, err = wrappers.GetDefaultKurtosisContext()
		if err != nil {
			return nil, fmt.Errorf("failed to create Kurtosis context: %w", err)
		}
	}

	return d, nil
}

func (d *KurtosisDeployer) getWallets(wallets deployer.WalletList) descriptors.WalletMap {
	walletMap := make(descriptors.WalletMap)
	for _, wallet := range wallets {
		walletMap[wallet.Name] = &descriptors.Wallet{
			Address:    devnetTypes.Address(wallet.Address),
			PrivateKey: wallet.PrivateKey,
		}
	}
	return walletMap
}

// GetEnvironmentInfo parses the input spec and inspect output to create KurtosisEnvironment
func (d *KurtosisDeployer) GetEnvironmentInfo(ctx context.Context, s *spec.EnclaveSpec) (*KurtosisEnvironment, error) {
	inspectResult, err := d.enclaveInspecter.EnclaveInspect(ctx, d.enclave)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	// Get contract addresses
	deployerData, err := d.enclaveObserver.EnclaveObserve(ctx, d.enclave)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployer state: %w", err)
	}

	// Get JWT data
	jwtData, err := d.jwtExtractor.ExtractData(ctx, d.enclave)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JWT data: %w", err)
	}

	// Get dependency set
	var depsets map[string]descriptors.DepSet
	if s.Features.Contains(spec.FeatureInterop) {
		depsets, err = d.depsetExtractor.ExtractData(ctx, d.enclave)
		if err != nil {
			return nil, fmt.Errorf("failed to extract dependency set: %w", err)
		}
	}

	env := &KurtosisEnvironment{
		DevnetEnvironment: &descriptors.DevnetEnvironment{
			Name:            d.enclave,
			ReverseProxyURL: defaultKurtosisReverseProxyURL,

			L2:       make([]*descriptors.L2Chain, 0, len(s.Chains)),
			Features: s.Features,
			DepSets:  depsets,
		},
	}

	// Find L1 endpoint
	finder := NewServiceFinder(
		inspectResult.UserServices,
		WithL1Chain(&spec.ChainSpec{
			NetworkID: deployerData.L1ChainID,
			Name:      "Ethereum",
		}),
		WithL2Chains(s.Chains),
		WithDepSets(depsets),
	)
	if nodes, services := finder.FindL1Services(); len(nodes) > 0 {
		chain := &descriptors.Chain{
			ID:        deployerData.L1ChainID,
			Name:      "Ethereum",
			Services:  services,
			Nodes:     nodes,
			JWT:       jwtData.L1JWT,
			Addresses: descriptors.AddressMap(deployerData.State.Addresses),
			Wallets:   d.getWallets(deployerData.L1ValidatorWallets),
			Config:    deployerData.L1ChainConfig,
		}
		if deployerData.State != nil {
			chain.Addresses = descriptors.AddressMap(deployerData.State.Addresses)
			chain.Wallets = d.getWallets(deployerData.L1ValidatorWallets)
		}
		env.L1 = chain
	}

	// Find L2 endpoints
	for _, chainSpec := range s.Chains {
		nodes, services := finder.FindL2Services(chainSpec)

		chain := &descriptors.L2Chain{
			Chain: &descriptors.Chain{
				Name:     chainSpec.Name,
				ID:       chainSpec.NetworkID,
				Services: services,
				Nodes:    nodes,
				JWT:      jwtData.L2JWT,
			},
		}

		// Add contract addresses if available
		if deployerData.State != nil && deployerData.State.Deployments != nil {
			if deployment, ok := deployerData.State.Deployments[chainSpec.NetworkID]; ok {
				chain.L1Addresses = descriptors.AddressMap(deployment.L1Addresses)
				chain.Addresses = descriptors.AddressMap(deployment.L2Addresses)
				chain.Config = deployment.Config
				chain.RollupConfig = deployment.RollupConfig
				chain.Wallets = d.getWallets(deployment.L2Wallets)
				chain.L1Wallets = d.getWallets(deployment.L1Wallets)
			}
		}

		env.L2 = append(env.L2, chain)
	}

	return env, nil
}

// Deploy executes the Kurtosis deployment command with the provided input
func (d *KurtosisDeployer) Deploy(ctx context.Context, input io.Reader) (*spec.EnclaveSpec, error) {
	// Parse the input spec first
	inputCopy := new(bytes.Buffer)
	tee := io.TeeReader(input, inputCopy)

	spec, err := d.enclaveSpec.EnclaveSpec(tee)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input spec: %w", err)
	}

	// Run kurtosis command
	kurtosisRunner, err := run.NewKurtosisRunner(
		run.WithKurtosisRunnerDryRun(d.dryRun),
		run.WithKurtosisRunnerEnclave(d.enclave),
		run.WithKurtosisRunnerKurtosisContext(d.kurtosisCtx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kurtosis runner: %w", err)
	}

	if err := kurtosisRunner.Run(ctx, d.packageName, inputCopy); err != nil {
		return nil, err
	}

	// If dry run, return empty environment
	if d.dryRun {
		return spec, nil
	}

	// Get environment information
	return spec, nil
}
