package parityreport

// Differences that more than one layer of the parity suite records. The report groups records by
// their text, so every Go layer takes the text from here. The vector layer, in Rust
// (rust/kona/crates/protocol/derive/tests/parity.rs), holds the same texts.
const (
	DivergenceSpanBatchDerivationFailure = "Span batch that fails span-batch derivation: kona drops the rest of its channel, op-node drops only that batch"
	DivergenceSpanBatchFee               = "Span batch fee fields above 128 bits: kona keeps only the low 128 bits, op-node keeps the full value"
	DivergenceBrotliFormatError          = "Brotli channel with a format error: kona discards the batches decoded before the error, op-node keeps them"
)
