package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

func sel(sig string) string {
	return hex.EncodeToString(crypto.Keccak256([]byte(sig))[:4])
}

func TestCalldataEncoding(t *testing.T) {
	enc := func(t *testing.T, data []byte, err error) string {
		require.NoError(t, err)
		return hex.EncodeToString(data)
	}
	t.Run("paused", func(t *testing.T) {
		data, err := pausedFn.EncodeArgs()
		require.Equal(t, "5c975abb", enc(t, data, err))
	})
	t.Run("sync", func(t *testing.T) {
		data, err := syncFn.EncodeArgs()
		require.Equal(t, sel("syncETHLiquidityPause()"), enc(t, data, err))
	})
	t.Run("pausedMessageNonce", func(t *testing.T) {
		data, err := pausedMessageNonceFn.EncodeArgs()
		require.Equal(t, sel("pausedMessageNonce()"), enc(t, data, err))
	})
	t.Run("setPaused", func(t *testing.T) {
		data, err := setPausedFn.EncodeArgs(true, big.NewInt(5))
		want := sel("setPaused(bool,uint256)") +
			"0000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000005"
		require.Equal(t, want, enc(t, data, err))
	})
	t.Run("sendETH", func(t *testing.T) {
		to := common.HexToAddress("0x00000000000000000000000000000000000000aa")
		data, err := sendETHFn.EncodeArgs(to, big.NewInt(902))
		want := sel("sendETH(address,uint256)") +
			"00000000000000000000000000000000000000000000000000000000000000aa" +
			"0000000000000000000000000000000000000000000000000000000000000386"
		require.Equal(t, want, enc(t, data, err))
	})
	lockbox := common.HexToAddress("0x1111111111111111111111111111111111111111")
	arg := "0000000000000000000000001111111111111111111111111111111111111111"
	for sig, fn := range map[string]interface {
		EncodeArgs(...any) ([]byte, error)
	}{
		"pause(address)":    scPauseFn,
		"unpause(address)":  scUnpauseFn,
		"extend(address)":   scExtendFn,
		"pausable(address)": pausableFn,
		"paused(address)":   pausedForFn,
	} {
		t.Run(sig, func(t *testing.T) {
			data, err := fn.EncodeArgs(lockbox)
			require.Equal(t, sel(sig)+arg, enc(t, data, err))
		})
	}
}

// revertError mimics a geth execution-reverted RPC error.
type revertError struct{ data string }

func (e *revertError) Error() string          { return "execution reverted" }
func (e *revertError) ErrorCode() int         { return 3 }
func (e *revertError) ErrorData() interface{} { return e.data }

func TestDecodeRevert(t *testing.T) {
	r := decodeRevert(&revertError{data: "0x" + sel("ETHLiquidity_Paused()")})
	require.NotNil(t, r)
	require.Equal(t, pausedErrorName, r.Name)
	require.Equal(t, "0x"+sel("ETHLiquidity_Paused()"), r.Selector)

	r = decodeRevert(fmt.Errorf("wrapped: %w", &revertError{data: "0xdeadbeef00"}))
	require.NotNil(t, r)
	require.Equal(t, "0xdeadbeef", r.Selector)
	require.Empty(t, r.Name)

	r = decodeRevert(&revertError{data: "0x"})
	require.NotNil(t, r)
	require.Empty(t, r.Selector)

	reason, err := abi.Arguments{{Type: abi.Type{T: abi.StringTy}}}.Pack("Proxy: implementation not initialized")
	require.NoError(t, err)
	r = decodeRevert(&revertError{data: "0x08c379a0" + hex.EncodeToString(reason)})
	require.Equal(t, `Error("Proxy: implementation not initialized")`, r.Name)

	// A custom error whose arguments happen to be one string is not Error(string).
	r = decodeRevert(&revertError{data: "0x" + sel("SuperchainConfig_OnlyGuardian()") + hex.EncodeToString(reason)})
	require.Equal(t, "SuperchainConfig_OnlyGuardian()", r.Name)

	code, err := abi.Arguments{{Type: uint256Type}}.Pack(big.NewInt(0x11))
	require.NoError(t, err)
	r = decodeRevert(&revertError{data: "0x4e487b71" + hex.EncodeToString(code)})
	require.Equal(t, "Panic(0x11)", r.Name)
	r = decodeRevert(&revertError{data: "0x4e487b71"})
	require.Equal(t, "Panic(uint256)", r.Name, "a malformed panic keeps the bare signature")

	require.NotNil(t, decodeRevert(errors.New("execution reverted")))
	require.Nil(t, decodeRevert(errors.New("connection refused")))
	require.Nil(t, decodeRevert(nil))
}

