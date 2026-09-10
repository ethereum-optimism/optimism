package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPublicExplorerPort  = uint(4000)
	defaultPrivateExplorerPort = uint(4001)
	minimumOtterscanAPILevel   = uint64(8)
	explorerReadyTimeout       = 90 * time.Second
	explorerStopTimeout        = 30 * time.Second
)

//go:embed explorers/compose.yaml
var otterscanCompose []byte

type otterscanConfig struct {
	publicRPCPort       uint
	privateRPCPort      uint
	publicExplorerPort  uint
	privateExplorerPort uint
	publicChainID       string
	privateChainID      string
}

func (c otterscanConfig) publicURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.publicExplorerPort)
}

func (c otterscanConfig) privateURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.privateExplorerPort)
}

func (c otterscanConfig) validate() error {
	ports := map[string]uint{
		"l2a-rpc-port":          c.publicRPCPort,
		"l2b-rpc-port":          c.privateRPCPort,
		"public-explorer-port":  c.publicExplorerPort,
		"private-explorer-port": c.privateExplorerPort,
	}
	seen := make(map[uint]string, len(ports))
	for name, port := range ports {
		if port == 0 || port > 65535 {
			return fmt.Errorf("%s must be between 1 and 65535", name)
		}
		if other, ok := seen[port]; ok {
			return fmt.Errorf("%s and %s must use different ports (both are %d)", other, name, port)
		}
		seen[port] = name
	}
	if c.publicChainID == "" || c.privateChainID == "" {
		return fmt.Errorf("explorer chain IDs must not be empty")
	}
	return nil
}

type otterscanStack struct {
	dockerPath  string
	composePath string
	projectName string
	environment []string
	output      io.Writer
	config      otterscanConfig

	stopOnce sync.Once
	stopErr  error
}

func startOtterscanStack(ctx context.Context, output io.Writer, workingDir string, config otterscanConfig) (*otterscanStack, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	for name, port := range map[string]uint{
		"public explorer":  config.publicExplorerPort,
		"private explorer": config.privateExplorerPort,
	} {
		if err := ensureHostPortAvailable(port); err != nil {
			return nil, fmt.Errorf("%s port: %w", name, err)
		}
	}
	if err := waitForOtterscanRPCs(ctx, config); err != nil {
		return nil, err
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("find Docker for Otterscan explorer GUIs (or use --explorers=false): %w", err)
	}
	versionCtx, cancelVersion := context.WithTimeout(ctx, 10*time.Second)
	defer cancelVersion()
	versionOutput, err := exec.CommandContext(versionCtx, dockerPath, "compose", "version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Docker Compose is required for Otterscan explorer GUIs: %w (%s)", err, strings.TrimSpace(string(versionOutput)))
	}

	composePath := filepath.Join(workingDir, "compose.yaml")
	if err := os.WriteFile(composePath, otterscanCompose, 0o600); err != nil {
		return nil, fmt.Errorf("write embedded Otterscan compose file: %w", err)
	}
	projectName := strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(filepath.Base(workingDir)))
	stack := &otterscanStack{
		dockerPath:  dockerPath,
		composePath: composePath,
		projectName: "op-up-" + projectName,
		environment: explorerEnvironment(os.Environ(), config),
		output:      output,
		config:      config,
	}

	fmt.Fprintln(output, "\n--- start Otterscan explorer GUIs")
	if err := stack.pull(ctx); err != nil {
		return nil, err
	}
	if err := stack.runCompose(ctx, "up", "--detach", "--no-build", "--remove-orphans"); err != nil {
		_ = stack.stop()
		return nil, fmt.Errorf("start Otterscan explorer GUIs: %w", err)
	}
	if err := stack.waitReady(ctx); err != nil {
		stack.printDiagnostics()
		_ = stack.stop()
		return nil, err
	}
	return stack, nil
}

func (s *otterscanStack) pull(ctx context.Context) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := s.runCompose(ctx, "pull"); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < 3 {
			fmt.Fprintf(s.output, "    Otterscan image pull failed (attempt %d/3); retrying...\n", attempt)
			timer := time.NewTimer(time.Duration(attempt) * 2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return fmt.Errorf("pull Otterscan image after 3 attempts: %w", lastErr)
}

func (s *otterscanStack) runCompose(ctx context.Context, args ...string) error {
	baseArgs := []string{"compose", "--project-name", s.projectName, "--file", s.composePath}
	cmd := exec.CommandContext(ctx, s.dockerPath, append(baseArgs, args...)...)
	cmd.Env = s.environment
	cmd.Stdout = s.output
	cmd.Stderr = s.output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (s *otterscanStack) stop() error {
	s.stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), explorerStopTimeout)
		defer cancel()
		s.stopErr = s.runCompose(ctx, "down", "--remove-orphans", "--timeout", "5")
	})
	return s.stopErr
}

func (s *otterscanStack) waitReady(ctx context.Context) error {
	waitCtx, cancel := context.WithTimeout(ctx, explorerReadyTimeout)
	defer cancel()
	ready := make(chan error, 2)
	go func() { ready <- waitForHTTP(waitCtx, s.config.publicURL()) }()
	go func() { ready <- waitForHTTP(waitCtx, s.config.privateURL()) }()
	for range 2 {
		if err := <-ready; err != nil {
			return fmt.Errorf("wait for Otterscan explorer GUIs: %w", err)
		}
	}
	return nil
}

