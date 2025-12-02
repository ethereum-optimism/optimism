package jsonutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/stretchr/testify/require"
)

func TestRoundTripJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json")
	data := &jsonTestData{A: "yay", B: 3}
	err := WriteJSON(data, ioutil.ToAtomicFile(file, 0o755))
	require.NoError(t, err)

	// Confirm the file is uncompressed
	fileContent, err := os.ReadFile(file)
	require.NoError(t, err)
	err = json.Unmarshal(fileContent, &jsonTestData{})
	require.NoError(t, err)

	var result *jsonTestData
	result, err = LoadJSON[jsonTestData](file)
	require.NoError(t, err)
	require.EqualValues(t, data, result)
}

func TestRoundTripJSONWithGzip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json.gz")
	data := &jsonTestData{A: "yay", B: 3}
	err := WriteJSON(data, ioutil.ToAtomicFile(file, 0o755))
	require.NoError(t, err)

	// Confirm the file isn't raw JSON
	fileContent, err := os.ReadFile(file)
	require.NoError(t, err)
	err = json.Unmarshal(fileContent, &jsonTestData{})
	require.Error(t, err, "should not be able to decode without decompressing")

	var result *jsonTestData
	result, err = LoadJSON[jsonTestData](file)
	require.NoError(t, err)
	require.EqualValues(t, data, result)
}

func TestLoadJSONWithExtraDataAppended(t *testing.T) {
	data := &jsonTestData{A: "yay", B: 3}

	cases := []struct {
		name      string
		extraData func() ([]byte, error)
	}{
		{
			name: "duplicate json object",
			extraData: func() ([]byte, error) {
				return json.Marshal(data)
			},
		},
		{
			name: "duplicate comma-separated json object",
			extraData: func() ([]byte, error) {
				data, err := json.Marshal(data)
				if err != nil {
					return nil, err
				}
				return append([]byte(","), data...), nil
			},
		},
		{
			name: "additional characters",
			extraData: func() ([]byte, error) {
				return []byte("some text"), nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "test.json")
			extraData, err := tc.extraData()
			require.NoError(t, err)

			// Write primary json payload + extra data to the file
			err = WriteJSON(data, ioutil.ToAtomicFile(file, 0o755))
			require.NoError(t, err)
			err = appendDataToFile(file, extraData)
			require.NoError(t, err)

			var result *jsonTestData
			result, err = LoadJSON[jsonTestData](file)
			require.ErrorContains(t, err, "unexpected trailing data")
			require.Nil(t, result)
		})
	}
}

func TestLoadJSONWithTrailingWhitespace(t *testing.T) {
	cases := []struct {
		name      string
		extraData []byte
	}{
		{
			name:      "space",
			extraData: []byte(" "),
		},
		{
			name:      "tab",
			extraData: []byte("\t"),
		},
		{
			name:      "new line",
			extraData: []byte("\n"),
		},
		{
			name:      "multiple chars",
			extraData: []byte(" \t\n"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "test.json")
			data := &jsonTestData{A: "yay", B: 3}

			// Write primary json payload + extra data to the file
			err := WriteJSON(data, ioutil.ToAtomicFile(file, 0o755))
			require.NoError(t, err)
			err = appendDataToFile(file, tc.extraData)
			require.NoError(t, err)

			var result *jsonTestData
			result, err = LoadJSON[jsonTestData](file)
			require.NoError(t, err)
			require.EqualValues(t, data, result)
		})
	}
}

type jsonTestData struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func appendDataToFile(outputPath string, data []byte) error {
	file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