func TestRefuseProduction(t *testing.T) {
	for id, name := range productionChains {
		if id == 1 {
			err := refuseProduction(1, []uint64{901})
			require.ErrorContains(t, err, name)
			continue
		}
		err := refuseProduction(11155111, []uint64{901, id})
		require.ErrorContains(t, err, name, "chain %d", id)
		require.ErrorContains(t, err, fmt.Sprint(id))
	}
	for _, id := range []uint64{10, 57073, 130, 11155420, 763373, 1301} {
		require.Contains(t, productionChains, id)
	}
	// Devnet L2s on a Sepolia or local L1.
	require.NoError(t, refuseProduction(11155111, []uint64{420120000, 420120001}))
	require.NoError(t, refuseProduction(900, []uint64{901, 902}))
}

func TestPairChains(t *testing.T) {
	_, err := pairChains(nil, nil, false)
	require.ErrorContains(t, err, "--l2")
	_, err = pairChains([]string{"a"}, nil, true)
	require.ErrorContains(t, err, "--l1-messenger is required")
	_, err = pairChains([]string{"a", "b"}, []string{"0x0000000000000000000000000000000000000001"}, false)
	require.ErrorContains(t, err, "got 2 --l2 and 1 --l1-messenger")
	_, err = pairChains([]string{"a"}, []string{"nope"}, false)
	require.ErrorContains(t, err, "not an address")
	_, err = pairChains([]string{"a", "b"}, []string{
		"0x0000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000000",
	}, false)
	require.ErrorContains(t, err, "--l1-messenger for --l2 b is the zero address")

	pairs, err := pairChains([]string{"a", "b"}, nil, false)
	require.NoError(t, err)
	require.Equal(t, []chainPair{{L2: "a"}, {L2: "b"}}, pairs)
	pairs, err = pairChains([]string{"a", "b"}, []string{
		"0x0000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000002",
	}, true)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0x2"), pairs[1].Messenger)
}

func TestPauseIdentifiersGroupSharedLockbox(t *testing.T) {
	sc := common.HexToAddress("0xc0")
	shared, other := common.HexToAddress("0xb1"), common.HexToAddress("0xb2")
	mk := func(id uint64, lockbox common.Address) chain {
		return chain{L2: &l2Client{ChainID: id}, L1: &l1Contracts{SuperchainConfig: sc, Lockbox: lockbox}}
	}
	ids := pauseIdentifiers([]chain{mk(902, shared), mk(903, other), mk(901, shared)})
	require.Equal(t, []pauseIdentifier{
		{SuperchainConfig: sc, Lockbox: shared, ChainIDs: []uint64{901, 902}},
		{SuperchainConfig: sc, Lockbox: other, ChainIDs: []uint64{903}},
	}, ids)
}

func TestPlanGuardianStep(t *testing.T) {
	for _, tc := range []struct {
		pause, pausable, active bool
		call                    string
	}{
		{pause: true, pausable: true, call: "pause"},
		{pause: true, active: true, call: ""},
		{pause: true, call: "extend"},
		{pause: false, pausable: true, call: ""},
		{pause: false, active: true, call: "unpause"},
		{pause: false, call: "unpause"}, // expired but still set: clear it
	} {
		require.Equal(t, tc.call, planGuardianStep(tc.pause, tc.pausable, tc.active).Call, "%+v", tc)
	}
	require.Equal(t, "pause expired but still set", planGuardianStep(false, false, false).Reason)
	require.Equal(t, "paused", planGuardianStep(false, false, true).Reason)
	require.Contains(t, planGuardianStep(true, false, false).Reason, "pause expired but still set")
}

