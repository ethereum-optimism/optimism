package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const testManifest = `{"transactions":[{"transactionType":"CALL","transaction":{"from":"0x0000000000000000000000000000000000000046","to":"0x0000000000000000000000000000000000000054","nonce":"0x3","value":"0x7","input":"0x0102","chainId":"0x7a69"}}]}`

func TestReadTransactions(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(map[string]any, map[string]any)
		err    string
	}{
		{name: "valid"},
		{name: "missing nonce", mutate: func(_ map[string]any, tx map[string]any) { delete(tx, "nonce") }, err: "missing nonce"},
		{name: "null value", mutate: func(_ map[string]any, tx map[string]any) { tx["value"] = nil }, err: "missing value"},
		{name: "missing target", mutate: func(_ map[string]any, tx map[string]any) { delete(tx, "to") }, err: "missing to"},
		{name: "authorizations", mutate: func(_ map[string]any, tx map[string]any) { tx["authorizationList"] = []any{} }, err: "unsupported transaction field"},
		{name: "blobs", mutate: func(_ map[string]any, tx map[string]any) { tx["blobVersionedHashes"] = []any{} }, err: "unsupported transaction field"},
		{name: "unknown kind", mutate: func(record map[string]any, _ map[string]any) { record["transactionType"] = "FUTURE" }, err: "unsupported forge transaction kind"},
		{name: "create with target", mutate: func(record map[string]any, _ map[string]any) { record["transactionType"] = "CREATE" }, err: "has a target"},
		{name: "call without target", mutate: func(_ map[string]any, tx map[string]any) { tx["to"] = nil }, err: "has no target"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(testManifest), &data))
			record := data["transactions"].([]any)[0].(map[string]any)
			if tt.mutate != nil {
				tt.mutate(record, record["transaction"].(map[string]any))
			}
			encoded, err := json.Marshal(data)
			require.NoError(t, err)
			dir := t.TempDir()
			writeManifest(t, dir, "Script.s.sol", string(encoded))
			out, err := readTransactions(dir, "Script ran successfully.")
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Len(t, out, 1)
			require.Equal(t, common.Address{19: 'F'}, out[0].From)
			require.Equal(t, common.Address{19: 'T'}, *out[0].To)
			require.EqualValues(t, 3, out[0].Nonce)
			require.Equal(t, []byte{1, 2}, []byte(out[0].Input))
		})
	}
}

func TestReadTransactionsRequiresOneInvocation(t *testing.T) {
	dir := t.TempDir()
	_, err := readTransactions(dir, "")
	require.ErrorContains(t, err, "expected one")

	_, err = readTransactions(dir, "Script ran successfully.\nTransactions saved to: unsupported/multi/dry-run/run-latest.json")
	require.ErrorContains(t, err, "expected one", "an unsupported manifest layout is not a no-op")

	out, err := readTransactions(dir, "Script ran successfully.")
	require.NoError(t, err)
	require.Empty(t, out, "forge emits no file for a no-op script")

	writeManifest(t, dir, "A.s.sol", `{"transactions":[]}`)
	out, err = readTransactions(dir, "Script ran successfully.")
	require.NoError(t, err)
	require.Empty(t, out)

	writeManifest(t, dir, "B.s.sol", testManifest)
	_, err = readTransactions(dir, "Script ran successfully.")
	require.ErrorContains(t, err, "expected one")
}

func TestReadTransactionsRejectsInvalidManifest(t *testing.T) {
	for _, data := range []string{`{`, `{}`, `{"transactions":null}`} {
		dir := t.TempDir()
		writeManifest(t, dir, "Script.s.sol", data)
		_, err := readTransactions(dir, "Script ran successfully.")
		require.Error(t, err)
	}
}

// TestCaptureTransactions reads only the private per-run file, not any shared project's
// broadcast directory. A second capture gets a fresh directory, so stale/injected records
// from another invocation cannot become transactions to sign.
func TestCaptureTransactions(t *testing.T) {
	project := stubProject(t)
	writeManifest(t, filepath.Join(project, "broadcast"), "Injected.s.sol", `{"transactions":[{"transactionType":"CALL","transaction":{}}]}`)
	cl := NewClient(stubBinary(t, fmt.Sprintf(`
mkdir -p "$FOUNDRY_BROADCAST/Script.s.sol/31337/dry-run"
printf '%%s' '%s' > "$FOUNDRY_BROADCAST/Script.s.sol/31337/dry-run/run-latest.json"
echo 'Script ran successfully.'`, testManifest)))
	cl.Wd = project
	cl.Stdout = io.Discard
	for range 2 {
		out, err := cl.CaptureTransactions(func(client *Client) error {
			_, err := client.RunScript(context.Background(), "Script.s.sol", "run()", nil)
			return err
		})
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, common.Address{19: 'T'}, *out[0].To)
	}
}

func TestCaptureTransactionsRequiresSingleScript(t *testing.T) {
	cl := NewClient(stubBinary(t, "echo 'Script ran successfully.'"))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard
	_, err := cl.CaptureTransactions(func(*Client) error { return nil })
	require.ErrorContains(t, err, "did not capture a script run")

	_, err = cl.CaptureTransactions(func(client *Client) error {
		if _, err := client.RunScript(context.Background(), "Script.s.sol", "run()", nil); err != nil {
			return err
		}
		_, err := client.RunScript(context.Background(), "Script.s.sol", "run()", nil)
		return err
	})
	require.ErrorContains(t, err, "exactly one script run")
}

func TestCaptureTransactionsRejectsBroadcast(t *testing.T) {
	cl := NewClient(stubBinary(t, "exit 99"))
	cl.Wd = stubProject(t)
	for _, flag := range []string{"--broadcast", "--broadcast=true", "--resume"} {
		_, err := cl.CaptureTransactions(func(client *Client) error {
			_, err := client.RunScript(context.Background(), "Script.s.sol", "run()", nil, flag)
			return err
		})
		require.ErrorContains(t, err, "keyless dry run")
	}
}

func writeManifest(t *testing.T, root, script, data string) {
	t.Helper()
	dir := filepath.Join(root, script, "31337", "dry-run")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run-latest.json"), []byte(data), 0o600))
}
