package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/enclave"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/engine"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	autofixTypes "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/types"
)

type EngineManager interface {
	EnsureRunning() error
	GetEngineType() (string, error)
}

type deployer interface {
	Deploy(ctx context.Context, input io.Reader) (*spec.EnclaveSpec, error)
	GetEnvironmentInfo(ctx context.Context, spec *spec.EnclaveSpec) (*kurtosis.KurtosisEnvironment, error)
}

type DeployerFunc func(opts ...kurtosis.KurtosisDeployerOptions) (deployer, error)

type DeployerOption func(*Deployer)

type Deployer struct {
	baseDir        string
	dryRun         bool
	kurtosisPkg    string
	enclave        string
	kurtosisBinary string
	ktDeployer     DeployerFunc
	engineManager  EngineManager
	templateFile   string
	dataFile       string
	newEnclaveFS   func(ctx context.Context, enclave string, opts ...ktfs.EnclaveFSOption) (*ktfs.EnclaveFS, error)
	enclaveManager *enclave.KurtosisEnclaveManager
	autofixMode    autofixTypes.AutofixMode
}

func WithKurtosisDeployer(ktDeployer DeployerFunc) DeployerOption {
	return func(d *Deployer) {
		d.ktDeployer = ktDeployer
	}
}

func WithEngineManager(engineManager EngineManager) DeployerOption {
	return func(d *Deployer) {
		d.engineManager = engineManager
	}
}

func WithKurtosisBinary(kurtosisBinary string) DeployerOption {
	return func(d *Deployer) {
		d.kurtosisBinary = kurtosisBinary
	}
}

func WithKurtosisPackage(kurtosisPkg string) DeployerOption {
	return func(d *Deployer) {
		d.kurtosisPkg = kurtosisPkg
	}
}

func WithTemplateFile(templateFile string) DeployerOption {
	return func(d *Deployer) {
		d.templateFile = templateFile
	}
}

func WithDataFile(dataFile string) DeployerOption {
	return func(d *Deployer) {
		d.dataFile = dataFile
	}
}

func WithBaseDir(baseDir string) DeployerOption {
	return func(d *Deployer) {
		d.baseDir = baseDir
	}
}

func WithDryRun(dryRun bool) DeployerOption {
	return func(d *Deployer) {
		d.dryRun = dryRun
	}
}

func WithEnclave(enclave string) DeployerOption {
	return func(d *Deployer) {
		d.enclave = enclave
	}
}

func WithAutofixMode(autofixMode autofixTypes.AutofixMode) DeployerOption {
	return func(d *Deployer) {
		d.autofixMode = autofixMode
	}
}

func WithNewEnclaveFSFunc(newEnclaveFS func(ctx context.Context, enclave string, opts ...ktfs.EnclaveFSOption) (*ktfs.EnclaveFS, error)) DeployerOption {
	return func(d *Deployer) {
		d.newEnclaveFS = newEnclaveFS
	}
}

func NewDeployer(opts ...DeployerOption) (*Deployer, error) {
	d := &Deployer{
		kurtosisBinary: "kurtosis",
		ktDeployer: func(opts ...kurtosis.KurtosisDeployerOptions) (deployer, error) {
			return kurtosis.NewKurtosisDeployer(opts...)
		},
		newEnclaveFS: ktfs.NewEnclaveFS,
	}
	for _, opt := range opts {
		opt(d)
	}

	if d.engineManager == nil {
		d.engineManager = engine.NewEngineManager(engine.WithKurtosisBinary(d.kurtosisBinary))
	}

	if !d.dryRun {
		if err := d.engineManager.EnsureRunning(); err != nil {
			return nil, fmt.Errorf("error ensuring kurtosis engine is running: %w", err)
		}

		// Get and log engine info
		engineType, err := d.engineManager.GetEngineType()
		if err != nil {
			log.Printf("Warning: failed to get engine type: %v", err)
		} else {
			log.Printf("Kurtosis engine type: %s", engineType)
		}
		var enclaveManager *enclave.KurtosisEnclaveManager
		if engineType == "docker" {
			enclaveManager, err = enclave.NewKurtosisEnclaveManager(
				enclave.WithDockerManager(&enclave.DefaultDockerManager{}),
			)
		} else {
			enclaveManager, err = enclave.NewKurtosisEnclaveManager()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create enclave manager: %w", err)
		}
		d.enclaveManager = enclaveManager
	} else {
		// This allows the deployer to work in dry run mode without a running Kurtosis engine
		log.Printf("No Kurtosis engine running, skipping enclave manager creation")
	}

	return d, nil
}

