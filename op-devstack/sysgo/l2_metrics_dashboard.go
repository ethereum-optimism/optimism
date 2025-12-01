package sysgo

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"gopkg.in/yaml.v3"
)

const dockerExecutablePathEnvVar = "SYSGO_DOCKER_EXEC_PATH"
const grafanaProvisioningDirEnvVar = "SYSGO_GRAFANA_PROVISIONING_DIR"
const grafanaDataDirEnvVar = "SYSGO_GRAFANA_DATA_DIR"
const grafanaDockerImageTagEnvVar = "SYSGO_GRAFANA_DOCKER_IMAGE_TAG"
const prometheusDockerImageTagEnvVar = "SYSGO_PROMETHEUS_DOCKER_IMAGE_TAG"

const dockerToLocalHost = "host.docker.internal"

const prometheusHost = "0.0.0.0"
const prometheusServerPort = "9999"
const prometheusDockerPort = "9090"

const grafanaHost = "0.0.0.0"
const grafanaServerPort = "3000"
const grafanaDockerPort = "3000"

type L2MetricsRegistrar interface {
	// RegisterL2MetricsTargets is called by components when they are started (or earlier) to register
	// their metrics endpoints so that a prometheus instance may be spun up to scrape metrics.
	RegisterL2MetricsTargets(serviceName stack.IDWithChain, endpoints ...PrometheusMetricsTarget)
}

type PrometheusMetricsTarget string

func NewPrometheusMetricsTarget(host string, port string, isRunningInDocker bool) PrometheusMetricsTarget {
	if !isRunningInDocker {
		host = dockerToLocalHost
	}
	return PrometheusMetricsTarget(fmt.Sprintf("%s:%s", host, port))
}

type L2MetricsDashboard struct {
	p devtest.P

	grafanaExecPath   string
	grafanaArgs       []string
	grafanaEnv        []string
	grafanaSubprocess *SubProcess

	prometheusExecPath   string
	prometheusArgs       []string
	prometheusEnv        []string
	prometheusSubprocess *SubProcess

	prometheusEndpoint string
}

func (g *L2MetricsDashboard) Start() {
	g.startPrometheus()
	g.startGrafana()
}

func (g *L2MetricsDashboard) Stop() {
	var stopWaitGroup sync.WaitGroup

	stopWaitGroup.Add(1)
	go func() {
		defer stopWaitGroup.Done()
		err := g.grafanaSubprocess.Stop(true)
		g.p.Require().NoError(err, "Grafana must stop")
		g.grafanaSubprocess = nil
	}()

	stopWaitGroup.Add(1)
	go func() {
		defer stopWaitGroup.Done()
		err := g.prometheusSubprocess.Stop(true)
		g.p.Require().NoError(err, "Prometheus must stop")
		g.prometheusSubprocess = nil
	}()

	stopWaitGroup.Wait()
}

func (g *L2MetricsDashboard) startPrometheus() {
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	logOut := logpipe.ToLogger(g.p.Logger().New("component", "prometheus", "src", "stdout"))
	logErr := logpipe.ToLogger(g.p.Logger().New("component", "prometheus", "src", "stderr"))

	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})

	g.prometheusSubprocess = NewSubProcess(g.p, stdOutLogs, stdErrLogs)

	err := g.prometheusSubprocess.Start(g.prometheusExecPath, g.prometheusArgs, g.prometheusEnv)
	g.p.Require().NoError(err, "prometheus must start")

	g.p.Logger().Info("Prometheus started", "endpoint", g.prometheusEndpoint)
}

func (g *L2MetricsDashboard) startGrafana() {
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	logOut := logpipe.ToLogger(g.p.Logger().New("component", "grafana", "src", "stdout"))
	logErr := logpipe.ToLogger(g.p.Logger().New("component", "grafana", "src", "stderr"))

	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})

	g.grafanaSubprocess = NewSubProcess(g.p, stdOutLogs, stdErrLogs)

	err := g.grafanaSubprocess.Start(g.grafanaExecPath, g.grafanaArgs, g.grafanaEnv)
	g.p.Require().NoError(err, "Grafana must start")

	g.p.Logger().Info("Grafana started")
}

