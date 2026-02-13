package forge

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	versionRegexp = regexp.MustCompile(`(?i)forge version: (.*)\ncommit sha: ([a-f0-9]+)\n`)
	sigilRegexp   = regexp.MustCompile(`(?i)== Return ==\n0: bytes 0x([a-f0-9]+)\n`)
)

type VersionInfo struct {
	Semver string
	SHA    string
}

type Client struct {
	Binary Binary
	Stdout io.Writer
	Stderr io.Writer
	Wd     string
}

func NewStandardClient(workdir string) (*Client, error) {
	forgeBinary, err := NewStandardBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize forge binary: %w", err)
	}
	if err := forgeBinary.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure forge binary: %w", err)
	}

	forgeClient := NewClient(forgeBinary)

	// Determine the working directory for forge
	// The artifacts FS points to a subdirectory (e.g., "forge-artifacts" or "out"),
	// but forge needs to run from the parent directory where foundry.toml is located.
	// This matches the structure from ExtractEmbedded/ExtractFromFile where:
	// - untarPath/forge-artifacts/ contains the artifacts
	// - untarPath/foundry.toml is the config file
	// This can be removed once we remove the OPCM code and use the artifacts FS directly.
	var forgeWd string
	info, err := os.Stat(workdir)
	if err != nil {
		forgeWd = workdir
	} else if info.IsDir() {
		foundryToml := filepath.Join(workdir, "foundry.toml")
		if _, err := os.Stat(foundryToml); err == nil {
			forgeWd = workdir
		} else {
			// foundry.toml not found, check parent directory
			// This handles the case where workdir points to forge-artifacts/ or out/
			parent := filepath.Dir(workdir)
			parentFoundryToml := filepath.Join(parent, "foundry.toml")
			if _, err := os.Stat(parentFoundryToml); err == nil {
				forgeWd = parent
			} else {
				forgeWd = workdir
			}
		}
	} else {
		forgeWd = filepath.Dir(workdir)
	}

	if err := os.MkdirAll(forgeWd, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure forge working directory exists: %w", err)
	}

	forgeClient.Wd = forgeWd
	fmt.Printf("Forge client working directory: %s\n", forgeClient.Wd)

	return forgeClient, nil
}

func NewClient(binary Binary) *Client {
	return &Client{
		Binary: binary,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (c *Client) Version(ctx context.Context) (VersionInfo, error) {
	buf := new(bytes.Buffer)
	if err := c.execCmd(ctx, buf, io.Discard, "--version"); err != nil {
		return VersionInfo{}, fmt.Errorf("failed to run forge version command: %w", err)
	}
	outputStr := buf.String()
	matches := versionRegexp.FindAllStringSubmatch(outputStr, -1)
	if len(matches) != 1 || len(matches[0]) != 3 {
		return VersionInfo{}, fmt.Errorf("failed to find forge version in output:\n%s", outputStr)
	}
	return VersionInfo{
		Semver: matches[0][1],
		SHA:    matches[0][2],
	}, nil
}

func (c *Client) Build(ctx context.Context, opts ...string) error {
	return c.execCmd(ctx, io.Discard, io.Discard, append([]string{"build"}, opts...)...)
}

func (c *Client) Clean(ctx context.Context, opts ...string) error {
	return c.execCmd(ctx, io.Discard, io.Discard, append([]string{"clean"}, opts...)...)
}

func (c *Client) RunScript(ctx context.Context, script string, sig string, args []byte, opts ...string) (string, error) {
	buf := new(bytes.Buffer)
	cliOpts := []string{"script"}
	cliOpts = append(cliOpts, opts...)
	cliOpts = append(cliOpts, "--sig", sig, script, "0x"+hex.EncodeToString(args))
	if err := c.execCmd(ctx, buf, io.Discard, cliOpts...); err != nil {
		return "", fmt.Errorf("failed to execute forge script: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) VerifyContract(ctx context.Context, opts ...string) (string, error) {
	buf := new(bytes.Buffer)
	cliOpts := []string{"verify-contract"}
	cliOpts = append(cliOpts, opts...)
	if err := c.execCmd(ctx, buf, buf, cliOpts...); err != nil {
		return buf.String(), fmt.Errorf("failed to verify contract: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) execCmd(ctx context.Context, stdout io.Writer, stderr io.Writer, args ...string) error {
	if err := c.Binary.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure binary: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.Binary.Path(), args...)
	cStdout := c.Stdout
	if cStdout == nil {
		cStdout = os.Stdout
	}
	cStderr := c.Stderr
	if cStderr == nil {
		cStderr = os.Stderr
	}

	mwStdout := io.MultiWriter(cStdout, stdout)
	mwStderr := io.MultiWriter(cStderr, stderr)
	cmd.Stdout = mwStdout
	cmd.Stderr = mwStderr
	cmd.Dir = c.Wd
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run forge command: %w", err)
	}
	return nil
}

type ScriptCallEncoder[I any] interface {
	Encode(I) ([]byte, error)
}

type ScriptCallDecoder[O any] interface {
	Decode(raw []byte) (O, error)
}

// ScriptCaller is a function that calls a forge script
// Ouputs:
// - Return value of the script (decoded into go type)
// - Bool indicating if the script was recompiled (mostly used for testing)
// - Error if the script fails to run
type ScriptCaller[I any, O any] func(ctx context.Context, input I, opts ...string) (O, bool, error)

func NewScriptCaller[I any, O any](client *Client, script string, sig string, encoder ScriptCallEncoder[I], decoder ScriptCallDecoder[O]) ScriptCaller[I, O] {
	return func(ctx context.Context, input I, opts ...string) (O, bool, error) {
		var out O
		encArgs, err := encoder.Encode(input)
		if err != nil {
			return out, false, fmt.Errorf("failed to encode forge args: %w", err)
		}
		rawOut, err := client.RunScript(ctx, script, sig, encArgs, opts...)
		if err != nil {
			return out, false, fmt.Errorf("failed to run forge script: %w", err)
		}
		sigilMatches := sigilRegexp.FindAllStringSubmatch(rawOut, -1)
		if len(sigilMatches) != 1 || len(sigilMatches[0]) != 2 {
			return out, false, fmt.Errorf("failed to find forge return value in output:\n%s", rawOut)
		}
		decoded, err := hex.DecodeString(sigilMatches[0][1])
		if err != nil {
			return out, false, fmt.Errorf("failed to decode forge return value %s: %w", sigilMatches[0][1], err)
		}
		out, err = decoder.Decode(decoded)
		if err != nil {
			return out, false, fmt.Errorf("failed to decode forge output: %w", err)
		}
		return out, strings.Contains(rawOut, "Compiler run successful!"), nil
	}
}