func (d *Deployer) deployEnvironment(ctx context.Context, r io.Reader) (*kurtosis.KurtosisEnvironment, error) {
	// Create a multi reader to output deployment input to stdout
	buf := bytes.NewBuffer(nil)
	tee := io.TeeReader(r, buf)

	// Log the deployment input
	log.Println("Deployment input:")
	if _, err := io.Copy(os.Stdout, tee); err != nil {
		return nil, fmt.Errorf("error copying deployment input: %w", err)
	}

	opts := []kurtosis.KurtosisDeployerOptions{
		kurtosis.WithKurtosisBaseDir(d.baseDir),
		kurtosis.WithKurtosisDryRun(d.dryRun),
		kurtosis.WithKurtosisPackageName(d.kurtosisPkg),
		kurtosis.WithKurtosisEnclave(d.enclave),
		kurtosis.WithKurtosisAutofixMode(d.autofixMode),
	}

	ktd, err := d.ktDeployer(opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating kurtosis deployer: %w", err)
	}

	spec, err := ktd.Deploy(ctx, buf)
	if err != nil {
		return nil, fmt.Errorf("error deploying kurtosis package: %w", err)
	}

	info, err := ktd.GetEnvironmentInfo(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("error getting environment info: %w", err)
	}

	// Upload the environment info to the enclave.
	fs, err := d.newEnclaveFS(ctx, d.enclave)
	if err != nil {
		return nil, fmt.Errorf("error getting enclave fs: %w", err)
	}
	devnetFS := ktfs.NewDevnetFS(fs)
	if err := devnetFS.UploadDevnetDescriptor(ctx, info.DevnetEnvironment); err != nil {
		return nil, fmt.Errorf("error uploading devnet descriptor: %w", err)
	}

	return info, nil
}

func (d *Deployer) renderTemplate(buildDir string, urlBuilder func(path ...string) string) (*bytes.Buffer, error) {
	t := &Templater{
		baseDir:        d.baseDir,
		dryRun:         d.dryRun,
		enclave:        d.enclave,
		templateFile:   d.templateFile,
		dataFile:       d.dataFile,
		enclaveManager: d.enclaveManager,
		buildDir:       buildDir,
		urlBuilder:     urlBuilder,
	}

	return t.Render()
}

func (d *Deployer) Deploy(ctx context.Context, r io.Reader) (*kurtosis.KurtosisEnvironment, error) {

	// Clean up the enclave before deploying
	if d.autofixMode == autofixTypes.AutofixModeNuke {
		if d.enclaveManager != nil {
			// Remove all the enclaves and destroy all the docker resources related to kurtosis
			err := d.enclaveManager.Nuke(ctx)
			if err != nil {
				return nil, fmt.Errorf("error nuking enclave: %w", err)
			}
		}
	} else if d.autofixMode == autofixTypes.AutofixModeNormal {
		if d.enclaveManager != nil {
			if err := d.enclaveManager.Autofix(ctx, d.enclave); err != nil {
				return nil, fmt.Errorf("error autofixing enclave: %w", err)
			}
		}
	}

	// Pre-create the enclave if it doesn't exist
	if d.enclaveManager != nil {
		_, err := d.enclaveManager.GetEnclave(ctx, d.enclave)
		if err != nil {
			return nil, fmt.Errorf("error getting enclave: %w", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", d.enclave)
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srv := &FileServer{
		baseDir:  d.baseDir,
		dryRun:   d.dryRun,
		enclave:  d.enclave,
		deployer: d.ktDeployer,
	}

	ch := srv.getState(ctx)

	buf, err := d.renderTemplate(tmpDir, srv.URL)
	if err != nil {
		return nil, fmt.Errorf("error rendering template: %w", err)
	}

	if err := srv.Deploy(ctx, tmpDir, ch); err != nil {
		return nil, fmt.Errorf("error deploying fileserver: %w", err)
	}

	return d.deployEnvironment(ctx, buf)
}