func (s *otterscanStack) monitor(ctx context.Context) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	failures := make(map[string]int, 2)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, url := range []string{s.config.publicURL(), s.config.privateURL()} {
				response, err := client.Get(url)
				if err == nil {
					_ = response.Body.Close()
					if response.StatusCode >= 200 && response.StatusCode < 500 {
						failures[url] = 0
						continue
					}
					err = fmt.Errorf("HTTP %s", response.Status)
				}
				failures[url]++
				if failures[url] >= 3 {
					return fmt.Errorf("Otterscan explorer at %s became unavailable: %w", url, err)
				}
			}
		}
	}
}

func (s *otterscanStack) printDiagnostics() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.runCompose(ctx, "ps")
	_ = s.runCompose(ctx, "logs", "--tail", "40")
}

func waitForHTTP(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
			closeErr := response.Body.Close()
			switch {
			case readErr != nil:
				lastErr = readErr
			case closeErr != nil:
				lastErr = closeErr
			case response.StatusCode >= 200 && response.StatusCode < 300:
				return nil
			default:
				lastErr = fmt.Errorf("HTTP %s", response.Status)
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", url, errors.Join(ctx.Err(), lastErr))
		case <-ticker.C:
		}
	}
}

func waitForOtterscanRPCs(ctx context.Context, config otterscanConfig) error {
	type endpoint struct {
		name    string
		url     string
		chainID string
	}
	endpoints := []endpoint{
		{"public", fmt.Sprintf("http://127.0.0.1:%d", config.publicRPCPort), config.publicChainID},
		{"private", fmt.Sprintf("http://127.0.0.1:%d", config.privateRPCPort), config.privateChainID},
	}
	for _, endpoint := range endpoints {
		if err := waitForOtterscanRPC(ctx, endpoint.url, endpoint.chainID); err != nil {
			return fmt.Errorf("%s chain Otterscan API: %w", endpoint.name, err)
		}
	}
	return nil
}

func waitForOtterscanRPC(ctx context.Context, url, chainID string) error {
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	wantChainID, err := strconv.ParseUint(chainID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse chain ID %q: %w", chainID, err)
	}
	requestBody := []byte(`[{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]},{"jsonrpc":"2.0","id":2,"method":"ots_getApiLevel","params":[]}]`)
	var lastErr error
	for {
		request, requestErr := http.NewRequestWithContext(waitCtx, http.MethodPost, url, bytes.NewReader(requestBody))
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("Content-Type", "application/json")
		response, requestErr := client.Do(request)
		if requestErr == nil {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			closeErr := response.Body.Close()
			switch {
			case readErr != nil:
				lastErr = readErr
			case closeErr != nil:
				lastErr = closeErr
			case response.StatusCode < 200 || response.StatusCode >= 300:
				lastErr = fmt.Errorf("HTTP %s", response.Status)
			default:
				lastErr = validateOtterscanRPCResponse(body, wantChainID)
				if lastErr == nil {
					return nil
				}
			}
		} else {
			lastErr = requestErr
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wait for %s: %w", url, errors.Join(waitCtx.Err(), lastErr))
		case <-ticker.C:
		}
	}
}

func validateOtterscanRPCResponse(body []byte, wantChainID uint64) error {
	var responses []struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &responses); err != nil {
		return fmt.Errorf("decode batch response: %w", err)
	}
	if len(responses) != 2 {
		return fmt.Errorf("got %d RPC responses, want 2", len(responses))
	}
	byID := make(map[int]json.RawMessage, len(responses))
	for _, response := range responses {
		if response.Error != nil {
			return fmt.Errorf("RPC %d failed: %s", response.ID, response.Error.Message)
		}
		byID[response.ID] = response.Result
	}
	var chainID string
	if err := json.Unmarshal(byID[1], &chainID); err != nil {
		return fmt.Errorf("decode eth_chainId result: %w", err)
	}
	gotChainID, err := strconv.ParseUint(strings.TrimPrefix(chainID, "0x"), 16, 64)
	if err != nil {
		return fmt.Errorf("decode eth_chainId %q: %w", chainID, err)
	}
	if gotChainID != wantChainID {
		return fmt.Errorf("RPC reports chain %d, want %d", gotChainID, wantChainID)
	}
	var apiLevel uint64
	if err := json.Unmarshal(byID[2], &apiLevel); err != nil {
		return fmt.Errorf("decode ots_getApiLevel result: %w", err)
	}
	if apiLevel < minimumOtterscanAPILevel {
		return fmt.Errorf("ots_getApiLevel reports %d, need at least %d", apiLevel, minimumOtterscanAPILevel)
	}
	return nil
}

func ensureHostPortAvailable(port uint) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("port %d is unavailable: %w", port, err)
	}
	return listener.Close()
}

func explorerEnvironment(base []string, config otterscanConfig) []string {
	updates := map[string]string{
		"PUBLIC_RPC_PORT":       strconv.FormatUint(uint64(config.publicRPCPort), 10),
		"PRIVATE_RPC_PORT":      strconv.FormatUint(uint64(config.privateRPCPort), 10),
		"PUBLIC_EXPLORER_PORT":  strconv.FormatUint(uint64(config.publicExplorerPort), 10),
		"PRIVATE_EXPLORER_PORT": strconv.FormatUint(uint64(config.privateExplorerPort), 10),
		"PUBLIC_CHAIN_ID":       config.publicChainID,
		"PRIVATE_CHAIN_ID":      config.privateChainID,
	}
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(base)+len(updates))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if found {
			if _, replaced := updates[key]; replaced {
				continue
			}
		}
		result = append(result, entry)
	}
	for _, key := range keys {
		result = append(result, key+"="+updates[key])
	}
	return result
}
