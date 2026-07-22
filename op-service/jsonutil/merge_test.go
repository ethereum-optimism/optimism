package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeJSON(t *testing.T) {
	type testStruct struct {
		A string `json:"a"`
		B int    `json:"b"`
		C bool   `json:"c"`
	}

	out, err := MergeJSON(
		testStruct{
			"hello",
			42,
			true,
		},
		map[string]any{
			"a": "world",
			"c": false,
		},
		map[string]any{
			"d": "shouldn't show up",
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, testStruct{
		"world",
		42,
		false,
	}, out)
}

func TestMergeJSONOverridePrecedence(t *testing.T) {
	type testStruct struct {
		A string `json:"a"`
		B int    `json:"b"`
		C bool   `json:"c"`
	}

	out, err := MergeJSON(
		testStruct{
			A: "initial",
			B: 1,
			C: false,
		},
		map[string]any{
			"a": "first",
			"b": 2,
		},
		nil,
		map[string]any{
			"a": "last",
			"c": true,
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, testStruct{
		A: "last",
		B: 2,
		C: true,
	}, out)
}

func TestMergeJSONErrors(t *testing.T) {
	t.Run("input marshal error", func(t *testing.T) {
		type testStruct struct {
			Bad chan int `json:"bad"`
		}

		out, err := MergeJSON(testStruct{Bad: make(chan int)})
		require.Error(t, err)
		require.Zero(t, out)
	})

	t.Run("input must marshal to object", func(t *testing.T) {
		out, err := MergeJSON("not an object")
		require.Error(t, err)
		require.Empty(t, out)
	})

	t.Run("override marshal error", func(t *testing.T) {
		type testStruct struct {
			A string `json:"a"`
		}

		out, err := MergeJSON(testStruct{A: "ok"}, map[string]any{
			"a": func() {},
		})
		require.Error(t, err)
		require.Zero(t, out)
	})

	t.Run("override decode error", func(t *testing.T) {
		type testStruct struct {
			B int `json:"b"`
		}

		out, err := MergeJSON(testStruct{B: 1}, map[string]any{
			"b": "not an int",
		})
		require.Error(t, err)
		require.Zero(t, out)
	})
}
