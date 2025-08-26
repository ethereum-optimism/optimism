package forge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/stretchr/testify/require"
)

type ioStruct struct {
	ID   uint8
	Data []byte
}

func TestMinimalSources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cl := NewClient(PathBinary())
	cl.Wd = projDir(t)

	// Build artifacts
	require.NoError(t, cl.Build(ctx))

	// Then copy them somewhere else
	tmpDir := t.TempDir()
	require.NoError(t, copyDir("testdata/testproject/out", path.Join(tmpDir, "out")))
	require.NoError(t, copyDir("testdata/testproject/cache", path.Join(tmpDir, "cache")))
	require.NoError(t, copyDir("testdata/testproject/script", path.Join(tmpDir, "script")))
	require.NoError(t, copyDir("testdata/testproject/foundry.toml", path.Join(tmpDir, "foundry.toml")))

	// Then see if we can successfully run a script
	cl.Wd = tmpDir
	encDec := new(testEncoderDecoder)
	caller := NewScriptCaller[ioStruct, ioStruct](cl, "script/Test.s.sol:TestScript", "run(bytes)", encDec, encDec)
	// It should not recompile since we included the cache.
	in := ioStruct{
		ID:   1,
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}
	out, changed, err := caller(ctx, in)
	require.NoError(t, err)
	require.False(t, changed)
	require.EqualValues(t, ioStruct{
		ID:   2,
		Data: in.Data,
	}, out)
}

// TestClient_Smoke smoke tests the Client by running the Version command on it.
func TestClient_Smoke(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)

	version, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`\d+\.\d+\.\d+`), version.Semver)
	require.Regexp(t, regexp.MustCompile(`^[a-f0-9]+$`), version.SHA)
}

func TestClient_OutputRedirection(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)
	cl.Stdout = new(bytes.Buffer)

	_, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(cl.Stdout.(*bytes.Buffer).String(), "forge Version"))
}

func TestScriptCaller(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bin := PathBinary()
	cl := NewClient(bin)
	cl.Wd = projDir(t)

	require.NoError(t, cl.Clean(ctx))
	encDec := new(testEncoderDecoder)
	caller := NewScriptCaller[ioStruct, ioStruct](cl, "script/Test.s.sol:TestScript", "run(bytes)", encDec, encDec)
	in := ioStruct{
		ID:   1,
		Data: []byte{0x01, 0x02},
	}
	out, recompiled, err := caller(context.Background(), in)
	require.NoError(t, err)
	require.True(t, recompiled)
	require.EqualValues(t, ioStruct{
		ID:   2,
		Data: []byte{0x01, 0x02},
	}, out)
	out, recompiled, err = caller(context.Background(), in)
	require.NoError(t, err)
	require.False(t, recompiled)
	require.EqualValues(t, ioStruct{
		ID:   2,
		Data: []byte{0x01, 0x02},
	}, out)
}

var (
	runArgs = abi.Arguments{{Type: mustTuple()}}
)

func mustTuple() abi.Type {
	t, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "ID", Type: "uint8"},
		{Name: "Data", Type: "bytes"},
	})
	if err != nil {
		panic(err)
	}
	return t
}

type testEncoderDecoder struct {
}

func (t *testEncoderDecoder) Encode(in ioStruct) ([]byte, error) {
	return runArgs.Pack(in)
}

func (t *testEncoderDecoder) Decode(v []byte) (ioStruct, error) {
	var out ioStruct
	decoded, err := runArgs.Unpack(v)
	if err != nil {
		return out, fmt.Errorf("error unpacking args: %w", err)
	}
	// Geth's ABI decoding library returns an anonymous strut
	// which requires reflection to parse.
	anonStruct := decoded[0]
	val := reflect.ValueOf(anonStruct)
	out.ID = uint8(val.FieldByName("ID").Uint())
	out.Data = val.FieldByName("Data").Bytes()
	return out, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func projDir(t *testing.T) string {
	_, testFilename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(testFilename), "testdata", "testproject")
	absProjDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	return absProjDir
}
