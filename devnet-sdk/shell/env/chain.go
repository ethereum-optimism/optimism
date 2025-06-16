package env

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
)

const (
	EnvURLVar              = "DEVNET_ENV_URL"
	ChainNameVar           = "DEVNET_CHAIN_NAME"
	NodeIndexVar           = "DEVNET_NODE_INDEX"
	ExpectPreconditionsMet = "DEVNET_EXPECT_PRECONDITIONS_MET"
	EnvCtrlVar             = "DEVNET_ENV_CTRL"
)

type ChainConfig struct {
	chain     *descriptors.Chain
	devnetURL string
	name      string
}

type ChainEnv struct {
	motd    string
	envVars map[string]string
}

func (c *ChainConfig) getRpcUrl(nodeIndex int) func() (string, error) {
	return func() (string, error) {
		if len(c.chain.Nodes) == 0 {
			return "", fmt.Errorf("chain '%s' has no nodes", c.chain.Name)
		}

		if nodeIndex >= len(c.chain.Nodes) {
			return "", fmt.Errorf("node index %d is out of bounds for chain '%s'", nodeIndex, c.chain.Name)
		}

		// Get RPC endpoint from the first node's execution layer service
		elService, ok := c.chain.Nodes[nodeIndex].Services["el"]
		if !ok {
			return "", fmt.Errorf("no execution layer service found for chain '%s'", c.chain.Name)
		}

		rpcEndpoint, ok := elService.Endpoints["rpc"]
		if !ok {
			return "", fmt.Errorf("no RPC endpoint found for chain '%s'", c.chain.Name)
		}

		scheme := rpcEndpoint.Scheme
		if scheme == "" {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s:%d", scheme, rpcEndpoint.Host, rpcEndpoint.Port), nil
	}
}

func (c *ChainConfig) getJwtSecret() (string, error) {
	jwt := c.chain.JWT
	if len(jwt) >= 2 && jwt[:2] == "0x" {
		jwt = jwt[2:]
	}

	return jwt, nil
}

func (c *ChainConfig) motd() string {
	tmpl := `You're in a {{.Name}} chain subshell.

	Some addresses of interest:
	{{ range $key, $value := .Addresses -}}
		{{ printf "%-35s" $key }} = {{ $value }}
	{{ end -}}
	`

	t := template.Must(template.New("motd").Parse(tmpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, c.chain); err != nil {
		panic(err)
	}

	return buf.String()
}

type ChainConfigOption func(*ChainConfig, *chainConfigOpts) error

type chainConfigOpts struct {
	extraEnvVars map[string]string
}

func WithCastIntegration(cast bool, nodeIndex int) ChainConfigOption {
	return func(c *ChainConfig, o *chainConfigOpts) error {
		mapping := map[string]func() (string, error){
			"ETH_RPC_URL":        c.getRpcUrl(nodeIndex),
			"ETH_RPC_JWT_SECRET": c.getJwtSecret,
		}

		for key, fn := range mapping {
			value := ""
			var err error
			if cast {
				if value, err = fn(); err != nil {
					return err
				}
			}
			o.extraEnvVars[key] = value
		}
		return nil
	}
}

func WithExpectedPreconditions(pre bool) ChainConfigOption {
	return func(c *ChainConfig, o *chainConfigOpts) error {
		if pre {
			o.extraEnvVars[ExpectPreconditionsMet] = "true"
		} else {
			o.extraEnvVars[ExpectPreconditionsMet] = ""
		}
		return nil
	}
}

func (c *ChainConfig) GetEnv(opts ...ChainConfigOption) (*ChainEnv, error) {
	motd := c.motd()
	o := &chainConfigOpts{
		extraEnvVars: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(c, o); err != nil {
			return nil, err
		}
	}

	// To allow commands within the shell to know which devnet and chain they are in
	absPath := c.devnetURL
	if u, err := url.Parse(c.devnetURL); err == nil {
		if u.Scheme == "" || u.Scheme == "file" {
			// make sure the path is absolute
			if abs, err := filepath.Abs(u.Path); err == nil {
				absPath = abs
			}
		}
	}
	o.extraEnvVars[EnvURLVar] = absPath
	o.extraEnvVars[ChainNameVar] = c.name

	return &ChainEnv{
		motd:    motd,
		envVars: o.extraEnvVars,
	}, nil
}

func (e *ChainEnv) ApplyToEnv(env []string) []string {
	// first identify which env vars to clear
	clearEnv := make(map[string]interface{})
	for key := range e.envVars {
		clearEnv[key] = nil
	}

	// then actually remove these from the env
	cleanEnv := make([]string, 0)
	for _, s := range env {
		key := strings.SplitN(s, "=", 2)[0]
		if _, ok := clearEnv[key]; !ok {
			cleanEnv = append(cleanEnv, s)
		}
	}

	// then add the remaining env vars
	for key, value := range e.envVars {
		if value == "" {
			continue
		}
		cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", key, value))
	}
	return cleanEnv
}

func (e *ChainEnv) GetMotd() string {
	return e.motd
}
