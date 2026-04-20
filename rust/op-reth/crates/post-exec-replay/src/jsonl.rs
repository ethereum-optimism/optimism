use crate::types::{
    PostExecReplayBlock, PostExecReplayMismatch, PostExecReplayRunConfig, PostExecReplaySummary,
    PostExecReplayTx,
};
use serde::Serialize;
use std::io::{self, Write};

/// JSONL record for replay output.
#[derive(Debug, Serialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum PostExecReplayJsonlRecord<'a> {
    /// Run-level config.
    RunConfig(&'a PostExecReplayRunConfig),
    /// Per-tx row.
    Tx(&'a PostExecReplayTx),
    /// Per-block row.
    Block(&'a PostExecReplayBlock),
    /// Mismatch row.
    Mismatch(&'a PostExecReplayMismatch),
    /// Summary row.
    Summary(&'a PostExecReplaySummary),
}

/// Write one JSONL record.
pub fn write_jsonl_record(
    mut writer: impl Write,
    record: &PostExecReplayJsonlRecord<'_>,
) -> io::Result<()> {
    serde_json::to_writer(&mut writer, record)?;
    writer.write_all(b"\n")
}
