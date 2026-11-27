package parityreport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteAndRender(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)
	Write(t, Record{Layer: LayerProof, Group: "TestB", Case: "crafted", Forks: "Holocene", Divergence: "wide fee: kona truncates, op-node keeps it"})
	Write(t, Record{Layer: LayerVectors, Group: "channel reader", Case: "control", Agree: true})
	Write(t, Record{Layer: LayerVectors, Group: "channel reader", Case: "wide fee", Divergence: "wide fee: kona truncates, op-node keeps it"})
	Write(t, Record{Layer: LayerVectors, Group: "deposit log", Case: "padding", Divergence: "padding | kona rejects"})

	records, err := Load(dir)
	require.NoError(t, err)
	require.Len(t, records, 4)

	require.Equal(t, `# Derivation parity: kona vs op-node

| Layer | Group | Cases | Agree | Diverge |
|---|---|---:|---:|---:|
| vectors | channel reader | 2 | 1 | 1 |
| vectors | deposit log | 1 | 0 | 1 |
| proof | TestB | 1 | 0 | 1 |
| **Total** | | **4** | **1** | **3** |

## 2 differences

| # | Difference | Cases | Layers |
|---:|---|---:|---|
| 1 | wide fee: kona truncates, op-node keeps it | 2 | vectors, proof |
| 2 | padding \| kona rejects | 1 | vectors |

## Diverging cases

| # | Layer | Group | Case | Forks |
|---:|---|---|---|---|
| 1 | vectors | channel reader | wide fee |  |
| 1 | proof | TestB | crafted | Holocene |
| 2 | vectors | deposit log | padding |  |
`, Markdown(records))
}

func TestWriteWithoutDirectory(t *testing.T) {
	t.Setenv(EnvDir, "")
	Write(t, Record{Layer: LayerVectors, Group: "g", Case: "c", Agree: true})
}

func TestAllAgree(t *testing.T) {
	require.Contains(t, Markdown([]Record{{Layer: LayerVectors, Group: "g", Case: "c", Agree: true}}),
		"kona and op-node agree on every case.")
}
