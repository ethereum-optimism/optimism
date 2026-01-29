package build

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/semaphore"
)

// cmdRunner abstracts command execution for testing
type cmdRunner interface {
	// CombinedOutput is kept for potential future use or simpler scenarios
	CombinedOutput() ([]byte, error)
	// Run starts the command and waits for it to complete.
	// It's often preferred when you want to manage stdout/stderr separately.
	Run() error
	// SetOutput sets the writers for stdout and stderr.
	SetOutput(stdout, stderr *bytes.Buffer)
	Dir() string
	SetDir(dir string)
}

// defaultCmdRunner is the default implementation that uses exec.Command
type defaultCmdRunner struct {
	*exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (r *defaultCmdRunner) CombinedOutput() ([]byte, error) {
	if r.stdout == nil || r.stderr == nil {
		var combined bytes.Buffer
		r.Cmd.Stdout = &combined
		r.Cmd.Stderr = &combined
		err := r.Cmd.Run()
		return combined.Bytes(), err
	}
	err := r.Run()
	combined := append(r.stdout.Bytes(), r.stderr.Bytes()...)
	return combined, err
}

func (r *defaultCmdRunner) SetOutput(stdout, stderr *bytes.Buffer) {
	r.stdout = stdout
	r.stderr = stderr
	r.Cmd.Stdout = stdout
	r.Cmd.Stderr = stderr
}

func (r *defaultCmdRunner) Run() error {
	return r.Cmd.Run()
}

func (r *defaultCmdRunner) Dir() string {
	return r.Cmd.Dir
}

func (r *defaultCmdRunner) SetDir(dir string) {
	r.Cmd.Dir = dir
}

// cmdFactory creates commands
type cmdFactory func(name string, arg ...string) cmdRunner

// defaultCmdFactory is the default implementation that uses exec.Command
func defaultCmdFactory(name string, arg ...string) cmdRunner {
	return &defaultCmdRunner{Cmd: exec.Command(name, arg...)}
}

// dockerClient interface defines the Docker client methods we use
type dockerClient interface {
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	ImageTag(ctx context.Context, source, target string) error
}

// dockerProvider abstracts the creation of Docker clients
type dockerProvider interface {
	newClient() (dockerClient, error)
}

// defaultDockerProvider is the default implementation of dockerProvider
type defaultDockerProvider struct{}

func (p *defaultDockerProvider) newClient() (dockerClient, error) {
	opts := []client.Opt{client.FromEnv}

	// Check if default docker socket exists
	hostURL, err := url.Parse(client.DefaultDockerHost)
	if err != nil {
		return nil, fmt.Errorf("failed to parse default docker host: %w", err)
	}

	// For unix sockets, check if the socket file exists
	unixOS := runtime.GOOS == "linux" || runtime.GOOS == "darwin"
	if hostURL.Scheme == "unix" && unixOS {
		if _, err := os.Stat(hostURL.Path); os.IsNotExist(err) {
			// Default socket doesn't exist, try to find an alternate location. Docker Desktop uses a socket in the home directory.
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			// Try to use the non-privileged socket if available
			homeSocketPath := fmt.Sprintf("%s/.docker/run/docker.sock", homeDir)
			if runtime.GOOS == "linux" {
				homeSocketPath = fmt.Sprintf("%s/.docker/desktop/docker.sock", homeDir)
			}

			// If that socket exists, make it the default. Otherwise, leave it alone, and hope some environment variable has been set.
			if _, err := os.Stat(homeSocketPath); err == nil {
				socketURL := &url.URL{
					Scheme: "unix",
					Path:   homeSocketPath,
				}
				// prepend the host, so that it can still be overridden by the environment.
				opts = append([]client.Opt{client.WithHost(socketURL.String())}, opts...)
			}
		}
	}

	return client.NewClientWithOpts(opts...)
}

// DockerBuilder handles building docker images using just commands
type DockerBuilder struct {
	// Base directory where the build commands should be executed
	baseDir string
	// Template for the build command
	cmdTemplate *template.Template
	// Dry run mode
	dryRun bool
	// Docker provider for creating clients
	dockerProvider dockerProvider
	// Command factory for testing
	cmdFactory cmdFactory
	// Concurrency limiting semaphore
	sem *semaphore.Weighted
	// Mutex to protect shared state (buildStates)
	mu sync.Mutex
	// Tracks the state of builds (ongoing or completed)
	buildStates map[string]*buildState
}

// buildState stores the result and status of a build
type buildState struct {
	result string
	err    error
	done   chan struct{}
	once   sync.Once
}

const cmdTemplateStr = "just {{.ProjectName}}-image {{.ImageTag}}"

var defaultCmdTemplate *template.Template

func init() {
	defaultCmdTemplate = template.Must(template.New("docker_build_cmd").Parse(cmdTemplateStr))
}

type DockerBuilderOptions func(*DockerBuilder)

func WithDockerCmdTemplate(cmdTemplate *template.Template) DockerBuilderOptions {
	return func(b *DockerBuilder) {
		b.cmdTemplate = cmdTemplate
	}
}

func WithDockerBaseDir(baseDir string) DockerBuilderOptions {
	return func(b *DockerBuilder) {
		b.baseDir = baseDir
	}
}

func WithDockerDryRun(dryRun bool) DockerBuilderOptions {
	return func(b *DockerBuilder) {
		b.dryRun = dryRun
	}
}

// WithDockerConcurrency sets the maximum number of concurrent builds.
func WithDockerConcurrency(limit int) DockerBuilderOptions {
	if limit <= 0 {
		limit = 1
	}
	if limit >= 32 {
		limit = 32
	}
	return func(b *DockerBuilder) {
		b.sem = semaphore.NewWeighted(int64(limit))
	}
}

// NewDockerBuilder creates a new DockerBuilder instance
func NewDockerBuilder(opts ...DockerBuilderOptions) *DockerBuilder {
	b := &DockerBuilder{
		baseDir:        ".",
		cmdTemplate:    defaultCmdTemplate,
		dryRun:         false,
		dockerProvider: &defaultDockerProvider{},
		cmdFactory:     defaultCmdFactory,
		sem:            semaphore.NewWeighted(1),
		buildStates:    make(map[string]*buildState),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// templateData holds the data for the command template
type templateData struct {
	ImageTag    string
	ProjectName string
}

// Build ensures the docker image for the given project is built, respecting concurrency limits.
// It blocks until the specific requested build is complete. Other builds may run concurrently.
func (b *DockerBuilder) Build(ctx context.Context, projectName, imageTag string) (string, error) {
	b.mu.Lock()
	state, exists := b.buildStates[projectName]
	if !exists {
		state = &buildState{
			done: make(chan struct{}),
		}
		b.buildStates[projectName] = state
	}
	b.mu.Unlock()

	if !exists {
		state.once.Do(func() {
			err := b.executeBuild(ctx, projectName, imageTag, state)
			if err != nil {
				state.err = err
				state.result = ""
			}
			close(state.done)
		})
	} else {
		<-state.done
	}

	return state.result, state.err
}

func (b *DockerBuilder) executeBuild(ctx context.Context, projectName, initialImageTag string, state *buildState) error {
	ctx, span := otel.Tracer("docker-builder").Start(ctx, fmt.Sprintf("build %s", projectName))
	defer span.End()

	log.Printf("Build started for project: %s (tag: %s)", projectName, initialImageTag)

	if b.dryRun {
		log.Printf("Dry run: Skipping build for project %s", projectName)
		state.result = initialImageTag
		return nil
	}

	if err := b.sem.Acquire(ctx, 1); err != nil {
		log.Printf("Failed to acquire build semaphore for %s: %v", projectName, err)
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer b.sem.Release(1)

	data := templateData{
		ImageTag:    initialImageTag,
		ProjectName: projectName,
	}

	var cmdBuf bytes.Buffer
	if err := b.cmdTemplate.Execute(&cmdBuf, data); err != nil {
		log.Printf("Build failed for %s: Failed to execute command template: %v", projectName, err)
		return fmt.Errorf("failed to execute command template: %w", err)
	}
	cmdStr := cmdBuf.String()

	cmd := b.cmdFactory("sh", "-c", cmdStr)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetOutput(&stdoutBuf, &stderrBuf)

	startTime := time.Now()
	log.Printf("Executing build command for %s: %s", projectName, cmdStr)
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Build failed for %s after %s: %v", projectName, duration, err)
		log.Printf("--- Start Output (stdout) for failed %s ---", projectName)
		log.Print(stdoutBuf.String())
		log.Printf("--- End Output (stdout) for failed %s ---", projectName)
		log.Printf("--- Start Output (stderr) for failed %s ---", projectName)
		log.Print(stderrBuf.String())
		log.Printf("--- End Output (stderr) for failed %s ---", projectName)
		return fmt.Errorf("build command failed: %w", err)
	}

	dockerClient, err := b.dockerProvider.newClient()
	if err != nil {
		log.Printf("Build command succeeded for %s, but Docker client creation failed: %v", projectName, err)
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	inspect, _, err := dockerClient.ImageInspectWithRaw(ctx, initialImageTag)
	if err != nil {
		log.Printf("Build command succeeded for %s in %s, but failed to inspect image '%s': %v", projectName, duration, initialImageTag, err)
		log.Printf("Stdout: %s", stdoutBuf.String())
		log.Printf("Stderr: %s", stderrBuf.String())
		return fmt.Errorf("build command succeeded but failed to inspect image %s: %w", initialImageTag, err)
	}

	shortID := TruncateID(inspect.ID)

	finalTag := fmt.Sprintf("%s:%s", projectName, shortID)
	err = dockerClient.ImageTag(ctx, initialImageTag, finalTag)
	if err != nil {
		log.Printf("Build succeeded for %s in %s, inspecting image '%s' OK, but failed to tag as '%s': %v", projectName, duration, initialImageTag, finalTag, err)
		return fmt.Errorf("failed to tag image %s as %s: %w", initialImageTag, finalTag, err)
	}

	state.result = finalTag
	log.Printf("Build successful for project: %s. Tagged as: %s (Duration: %s)", projectName, finalTag, duration)
	return nil
}

func TruncateID(id string) string {
	shortID := strings.TrimPrefix(id, "sha256:")
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	return shortID
}
