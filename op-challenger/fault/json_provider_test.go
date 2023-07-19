package fault

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	traceLen      = 32
	testDirectory = "__test_dir"
)

// beforeTestHook is meant to be called before each [JsonProvider] test.
// It creates a directory with [traceLen] number of files.
func beforeTestHook(t *testing.T) {
	err := os.MkdirAll(testDirectory, 0755)
	require.NoError(t, err)

	for i := 0; i < traceLen; i++ {
		content, err := json.Marshal("{}")
		require.NoError(t, err)
		require.NoError(t, ioutil.WriteFile(testDirectory+"/"+fmt.Sprint(i)+".json", content, 0644))
	}

}

// postTestHook is meant to be called after each [JsonProvider] test.
// It removes the test directory and all files in it.
func postTestHook(t *testing.T) {
	err := os.RemoveAll("__test_dir")
	require.NoError(t, err)
}

// wrapTest is a helper function that wraps a test function with the
// before and after hooks.
func wrapTest(t *testing.T, test func(t *testing.T)) {
	beforeTestHook(t)
	// TODO: verify that the before test hook created the test directory
	test(t)
	postTestHook(t)
	// TODO: verify that the post test hook removed the test directory
}

// TestJsonProvider_WithHooks tests the [JsonProvider] with the before and after hooks.
func TestJsonProvider_WithHooks(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"TestJsonProvider_GetPreimage", JsonProvider_GetPreimage_Test},
		{"TestJsonProvider_Get", JsonProvider_Get_Test},
		{"TestJsonProvider_AbsolutePreState", JsonProvider_AbsolutePreState_Test},
	}
	for _, test := range tests {
		wrapTest(t, test.test)
	}
}

// JsonProvider_GetPreimage_Test tests the [JsonProvider.GetPreimage] function.
func JsonProvider_GetPreimage_Test(t *testing.T) {
	provider, err := NewJsonProvider(testDirectory, 5)
	require.NoError(t, err)

	// Test that the preimage for the first index is the first file.
	preimage, err := provider.GetPreimage(0)
	require.NoError(t, err)
	require.Equal(t, IndexToBytes(0), preimage)

	// Test that the preimage for the last index is the last file.
	preimage, err = provider.GetPreimage(traceLen - 1)
	require.NoError(t, err)
	require.Equal(t, IndexToBytes(traceLen-1), preimage)

	// Test that the preimage for an index larger than the trace length is the last file.
	preimage, err = provider.GetPreimage(traceLen)
	require.NoError(t, err)
	require.Equal(t, IndexToBytes(traceLen-1), preimage)

	// Test that the preimage for an index larger than the maximum index is an error.
	preimage, err = provider.GetPreimage(traceLen + 1)
	require.Error(t, err)
	require.Equal(t, []byte{}, preimage)
	require.Equal(t, ErrIndexTooLarge, err)
}

// JsonProvider_Get_Test tests the [JsonProvider.Get] function.
func JsonProvider_Get_Test(t *testing.T) {
	provider, err := NewJsonProvider(testDirectory, 5)
	require.NoError(t, err)

	// Test that the first index is the first file.
	content, err := provider.Get(0)
	require.NoError(t, err)
	require.Equal(t, "{}", content.String())

	// Test that the last index is the last file.
	content, err = provider.Get(traceLen - 1)
	require.NoError(t, err)
	require.Equal(t, "{}", content.String())

	// Test that an index larger than the trace length is the last file.
	content, err = provider.Get(traceLen)
	require.NoError(t, err)
	require.Equal(t, "{}", content.String())

	// Test that an index larger than the maximum index is an error.
	content, err = provider.Get(traceLen + 1)
	require.Error(t, err)
	require.Equal(t, []byte{}, content)
	require.Equal(t, ErrIndexTooLarge, err)
}

// JsonProvider_AbsolutePreState_Test tests the [JsonProvider.AbsolutePreState] function.
func JsonProvider_AbsolutePreState_Test(t *testing.T) {
	provider, err := NewJsonProvider(testDirectory, 5)
	require.NoError(t, err)

	// Test that the first index is the first file.
	content := provider.AbsolutePreState()
	require.Equal(t, "{}", string(content))
}
