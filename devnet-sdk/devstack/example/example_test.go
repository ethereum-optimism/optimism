package example

import (
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m)
}

func TestExample1(t *testing.T) {
	preset := presets.NewSimpleInterop(t)
	preset.Log.Info("hello world")
}

func TestExample2(t *testing.T) {
	preset := presets.NewSimpleInterop(t)
	preset.Log.Info("foobar 123")
}
