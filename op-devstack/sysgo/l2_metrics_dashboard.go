package sysgo

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"gopkg.in/yaml.v3"
)

const DockerExecutablePathEnvVar = "SYSGO_DOCKER_EXEC_PATH"
const GrafanaProvisioningDirEnvVar = "SYSGO_GRAFANA_PROVISIONING_DIR"
const GrafanaDataDirEnvVar = "SYSGO_GRAFANA_DATA_DIR"

const DockerToLocalHost = "host.docker.internal"

const PrometheusHost = "0.0.0.0"
const PrometheusServerPort = "9999"
const PrometheusDockerPort = "9090"

const GrafanaHost = "0.0.0.0"
const GrafanaServerPort = "3000"
const GrafanaDockerPort = "3000"

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
	err := g.grafanaSubprocess.Stop()
	g.p.Require().NoError(err, "Grafana must stop")

	err = g.prometheusSubprocess.Stop()
	g.p.Require().NoError(err, "Prometheus must stop")

	g.grafanaSubprocess = nil
	g.prometheusSubprocess = nil
}

func (g *L2MetricsDashboard) startPrometheus() {
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	logOut := logpipe.ToLogger(g.p.Logger().New("src", "stdout"))
	logErr := logpipe.ToLogger(g.p.Logger().New("src", "stderr"))

	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})

	g.prometheusSubprocess = NewSubProcess(g.p, stdOutLogs, stdErrLogs)

	if err := g.prometheusSubprocess.Start(g.prometheusExecPath, g.prometheusArgs, g.prometheusEnv); err != nil {
		g.p.Logger().Error(fmt.Sprintf("Error starting prometheus: %+v", err))
		g.p.Require().NoError(err, "Must start")
	}

	// Wait until prometheus is ready
	url := fmt.Sprintf("%s/-/ready", g.prometheusEndpoint)
	interval := 2 * time.Second

	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}

		g.p.Logger().Info(fmt.Sprintf("Waiting for prometheus to start at %s...", g.prometheusEndpoint))

		time.Sleep(interval)
	}

	g.p.Logger().Info(fmt.Sprintf("Prometheus started at %s", g.prometheusEndpoint))
}

func (g *L2MetricsDashboard) startGrafana() {
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	logOut := logpipe.ToLogger(g.p.Logger().New("src", "stdout"))
	logErr := logpipe.ToLogger(g.p.Logger().New("src", "stderr"))

	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})

	g.grafanaSubprocess = NewSubProcess(g.p, stdOutLogs, stdErrLogs)

	if err := g.grafanaSubprocess.Start(g.grafanaExecPath, g.grafanaArgs, g.grafanaEnv); err != nil {
		g.p.Logger().Error(fmt.Sprintf("Error starting grafana: %+v", err))
		g.p.Require().NoError(err, "Must start")
	}

	g.p.Logger().Info("Grafana started")
}

func WithL2MetricsDashboard() stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		// don't start prometheus or grafana if there is nothing exporting metrics.
		if orch.l2MetricsEndpoints.Len() == 0 {
			return
		}

		p := orch.P()

		prometheusEndpoint := fmt.Sprintf("http://%s:%s", PrometheusHost, PrometheusServerPort)
		promConfig := getPrometheusConfigFilePath(p, &orch.l2MetricsEndpoints)
		// these are args to run via docker; see dashboard definition below
		prometheusArgs := []string{
			"run",
			"-p", fmt.Sprintf("%s:%s", PrometheusServerPort, PrometheusDockerPort),
			"-v", fmt.Sprintf("%s:/etc/prometheus/prometheus.yml:ro", promConfig),
			"prom/prometheus",
			"--config.file=/etc/prometheus/prometheus.yml",
		}

		grafanaEndpoint := fmt.Sprintf("http://%s:%s", GrafanaHost, GrafanaServerPort)
		grafanaProvDir := getGrafanaProvisioningDirPath(p)
		grafanaDataDir := getGrafanaDataDir(p)
		// these are args to run via docker; see dashboard definition below
		grafanaArgs := []string{
			"run",
			"-p", fmt.Sprintf("%s:%s", GrafanaServerPort, GrafanaDockerPort),
			"-v", fmt.Sprintf("%s:/etc/grafana/provisioning:ro", grafanaProvDir),
			"-v", fmt.Sprintf("%s:/var/lib/grafana", grafanaDataDir),
			"grafana/grafana",
		}
		grafanaEnv := []string{
			PropagateEnvVarOrDefault("GF_SECURITY_ADMIN_USER", "admin"),
			PropagateEnvVarOrDefault("GF_SECURITY_ADMIN_PASSWORD", "admin"),
			PropagateEnvVarOrDefault("GF_USERS_ALLOW_SIGN_UP", "false"),
			PropagateEnvVarOrDefault("GF_INSTALL_PLUGINS", "grafana-piechart-panel"),
		}

		dashboard := &L2MetricsDashboard{
			p: p,

			prometheusExecPath: GetEnvVarOrDefault(DockerExecutablePathEnvVar, "/usr/local/bin/docker"),
			prometheusArgs:     prometheusArgs,
			prometheusEnv:      []string{},
			prometheusEndpoint: prometheusEndpoint,

			grafanaExecPath: GetEnvVarOrDefault(DockerExecutablePathEnvVar, "/usr/local/bin/docker"),
			grafanaArgs:     grafanaArgs,
			grafanaEnv:      grafanaEnv,
		}

		p.Logger().Info(fmt.Sprintf("Starting metrics dashboard: %+v", dashboard))

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

