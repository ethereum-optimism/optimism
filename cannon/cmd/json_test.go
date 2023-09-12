package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundTripJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json")
	data := &jsonTestData{A: "yay", B: 3}
	err := writeJSON(file, data)
	require.NoError(t, err)

	// Confirm the file is uncompressed
	fileContent, err := os.ReadFile(file)
	require.NoError(t, err)
	err = json.Unmarshal(fileContent, &jsonTestData{})
	require.NoError(t, err)

	var result *jsonTestData
	result, err = loadJSON[jsonTestData](file)
	require.NoError(t, err)
	require.EqualValues(t, data, result)
}

func TestRoundTripJSONWithGzip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json.gz")
	data := &jsonTestData{A: "yay", B: 3}
	err := writeJSON(file, data)
	require.NoError(t, err)

	// Confirm the file isn't raw JSON
	fileContent, err := os.ReadFile(file)
	require.NoError(t, err)
	err = json.Unmarshal(fileContent, &jsonTestData{})
	require.Error(t, err, "should not be able to decode without decompressing")

	var result *jsonTestData
	result, err = loadJSON[jsonTestData](file)
	require.NoError(t, err)
	require.EqualValues(t, data, result)
}

type jsonTestData struct {
	A string `json:"a"`
	B int    `json:"b"`
}
