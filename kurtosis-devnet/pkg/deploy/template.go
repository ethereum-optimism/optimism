package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/build"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/tmpl"
)

var (
	dockerBuildConcurrency = 8
)

type Templater struct {
	enclave      string
	dryRun       bool
	baseDir      string
	templateFile string
	dataFile     string
	buildDir     string
	urlBuilder   func(path ...string) string

	// Common state across template functions
	buildJobsMux sync.Mutex
	buildJobs    map[string]*dockerBuildJob
}

// dockerBuildJob helps collect and group build jobs
type dockerBuildJob struct {
	projectName string
	imageTag    string
	result      string
	err         error
	done        chan struct{}
}

func (f *Templater) localDockerImageOption() tmpl.TemplateContextOptions {
	// Initialize the build jobs map if it's nil
	if f.buildJobs == nil {
		f.buildJobs = make(map[string]*dockerBuildJob)
	}

	imageTag := func(projectName string) string {
		return fmt.Sprintf("%s:%s", projectName, f.enclave)
	}

	// Function that gets called during template rendering
	return tmpl.WithFunction("localDockerImage", func(projectName string) (string, error) {
		tag := imageTag(projectName)

		// First, check if we already have this build job
		f.buildJobsMux.Lock()
		job, exists := f.buildJobs[projectName]
		if !exists {
			// If not, create a new job but don't start it yet
			job = &dockerBuildJob{
				projectName: projectName,
				imageTag:    tag,
				done:        make(chan struct{}),
			}
			f.buildJobs[projectName] = job
		}
		f.buildJobsMux.Unlock()

		// If the job is already done, return its result
		select {
		case <-job.done:
			return job.result, job.err
		default:
			// Just collect the build request for now and return a placeholder
			// The actual build will happen in Render() before final template evaluation
			return fmt.Sprintf("__PLACEHOLDER_DOCKER_IMAGE_%s__", projectName), nil
		}
	})
}

func (f *Templater) localContractArtifactsOption() tmpl.TemplateContextOptions {
	contractsBundle := fmt.Sprintf("contracts-bundle-%s.tar.gz", f.enclave)
	contractsBundlePath := func(_ string) string {
		return filepath.Join(f.buildDir, contractsBundle)
	}
	contractsURL := f.urlBuilder(contractsBundle)

	contractBuilder := build.NewContractBuilder(
		build.WithContractBaseDir(f.baseDir),
		build.WithContractDryRun(f.dryRun),
	)

	return tmpl.WithFunction("localContractArtifacts", func(layer string) (string, error) {
		bundlePath := contractsBundlePath(layer)
		if err := contractBuilder.Build(layer, bundlePath); err != nil {
			return "", err
		}

		log.Printf("%s: contract artifacts available at: %s\n", layer, contractsURL)
		return contractsURL, nil
	})
}

func (f *Templater) localPrestateOption() tmpl.TemplateContextOptions {
	holder := &localPrestateHolder{
		baseDir:  f.baseDir,
		buildDir: f.buildDir,
		dryRun:   f.dryRun,
		builder: build.NewPrestateBuilder(
			build.WithPrestateBaseDir(f.baseDir),
			build.WithPrestateDryRun(f.dryRun),
		),
		urlBuilder: f.urlBuilder,
	}

	return tmpl.WithFunction("localPrestate", func() (*PrestateInfo, error) {
		return holder.GetPrestateInfo()
	})
}

func (f *Templater) Render() (*bytes.Buffer, error) {
	// Initialize the build jobs map if it's nil
	if f.buildJobs == nil {
		f.buildJobs = make(map[string]*dockerBuildJob)
	}

	opts := []tmpl.TemplateContextOptions{
		f.localDockerImageOption(),
		f.localContractArtifactsOption(),
		f.localPrestateOption(),
		tmpl.WithBaseDir(f.baseDir),
	}

	// Read and parse the data file if provided
	if f.dataFile != "" {
		data, err := os.ReadFile(f.dataFile)
		if err != nil {
			return nil, fmt.Errorf("error reading data file: %w", err)
		}

		var templateData map[string]interface{}
		if err := json.Unmarshal(data, &templateData); err != nil {
			return nil, fmt.Errorf("error parsing JSON data: %w", err)
		}

		opts = append(opts, tmpl.WithData(templateData))
	}

	// Open template file
	tmplFile, err := os.Open(f.templateFile)
	if err != nil {
		return nil, fmt.Errorf("error opening template file: %w", err)
	}
	defer tmplFile.Close()

	// Create template context
	tmplCtx := tmpl.NewTemplateContext(opts...)

	// First pass: Collect all build jobs without executing them
	prelimBuf := bytes.NewBuffer(nil)
	if err := tmplCtx.InstantiateTemplate(tmplFile, prelimBuf); err != nil {
		return nil, fmt.Errorf("error in first-pass template processing: %w", err)
	}

	// Find all docker build jobs and execute them concurrently
	var dockerJobs []*dockerBuildJob
	f.buildJobsMux.Lock()
	for _, job := range f.buildJobs {
		dockerJobs = append(dockerJobs, job)
	}
	f.buildJobsMux.Unlock()

	if len(dockerJobs) > 0 {
		// Create a single Docker builder for all builds using the factory
		dockerBuilder := build.NewDockerBuilder(
			build.WithDockerBaseDir(f.baseDir),
			build.WithDockerDryRun(f.dryRun),
			build.WithDockerConcurrency(dockerBuildConcurrency), // Set concurrency
		)

		// Start all the builds
		var wg sync.WaitGroup
		wg.Add(len(dockerJobs))
		for _, job := range dockerJobs {
			go func(j *dockerBuildJob) {
				defer wg.Done()
				log.Printf("Starting build for %s (tag: %s)", j.projectName, j.imageTag)
				j.result, j.err = dockerBuilder.Build(j.projectName, j.imageTag)
				close(j.done) // Mark this job as done
			}(job)
		}
		wg.Wait() // Wait for all builds to complete

		// Check for any build errors
		for _, job := range dockerJobs {
			if job.err != nil {
				return nil, fmt.Errorf("error building docker image for %s: %w", job.projectName, job.err)
			}
		}

		// Now reopen the template file for the second pass
		tmplFile.Close()
		tmplFile, err = os.Open(f.templateFile)
		if err != nil {
			return nil, fmt.Errorf("error reopening template file: %w", err)
		}
		defer tmplFile.Close()
	}

	// Second pass: Render with actual build results
	buf := bytes.NewBuffer(nil)
	if err := tmplCtx.InstantiateTemplate(tmplFile, buf); err != nil {
		return nil, fmt.Errorf("error processing template: %w", err)
	}

	return buf, nil
}