// endpointHostPortString resolves the host:port string, accounting for the fact that the requester
// will be in a docker container.
func endpointHostPortString(p PrometheusMetricsEndpoint) string {
	host := p.host
	if p.isLocal && !p.isRunningInDocker {
		host = DockerToLocalHost
	}
	return fmt.Sprintf("%s:%s", host, p.port)
}

// Returns the path to the dynamically-generated prometheus.yml file for metrics scraping.
func getPrometheusConfigFilePath(p devtest.P, metricsEndpoints *locks.RWMap[string, []PrometheusMetricsEndpoint]) string {

	var scrapeConfigs []prometheusScrapeConfigEntry

	metricsEndpoints.Range(func(name string, endpoints []PrometheusMetricsEndpoint) bool {
		var targets []string
		for _, endpoint := range endpoints {
			targets = append(targets, endpointHostPortString(endpoint))
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
	p.Require().NoError(err, fmt.Sprintf("getPrometheusConfigFilePath: error creating yaml from scrape configs %+v", scrapeConfigs))

	p.Logger().Info(fmt.Sprintf(`getPrometheusConfigFilePath: generated prometheus.yml: %s`, string(b)))

	filePath := filepath.Join(p.TempDir(), "prometheus.yml")
	file, err := os.Create(filePath)
	p.Require().NoError(err, fmt.Sprintf("getPrometheusConfigFilePath:error creating prometheus file at %s", filePath))
	defer file.Close()

	_, err = file.Write(b)
	p.Require().NoError(err, fmt.Sprintf("getPrometheusConfigFilePath:error writing string to prom file at %s, string: %s", filePath, string(b)))

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
	baseDir := os.Getenv(GrafanaProvisioningDirEnvVar)
	if baseDir == "" {
		baseDir = filepath.Join(p.TempDir(), "grafana")
	}

	dirPath := filepath.Join(baseDir, "provisioning/datasources")
	err := os.MkdirAll(dirPath, 0777)
	p.Require().NoError(err, fmt.Sprintf("getGrafanaProvisioningDirPath: error writing dir path %s", dirPath))

	p.Logger().Info(fmt.Sprintf("Created grafana/provisioning/datasources dir at %s", dirPath))

	filePath := filepath.Join(dirPath, "prometheus.yml")
	file, err := os.Create(filePath)
	p.Require().NoError(err, fmt.Sprintf("getGrafanaProvisioningDirPath: error creating prometheus file at %s", filePath))
	defer file.Close()

	contents := fmt.Sprintf(
		`
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://%s:%s
    isDefault: true
`, DockerToLocalHost, PrometheusServerPort)

	if _, err = file.WriteString(contents); err != nil {
		p.Require().NoError(err, fmt.Sprintf("getGrafanaProvisioningDirPath: error prom file at %s, string: %s", filePath, contents))
	}

	p.Logger().Info(fmt.Sprintf("getGrafanaProvisioningDirPath: wrote prom config to file %s; config: %s", filePath, contents))

	return baseDir
}

// getGrafanaDataDir returns the path to the grafana provisioning dir for metrics.
// If the data dir env var is set, this function will use that path. If not, a temp dir
// will be created and removed when this process terminates.
func getGrafanaDataDir(p devtest.P) string {
	// If the caller provides a Grafana data directory, use that, otherwise use a temp dir
	baseDir := os.Getenv(GrafanaDataDirEnvVar)
	if baseDir == "" {
		baseDir = filepath.Join(p.TempDir(), "grafana-data")
	}

	if _, err := os.Stat(baseDir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(baseDir, 0777); err != nil {
			p.Require().NoError(err, fmt.Sprintf("getGrafanaDataDir: creating grafana data directory at %s", baseDir))
		}
	} else {
		p.Require().NoError(err, fmt.Sprintf("getGrafanaDataDir: checking if grafana data directory exists at %s", baseDir))
	}

	return baseDir
}
