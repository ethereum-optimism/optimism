// Package rustengine is a spike-only Go client for the Rust op-script-engine, used by the
// parity test to drive the Rust engine over a Unix-socket JSON-RPC connection
// (go-ethereum rpc.DialIPC, reth-ipc newline framing; the #20415 transport).
package rustengine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// Engine is a handle to a spawned Rust op-script-engine subprocess.
type Engine struct {
	cmd     *exec.Cmd
	cl      *rpc.Client
	tmpDir  string
}

// SpawnOpts is the engine host context: the fields of script.Context (plus host options)
// that the engine supports. Zero values match script.DefaultContext apart from ChainID.
type SpawnOpts struct {
	ArtifactsDir    string
	ChainID         uint64
	Create2Deployer bool
	NoMaxCodeSize   bool
	BlockNum        uint64
	Timestamp       uint64
	PrevRandao      common.Hash
}

// Spawn launches the Rust engine binary and dials its Unix socket. The child's stderr is
// forwarded to logw (engine tracing). Call Close to terminate it.
func Spawn(binPath string, opts SpawnOpts, logw io.Writer) (*Engine, error) {
	tmpDir, err := os.MkdirTemp("", "op-script-engine")
	if err != nil {
		return nil, err
	}
	sock := filepath.Join(tmpDir, "engine.sock")

	args := []string{"--socket", sock, "--chain-id", fmt.Sprintf("%d", opts.ChainID), "--artifacts", opts.ArtifactsDir}
	if opts.Create2Deployer {
		args = append(args, "--create2-deployer")
	}
	if opts.NoMaxCodeSize {
		args = append(args, "--no-max-code-size")
	}
	if opts.BlockNum != 0 {
		args = append(args, "--block-num", fmt.Sprintf("%d", opts.BlockNum))
	}
	if opts.Timestamp != 0 {
		args = append(args, "--timestamp", fmt.Sprintf("%d", opts.Timestamp))
	}
	if opts.PrevRandao != (common.Hash{}) {
		args = append(args, "--prev-randao", opts.PrevRandao.Hex())
	}
	cmd := exec.Command(binPath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			fmt.Fprintf(logw, "[engine] %s\n", sc.Text())
		}
	}()

	e := &Engine{cmd: cmd, tmpDir: tmpDir}
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, statErr := os.Stat(sock); statErr == nil {
			cl, dialErr := rpc.DialIPC(context.Background(), sock)
			if dialErr == nil {
				e.cl = cl
				return e, nil
			}
		}
		if time.Now().After(deadline) {
			e.Close()
			return nil, fmt.Errorf("engine never became ready on %s", sock)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (e *Engine) Close() {
	if e.cl != nil {
		e.cl.Close()
	}
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
	}
	if e.tmpDir != "" {
		_ = os.RemoveAll(e.tmpDir)
	}
}

func (e *Engine) LoadContract(file, contract string, from common.Address) (common.Address, error) {
	var out common.Address
	err := e.cl.Call(&out, "script_loadContract", file, contract, from.Hex())
	return out, err
}

func (e *Engine) AllowCheatcodes(addr common.Address) error {
	var ok bool
	return e.cl.Call(&ok, "script_allowCheatcodes", addr.Hex())
}

func (e *Engine) SetEnv(key, value string) error {
	var ok bool
	return e.cl.Call(&ok, "script_setEnv", key, value)
}

func (e *Engine) Call(from, to common.Address, input []byte) ([]byte, error) {
	var out hexutil.Bytes
	err := e.cl.Call(&out, "script_call", from.Hex(), to.Hex(), hexutil.Encode(input))
	return out, err
}

// RunScript deploys a forge script from the script-deployer, runs its run(input) entrypoint from
// deployer, then wipes the script account — mirroring the Go host's DeployScript.Call flow.
func (e *Engine) RunScript(file, contract string, calldata []byte, deployer common.Address) ([]byte, error) {
	var out hexutil.Bytes
	err := e.cl.Call(&out, "script_runScript", file, contract, hexutil.Encode(calldata), deployer.Hex())
	return out, err
}

// Wipe clears an account's code/nonce/balance, matching script.Host.Wipe.
func (e *Engine) Wipe(addr common.Address) error {
	var ok bool
	return e.cl.Call(&ok, "script_wipe", addr.Hex())
}

func (e *Engine) GetNonce(addr common.Address) (uint64, error) {
	var out uint64
	err := e.cl.Call(&out, "script_getNonce", addr.Hex())
	return out, err
}

// InstallInputPrecompile installs a getter-snapshot precompile (OPCM RunScript* input) at a
// freshly minted script address and returns that address, to be passed as an ABI arg to run().
func (e *Engine) InstallInputPrecompile(snapshot map[[4]byte][]byte) (common.Address, error) {
	m := make(map[string]string, len(snapshot))
	for sel, data := range snapshot {
		m[hexutil.Encode(sel[:])] = hexutil.Encode(data)
	}
	var out common.Address
	err := e.cl.Call(&out, "script_installInputPrecompile", m)
	return out, err
}

// InstallOutputPrecompile installs a setter-capture precompile (OPCM RunScriptSingle output) at a
// freshly minted script address and returns that address. getters are the output struct's valid
// field-getter selectors.
func (e *Engine) InstallOutputPrecompile(getters [][4]byte) (common.Address, error) {
	sels := make([]string, len(getters))
	for i, g := range getters {
		sels[i] = hexutil.Encode(g[:])
	}
	var out common.Address
	err := e.cl.Call(&out, "script_installOutputPrecompile", sels)
	return out, err
}

// TakeCapturedSets drains the raw set() calldata captured by an output precompile, in call order,
// for replay through the Go WithFieldSetter precompile.
func (e *Engine) TakeCapturedSets(addr common.Address) ([][]byte, error) {
	var raw []hexutil.Bytes
	if err := e.cl.Call(&raw, "script_takeCapturedSets", addr.Hex()); err != nil {
		return nil, err
	}
	out := make([][]byte, len(raw))
	for i, b := range raw {
		out[i] = b
	}
	return out, nil
}

// RemovePrecompile removes an installed input/output precompile override.
func (e *Engine) RemovePrecompile(addr common.Address) error {
	var ok bool
	return e.cl.Call(&ok, "script_removePrecompile", addr.Hex())
}

func (e *Engine) StateDump() (*foundry.ForgeAllocs, error) {
	var raw json.RawMessage
	if err := e.cl.Call(&raw, "script_stateDump"); err != nil {
		return nil, err
	}
	var allocs foundry.ForgeAllocs
	if err := json.Unmarshal(raw, &allocs); err != nil {
		return nil, fmt.Errorf("decode state dump: %w", err)
	}
	return &allocs, nil
}

func (e *Engine) TakeBroadcasts() ([]script.Broadcast, error) {
	var out []script.Broadcast
	err := e.cl.Call(&out, "script_takeBroadcasts")
	return out, err
}