func WithL2MetricsDashboard() stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		// don't start prometheus or grafana if metrics are disabled or there is nothing exporting metrics.
		if !areMetricsEnabled() || orch.l2MetricsEndpoints.Len() == 0 {
			return
		}

		p := orch.P()

		prometheusImageTag := getEnvVarOrDefault(prometheusDockerImageTagEnvVar, "v3.7.2")
		prometheusEndpoint := fmt.Sprintf("http://%s:%s", prometheusHost, prometheusServerPort)
		promConfig := getPrometheusConfigFilePath(p, &orch.l2MetricsEndpoints)
		// these are args to run via docker; see dashboard definition below
		prometheusArgs := []string{
			"run",
			"-p", fmt.Sprintf("%s:%s", prometheusServerPort, prometheusDockerPort),
			"-v", fmt.Sprintf("%s:/etc/prometheus/prometheus.yml:ro", promConfig),
			fmt.Sprintf("prom/prometheus:%s", prometheusImageTag),
			"--config.file=/etc/prometheus/prometheus.yml",
		}

		grafanaImageTag := getEnvVarOrDefault(grafanaDockerImageTagEnvVar, "12.2")
		grafanaEndpoint := fmt.Sprintf("http://%s:%s", grafanaHost, grafanaServerPort)
		grafanaProvDir := getGrafanaProvisioningDirPath(p)
		grafanaDataDir := getGrafanaDataDir(p)
		// these are args to run via docker; see dashboard definition below
		grafanaArgs := []string{
			"run",
			"-p", fmt.Sprintf("%s:%s", grafanaServerPort, grafanaDockerPort),
			"-v", fmt.Sprintf("%s:/etc/grafana/provisioning:ro", grafanaProvDir),
			"-v", fmt.Sprintf("%s:/var/lib/grafana", grafanaDataDir),
			fmt.Sprintf("grafana/grafana:%s", grafanaImageTag),
		}
		grafanaEnv := []string{
			propagateEnvVarOrDefault("GF_SECURITY_ADMIN_USER", "admin"),
			propagateEnvVarOrDefault("GF_SECURITY_ADMIN_PASSWORD", "admin"),
			propagateEnvVarOrDefault("GF_USERS_ALLOW_SIGN_UP", "false"),
			propagateEnvVarOrDefault("GF_INSTALL_PLUGINS", "grafana-piechart-panel"),
		}
		dashboard := &L2MetricsDashboard{
			p: p,

			prometheusExecPath: getEnvVarOrDefault(dockerExecutablePathEnvVar, "docker"),
			prometheusArgs:     prometheusArgs,
			prometheusEnv:      os.Environ(),
			prometheusEndpoint: prometheusEndpoint,

			grafanaExecPath: getEnvVarOrDefault(dockerExecutablePathEnvVar, "docker"),
			grafanaArgs:     grafanaArgs,
			grafanaEnv:      append(grafanaEnv, os.Environ()...),
		}

		p.Logger().Info("Starting metrics dashboard", "dashboard", dashboard)

		dashboard.Start()
		p.Cleanup(dashboard.Stop)
		p.Logger().Info("Metrics dashboard is up", "url", grafanaEndpoint)
	})
}

// TODO: If our needs get more complex, use https://pkg.go.dev/github.com/prometheus/prometheus/config instead.
type prometheusConfig struct {
	Global        prometheusGlobalConfig        `yaml:"global"`
	ScrapeConfigs []prometheusScrapeConfigEntry `yaml:"scrape_configs"`
}

type prometheusGlobalConfig struct {
	ScrapeInterval     string `yaml:"scrape_interval"`
	EvaluationInterval string `yaml:"evaluation_interval"`
}

type prometheusScrapeConfigEntry struct {
	Name          string                   `yaml:"job_name"`
	Scheme        string                   `yaml:"scheme"`
	StaticConfigs []prometheusStaticConfig `yaml:"static_configs"`
}

type prometheusStaticConfig struct {
	Targets []string `yaml:"targets"`
}