func TestCheckExpectation(t *testing.T) {
	yes, no := true, false
	pausedRevert := &revertInfo{Name: pausedErrorName, Selector: "0x" + sel(pausedErrorName)}
	paused := chainStatus{ChainID: 901, L2Paused: &yes, SendETHReverts: true, SendETHRevert: pausedRevert}
	unpaused := chainStatus{ChainID: 901, L2Paused: &no}

	require.Empty(t, checkExpectation(paused, expectPaused))
	require.Empty(t, checkExpectation(unpaused, expectUnpaused))
	require.Len(t, checkExpectation(paused, expectUnpaused), 2)
	require.Len(t, checkExpectation(unpaused, expectPaused), 2)

	wrongRevert := paused
	wrongRevert.SendETHRevert = &revertInfo{Name: "Unauthorized()"}
	require.Equal(t, []string{"chain 901: sendETH reverts with Unauthorized() (), not ETHLiquidity_Paused()"},
		checkExpectation(wrongRevert, expectPaused))

	noPauseFn := chainStatus{ChainID: 901, L2PausedError: "execution reverted"}
	require.Len(t, checkExpectation(noPauseFn, expectUnpaused), 1)

	// L1 paused but L2 not yet synced fails both expectations.
	unsynced := unpaused
	unsynced.L1 = &l1Contracts{}
	unsynced.L1Paused = &yes
	require.Len(t, checkExpectation(unsynced, expectUnpaused), 1)
	require.Len(t, checkExpectation(unsynced, expectPaused), 2)
}

