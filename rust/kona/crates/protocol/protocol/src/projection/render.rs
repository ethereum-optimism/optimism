//! Rust twin of the Go rendering primitive (`op-private-interop/render/render.go`):
//! `EmitterSet.Renders` for the zero-value emitter set, `RenderedLogs`, and the replay
//! calldata of `actionFor` + `ReplayTx`. There are no extra emitters in v1.

use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, Log, U256, keccak256};
use alloy_sol_types::{SolCall, SolEvent, SolValue};

use super::{
    BRIDGE, ExecutingMessage, INBOX, MAX_MESSAGE, MESSENGER, MessageKind, ProjectionError,
    SentMessage, message_leaf, replaySentMessageCall, validateMessageCall,
};

/// One private log that survives the emitter-set filter, with both positions.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RenderedLog<'a> {
    /// The caller's log, never modified.
    pub log: &'a Log,
    /// Block-level position among all of the private block's logs.
    pub private_log_index: u32,
    /// Rank among the rendered logs, `0..k-1`: the message's canonical log index.
    pub rendered_log_index: u32,
}

/// THE predicate: `(L2ToL2CrossDomainMessenger, SentMessage)` or
/// `(CrossL2Inbox, ExecutingMessage)`, matched on address and topic0 only.
pub fn renders(log: &Log) -> bool {
    let topic0 = log.topics().first();
    match log.address {
        MESSENGER => topic0 == Some(&SentMessage::SIGNATURE_HASH),
        INBOX => topic0 == Some(&ExecutingMessage::SIGNATURE_HASH),
        _ => false,
    }
}

/// The block's complete log sequence (in block order) restricted to [`renders`],
/// in original order, re-indexed from zero.
pub fn rendered_logs<'a>(
    logs: impl Iterator<Item = &'a Log>,
) -> Result<Vec<RenderedLog<'a>>, ProjectionError> {
    let mut out = Vec::new();
    for (i, log) in logs.enumerate() {
        if !renders(log) {
            continue;
        }
        let private_log_index =
            u32::try_from(i).map_err(|_| ProjectionError("private log index overflow"))?;
        let rendered_log_index =
            u32::try_from(out.len()).map_err(|_| ProjectionError("rendered log index overflow"))?;
        out.push(RenderedLog { log, private_log_index, rendered_log_index });
    }
    Ok(out)
}

/// The `SentMessage` topics and data that `replaySentMessage` emits for `m`.
pub fn sent_message_log(m: &replaySentMessageCall) -> ([B256; 4], Bytes) {
    let topics = [
        SentMessage::SIGNATURE_HASH,
        B256::from(m.destination),
        m.target.into_word(),
        B256::from(m.nonce),
    ];
    (topics, (m.sender, m.message.clone()).abi_encode_params().into())
}

/// Decode a rendered messenger log strictly: four topics, an address-shaped target
/// topic and a data section that re-encodes to exactly its bytes.
fn decode_sent_message(log: &Log) -> Result<replaySentMessageCall, ProjectionError> {
    let topics = log.topics();
    if topics.len() != 4 || topics[0] != SentMessage::SIGNATURE_HASH {
        return Err(ProjectionError("unrenderable SentMessage topics"));
    }
    let (sender, message) = <(Address, Bytes)>::abi_decode_params_validate(&log.data.data)
        .map_err(|_| ProjectionError("unrenderable SentMessage data"))?;
    let target = Address::from_word(topics[2]);
    let m = replaySentMessageCall {
        destination: U256::from_be_bytes(topics[1].0),
        nonce: U256::from_be_bytes(topics[3].0),
        sender,
        target,
        message,
    };
    let (expected_topics, expected_data) = sent_message_log(&m);
    if expected_topics.as_slice() != topics || expected_data != log.data.data {
        return Err(ProjectionError("noncanonical SentMessage"));
    }
    Ok(m)
}

/// Strict `ExecutingMessage` identifier words, as Go `messages.Message.DecodeEvent`:
/// two topics, 160 data bytes, address/u64/u32/u64 left padding all zero.
fn executing_message_args(log: &Log) -> Result<[u8; 192], ProjectionError> {
    let topics = log.topics();
    let data = log.data.data.as_ref();
    if topics.len() != 2 || topics[0] != ExecutingMessage::SIGNATURE_HASH || data.len() != 160 {
        return Err(ProjectionError("unrenderable ExecutingMessage"));
    }
    let zero = |r: core::ops::Range<usize>| data[r].iter().all(|b| *b == 0);
    if !zero(0..12) || !zero(32..56) || !zero(64..92) || !zero(96..120) {
        return Err(ProjectionError("noncanonical ExecutingMessage identifier"));
    }
    let mut args = [0u8; 192];
    args[..160].copy_from_slice(data);
    args[160..].copy_from_slice(topics[1].as_slice());
    Ok(args)
}

/// `(to, calldata)` of the one replay transaction that renders `rl`, including the
/// fatal cases of Go `actionFor`: bridge sender or target, oversize, malformed.
pub fn replay_calldata(rl: &RenderedLog<'_>) -> Result<(Address, Bytes), ProjectionError> {
    match rl.log.address {
        MESSENGER => {
            let m = decode_sent_message(rl.log)?;
            if m.sender == BRIDGE || m.target == BRIDGE {
                return Err(ProjectionError("unrenderable SuperchainETHBridge message"));
            }
            if m.message.len() > MAX_MESSAGE {
                return Err(ProjectionError("unrenderable oversize message"));
            }
            Ok((MESSENGER, m.abi_encode().into()))
        }
        INBOX => {
            let args = executing_message_args(rl.log)?;
            let mut data = validateMessageCall::SELECTOR.to_vec();
            data.extend_from_slice(&args);
            Ok((INBOX, data.into()))
        }
        _ => Err(ProjectionError("unrenderable emitter")),
    }
}

/// Message kind and hash of a rendered private log (§C.3.7 relation side). Rejects
/// exactly what [`replay_calldata`] rejects.
pub fn rendered_message(rl: &RenderedLog<'_>) -> Result<(MessageKind, B256), ProjectionError> {
    replay_calldata(rl)?;
    match rl.log.address {
        MESSENGER => {
            let mut buf = Vec::with_capacity(128 + rl.log.data.data.len());
            for t in rl.log.topics() {
                buf.extend_from_slice(t.as_slice());
            }
            buf.extend_from_slice(&rl.log.data.data);
            Ok((MessageKind::Init, keccak256(buf)))
        }
        _ => Ok((MessageKind::Exec, keccak256(executing_message_args(rl.log)?))),
    }
}

/// Message leaves of one executed private block, from its rendered logs. Twin of Go
/// `render.MessageLeaves`.
pub fn message_leaves(
    block_number: u64,
    logs: &[RenderedLog<'_>],
) -> Result<Vec<B256>, ProjectionError> {
    logs.iter()
        .map(|rl| {
            let (kind, hash) = rendered_message(rl)?;
            Ok(message_leaf(block_number, rl.rendered_log_index, kind, hash))
        })
        .collect()
}