// Returns the path to the dynamically-generated prometheus.yml file for metrics scraping.
func getPrometheusConfigFilePath(p devtest.P, metricsEndpoints *locks.RWMap[string, []PrometheusMetricsTarget]) string {

	var scrapeConfigs []prometheusScrapeConfigEntry

	metricsEndpoints.Range(func(name string, endpoints []PrometheusMetricsTarget) bool {
		var targets []string
		for _, endpoint := range endpoints {
			targets = append(targets, string(endpoint))
		}
		scrapeConfigs = append(scrapeConfigs, prometheusScrapeConfigEntry{
			Name:          name,
			Scheme:        "http",
			StaticConfigs: []prometheusStaticConfig{{Targets: targets}},
		})
		return true
	})

	yamlConfig := prometheusConfig{
		Global: prometheusGlobalConfig{
			ScrapeInterval:     "5s",
			EvaluationInterval: "5s",
		},
		ScrapeConfigs: scrapeConfigs,
	}

	b, err := yaml.Marshal(&yamlConfig)
	p.Require().NoError(err, "getPrometheusConfigFilePath: error creating yaml from scrape configs", "scrapeConfigs", scrapeConfigs)

	p.Logger().Info(`getPrometheusConfigFilePath: generated prometheus.yml`, "prometheus.yaml", string(b))

	filePath := filepath.Join(p.TempDir(), "prometheus.yml")
	file, err := os.Create(filePath)
	p.Require().NoError(err, "getPrometheusConfigFilePath:error creating prometheus file", "filePath", filePath)
	defer func() {
		p.Require().NoError(file.Close())
	}()

	_, err = file.Write(b)
	p.Require().NoError(err, "getPrometheusConfigFilePath:error writing string to prom file", "filePath", filePath, "contents", string(b))

	return filePath
}

// getGrafanaProvisioningDirPath returns the path to the grafana provisioning dir for metrics.
// If the provisioning dir env var is set, this function will use that path. If not, a temp dir
// will be created and removed when this process terminates.
// Note: from the returned directory, the generated prometheus.yml will be at:
//
//	returned_dir_path/provisioning/datasources/prometheus.yml
func getGrafanaProvisioningDirPath(p devtest.P) string {
	// If the caller provides a Grafana provisioning directory, use that, otherwise use a temp dir
	baseDir := os.Getenv(grafanaProvisioningDirEnvVar)
	if baseDir == "" {
		baseDir = filepath.Join(p.TempDir(), "grafana")
	}

	dirPath := filepath.Join(baseDir, "provisioning/datasources")
	err := os.MkdirAll(dirPath, 0777)
	p.Require().NoError(err, "getGrafanaProvisioningDirPath: error writing dir path", "dirPath", dirPath)

	p.Logger().Info("Created grafana/provisioning/datasources dir", "dirPath", dirPath)

	filePath := filepath.Join(dirPath, "prometheus.yml")
	file, err := os.Create(filePath)
	p.Require().NoError(err, "getGrafanaProvisioningDirPath: error creating prometheus file", "filePath", filePath)
	defer func() {
		p.Require().NoError(file.Close())
	}()

	contents := fmt.Sprintf(
		`
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://%s:%s
    isDefault: true
`, dockerToLocalHost, prometheusServerPort)

	if _, err = file.WriteString(contents); err != nil {
		p.Require().NoError(err, "getGrafanaProvisioningDirPath: error writing prom file", "filePath", filePath, "contents", contents)
	}

	p.Logger().Info("getGrafanaProvisioningDirPath: wrote prom config to file", "filePath", filePath, "contents", contents)

	return baseDir
}

// getGrafanaDataDir returns the path to the grafana provisioning dir for metrics.
// If the data dir env var is set, this function will use that path. If not, a temp dir
// will be created and removed when this process terminates.
func getGrafanaDataDir(p devtest.P) string {
	// If the caller provides a Grafana data directory, use that, otherwise use a temp dir
	baseDir := os.Getenv(grafanaDataDirEnvVar)
	if baseDir == "" {
		baseDir = filepath.Join(p.TempDir(), "grafana-data")
	}

	if _, err := os.Stat(baseDir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(baseDir, 0777); err != nil {
			p.Require().NoError(err, "getGrafanaDataDir: error creating grafana data directory", "baseDir", baseDir)
		}
	} else {
		p.Require().NoError(err, "getGrafanaDataDir: checking if grafana data directory exists", "baseDir", baseDir)
	}

	return baseDir
}