func TestConfigFileSuppliesChains(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pause.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
l1 = "http://l1"
l2 = ["http://a", "http://b"]
l1-messenger = ["0x0000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000002"]
key = "abcd"
timeout = "30s"
`), 0o600))
	app := newApp()
	var observed bool
	for _, cmd := range app.Commands {
		if cmd.Name == "sync" {
			cmd.Action = func(c *cli.Context) error {
				observed = true
				require.Equal(t, "http://l1", c.String(L1RPC.Name))
				require.Equal(t, []string{"http://a", "http://b"}, c.StringSlice(L2RPCs.Name))
				require.Len(t, c.StringSlice(L1Messengers.Name), 2)
				require.Equal(t, "override", c.String(Key.Name)) // the CLI flag beats the file
				require.Equal(t, "30s", c.Duration(Timeout.Name).String())
				return nil
			}
		}
	}
	require.NoError(t, app.Run([]string{"eth-liquidity-pause", "sync", "--config", path, "--key", "override"}))
	require.True(t, observed)
}

// fakeChain serves the eth_chainId, eth_blockNumber and eth_call answers the commands need.
type fakeChain struct {
	chainID uint64
	mu      sync.Mutex
	// views maps (contract, 4-byte selector) to a 32-byte return word.
	views map[common.Address]map[string][]byte
	// l2Paused, when set, makes SuperchainETHBridge.sendETH revert with ETHLiquidity_Paused().
	l2Paused bool
	// beforeCall, if set, runs under mu before each eth_call, so tests can change state as the
	// tool polls.
	beforeCall func(c *fakeChain)
}

// setL2Pause sets ETHLiquidity's paused() and pausedMessageNonce(), as an applied
// setPaused(paused, nonce) would.
func (c *fakeChain) setL2Pause(paused bool, nonce *big.Int) {
	c.l2Paused = paused
	c.views[predeploys.ETHLiquidityAddr][sel("paused()")] = word(paused)
	c.views[predeploys.ETHLiquidityAddr][sel("pausedMessageNonce()")] = common.LeftPadBytes(nonce.Bytes(), 32)
}

type callArgs struct {
	From  *common.Address `json:"from"`
	To    *common.Address `json:"to"`
	Input hexutil.Bytes   `json:"input"`
	Value *hexutil.Big    `json:"value"`
}

type fakeEth struct{ c *fakeChain }

func (f *fakeEth) ChainId() hexutil.Uint64 { return hexutil.Uint64(f.c.chainID) }

func (f *fakeEth) BlockNumber() hexutil.Uint64 { return 1 }

func (f *fakeEth) Call(args callArgs, _ string, overrides *map[common.Address]map[string]any) (hexutil.Bytes, error) {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	if f.c.beforeCall != nil {
		f.c.beforeCall(f.c)
	}
	if args.To == nil || len(args.Input) < 4 {
		return nil, errors.New("bad call")
	}
	s := hex.EncodeToString(args.Input[:4])
	if *args.To == predeploys.SuperchainETHBridgeAddr && s == sel("sendETH(address,uint256)") {
		// Like geth, a value transfer from an unfunded sender fails unless overridden.
		if args.Value != nil && args.Value.ToInt().Sign() > 0 {
			if overrides == nil || args.From == nil || (*overrides)[*args.From]["balance"] == nil {
				return nil, errors.New("insufficient funds for transfer")
			}
		}
		if f.c.l2Paused {
			return nil, &revertError{data: "0x" + sel(pausedErrorName)}
		}
		return make([]byte, 32), nil
	}
	if out, ok := f.c.views[*args.To][s]; ok {
		return out, nil
	}
	return nil, &revertError{data: "0x"}
}

func serve(t *testing.T, c *fakeChain) string {
	srv := gethrpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", &fakeEth{c: c}))
	h := httptest.NewServer(srv)
	t.Cleanup(func() { h.Close(); srv.Stop() })
	return h.URL
}

func word(v any) []byte {
	switch v := v.(type) {
	case bool:
		if v {
			return common.LeftPadBytes([]byte{1}, 32)
		}
		return make([]byte, 32)
	case common.Address:
		return common.LeftPadBytes(v.Bytes(), 32)
	case uint64:
		return common.LeftPadBytes(new(big.Int).SetUint64(v).Bytes(), 32)
	}
	panic("unsupported")
}

// fakeL1 wires messenger -> SystemConfig -> portal -> lockbox -> SuperchainConfig for one L2.
func fakeL1(l2ChainID uint64, paused bool) (*fakeChain, common.Address) {
	messenger := common.HexToAddress("0xa1")
	sysCfg, portal := common.HexToAddress("0xa2"), common.HexToAddress("0xa3")
	lockbox, scfg := common.HexToAddress("0xa4"), common.HexToAddress("0xa5")
	return &fakeChain{chainID: 900, views: map[common.Address]map[string][]byte{
		messenger: {sel("systemConfig()"): word(sysCfg), sel("paused()"): word(paused)},
		sysCfg:    {sel("l2ChainId()"): word(l2ChainID), sel("optimismPortal()"): word(portal)},
		portal:    {sel("ethLockbox()"): word(lockbox)},
		lockbox:   {sel("superchainConfig()"): word(scfg)},
	}}, messenger
}

// fakeL2 is an L2 whose ETHLiquidity was never synced, and whose L2CrossDomainMessenger has
// no failed messages.
func fakeL2(chainID uint64, paused bool) *fakeChain {
	return &fakeChain{chainID: chainID, l2Paused: paused, views: map[common.Address]map[string][]byte{
		predeploys.ETHLiquidityAddr:           {sel("paused()"): word(paused), sel("pausedMessageNonce()"): word(uint64(0))},
		predeploys.L2CrossDomainMessengerAddr: {sel("failedMessages(bytes32)"): word(false)},
	}}
}

func runStatus(t *testing.T, args ...string) (string, error) {
	out, _, err := runStatusIO(t, args...)
	return out, err
}

// runStatusIO runs status and returns its stdout and stderr.
func runStatusIO(t *testing.T, args ...string) (string, string, error) {
	app := newApp()
	var out, errOut bytes.Buffer
	app.Writer = &out
	app.ErrWriter = &errOut
	err := app.Run(append([]string{"eth-liquidity-pause", "status"}, args...))
	return out.String(), errOut.String(), err
}

func TestStatusAgainstFakeChains(t *testing.T) {
	t.Run("paused and synced", func(t *testing.T) {
		l1, messenger := fakeL1(901, true)
		l2 := fakeL2(901, true)
		l2.setL2Pause(true, crossdomain.EncodeVersionedNonce(big.NewInt(42), big.NewInt(1)))
		out, err := runStatus(t, "--l2", serve(t, l2), "--l1", serve(t, l1),
			"--l1-messenger", messenger.Hex(), "--expect", "paused")
		require.NoError(t, err, out)
		require.Contains(t, out, "reverts "+pausedErrorName)
		require.Contains(t, out, "L2 ETHLiquidity.paused(): true (synced by L1 message nonce 42 (version 1))")
		require.Contains(t, out, "OK: every chain is paused")
	})
	t.Run("ETHLiquidity without pausedMessageNonce", func(t *testing.T) {
		l2 := fakeL2(901, false)
		delete(l2.views[predeploys.ETHLiquidityAddr], sel("pausedMessageNonce()"))
		out, err := runStatus(t, "--l2", serve(t, l2), "--expect", "unpaused")
		require.NoError(t, err, out)
		require.Contains(t, out, "L2 ETHLiquidity.paused(): false (pausedMessageNonce() failed: execution reverted)")
	})
	t.Run("never synced", func(t *testing.T) {
		out, err := runStatus(t, "--l2", serve(t, fakeL2(901, false)))
		require.NoError(t, err, out)
		require.Contains(t, out, "L2 ETHLiquidity.paused(): false (never synced)")
	})
	t.Run("zero messenger is rejected, not read as L2-only", func(t *testing.T) {
		// L1 is unpaused, so reading it would fail --expect paused; a zero messenger used to
		// skip L1 and report OK.
		l1, _ := fakeL1(901, false)
		_, err := runStatus(t, "--l2", serve(t, fakeL2(901, true)), "--l1", serve(t, l1),
			"--l1-messenger", common.Address{}.Hex(), "--expect", "paused")
		require.ErrorContains(t, err, "zero address")
	})
	t.Run("--l1 without messengers", func(t *testing.T) {
		l1, _ := fakeL1(901, false)
		_, err := runStatus(t, "--l2", serve(t, fakeL2(901, true)), "--l1", serve(t, l1), "--expect", "paused")
		require.ErrorContains(t, err, "--l1 needs one --l1-messenger per --l2")
	})
	t.Run("unpaused without L1", func(t *testing.T) {
		out, err := runStatus(t, "--l2", serve(t, fakeL2(901, false)), "--l2", serve(t, fakeL2(902, false)),
			"--expect", "unpaused")
		require.NoError(t, err, out)
		require.Equal(t, 2, strings.Count(out, "sendETH eth_call:         succeeds"))
	})
	t.Run("L1 paused but L2 not synced", func(t *testing.T) {
		l1, messenger := fakeL1(901, true)
		out, err := runStatus(t, "--l2", serve(t, fakeL2(901, false)), "--l1", serve(t, l1),
			"--l1-messenger", messenger.Hex(), "--expect", "paused")
		require.ErrorContains(t, err, "ETHLiquidity.paused() is false")
		require.Contains(t, out, "OUT OF SYNC: L1 paused=true but L2 paused=false")
	})
	t.Run("pre-pause ETHLiquidity", func(t *testing.T) {
		l2 := fakeL2(901, false)
		delete(l2.views, predeploys.ETHLiquidityAddr)
		out, err := runStatus(t, "--l2", serve(t, l2), "--expect", "unpaused")
		require.ErrorContains(t, err, "ETHLiquidity.paused() failed")
		require.Contains(t, out, "L2 ETHLiquidity.paused(): error:")
	})
	t.Run("messenger paired with the wrong chain", func(t *testing.T) {
		l1, messenger := fakeL1(999, false)
		_, err := runStatus(t, "--l2", serve(t, fakeL2(901, false)), "--l1", serve(t, l1),
			"--l1-messenger", messenger.Hex())
		require.ErrorContains(t, err, "belongs to chain 999")
	})
	t.Run("json", func(t *testing.T) {
		l2 := fakeL2(901, true)
		l2.setL2Pause(true, big.NewInt(0x2a))
		out, err := runStatus(t, "--l2", serve(t, l2), "--json")
		require.NoError(t, err)
		require.Contains(t, out, `"l2Paused": true`)
		require.Contains(t, out, `"l2PausedMessageNonce": "0x2a"`)
		require.Contains(t, out, `"name": "ETHLiquidity_Paused()"`)
	})
	t.Run("json with expect keeps stdout valid JSON", func(t *testing.T) {
		l1, messenger := fakeL1(901, true)
		out, errOut, err := runStatusIO(t, "--l2", serve(t, fakeL2(901, true)), "--l1", serve(t, l1),
			"--l1-messenger", messenger.Hex(), "--json", "--expect", "paused")
		require.NoError(t, err)
		var statuses []chainStatus
		require.NoError(t, json.Unmarshal([]byte(out), &statuses), out)
		require.Len(t, statuses, 1)
		require.Contains(t, errOut, "OK: every chain is paused")
	})
}

func TestExampleConfigLoadsForEveryCommand(t *testing.T) {
	app := newApp()
	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	ran := make(map[string]bool)
	for _, cmd := range app.Commands {
		name := cmd.Name
		cmd.Action = func(c *cli.Context) error {
			ran[name] = true
			require.Equal(t, []string{"http://localhost:9545", "http://localhost:9546"}, c.StringSlice(L2RPCs.Name))
			// The L1 side ships commented out, so an unfilled copy can't pass placeholders.
			require.Empty(t, c.String(L1RPC.Name))
			require.Empty(t, c.StringSlice(L1Messengers.Name))
			return nil
		}
	}
	for _, cmd := range app.Commands {
		require.NoError(t, app.Run([]string{"eth-liquidity-pause", cmd.Name, "--config", "config.example.toml"}), cmd.Name)
	}
	require.Equal(t, map[string]bool{"status": true, "sync": true, "devnet-pause": true, "devnet-unpause": true}, ran)
}

// TestABIMatchesContractSnapshots checks the hand-written signatures against the
// contracts-bedrock ABI snapshots, so a rename on the contract side fails here.
func TestABIMatchesContractSnapshots(t *testing.T) {
	load := func(t *testing.T, name string) abi.ABI {
		f, err := os.Open(filepath.Join("..", "..", "..", "packages", "contracts-bedrock", "snapshots", "abi", name+".json"))
		require.NoError(t, err)
		defer f.Close()
		parsed, err := abi.JSON(f)
		require.NoError(t, err)
		return parsed
	}
	methods := map[string][]string{
		"ETHLiquidity":           {"paused()", "pausedMessageNonce()", "setPaused(bool,uint256)"},
		"L2CrossDomainMessenger": {"failedMessages(bytes32)"},
		"SuperchainETHBridge":    {"sendETH(address,uint256)"},
		"L1CrossDomainMessenger": {syncSignature, "paused()", "systemConfig()"},
		"SystemConfig":           {"l2ChainId()", "optimismPortal()"},
		"OptimismPortal2":        {"ethLockbox()"},
		"ETHLockbox":             {"superchainConfig()"},
		"SuperchainConfig": {"guardian()", "paused(address)", "pausable(address)",
			"pause(address)", "unpause(address)", "extend(address)"},
	}
	errs := map[string][]string{
		"ETHLiquidity":           {pausedErrorName, "ETHLiquidity_NotL1CrossDomainMessenger()"},
		"L1CrossDomainMessenger": {"L1CrossDomainMessenger_InteropNotEnabled()"},
		"SuperchainConfig": {"SuperchainConfig_OnlyGuardian()", "SuperchainConfig_AlreadyPaused(address)",
			"SuperchainConfig_NotAlreadyPaused(address)"},
	}
	events := map[string][]string{
		"L1CrossDomainMessenger": {sentMessageEvt.Signature},
	}
	for contract, sigs := range events {
		parsed := load(t, contract)
		for _, sig := range sigs {
			found := false
			for _, e := range parsed.Events {
				found = found || e.Sig == sig
			}
			require.True(t, found, "%s has no event %s", contract, sig)
		}
	}
	for contract, sigs := range methods {
		parsed := load(t, contract)
		for _, sig := range sigs {
			found := false
			for _, m := range parsed.Methods {
				found = found || m.Sig == sig
			}
			require.True(t, found, "%s has no %s", contract, sig)
		}
		for _, sig := range errs[contract] {
			found := false
			for _, e := range parsed.Errors {
				found = found || e.Sig == sig
			}
			require.True(t, found, "%s has no error %s", contract, sig)
			require.Contains(t, knownErrors, selector(sig))
		}
	}
}
