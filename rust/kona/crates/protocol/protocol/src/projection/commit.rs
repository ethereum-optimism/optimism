//! Domain-parameterised commitment tree and the output and message leaves of the
//! sound private-projection statement. Twin of Go `op-private-interop/projection/commit.go`.

use alloc::vec::Vec;
use alloy_primitives::{B256, keccak256};

use super::replaySentMessageCall;

/// Domain of the per-block private output commitment (`outputsRoot`).
pub const OUTPUTS_DOMAIN: &[u8] = b"optimism.private-outputs.v1\0";
/// Domain of the rendered-message commitment (`messagesRoot`).
pub const MESSAGES_DOMAIN: &[u8] = b"optimism.private-messages.v1\0";

/// Message leaf kind of a rendered message.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum MessageKind {
    /// Export: a `SentMessage` from `L2ToL2CrossDomainMessenger`.
    Init = 0x01,
    /// Import: an `ExecutingMessage` from `CrossL2Inbox`.
    Exec = 0x02,
}

fn node(left: &B256, right: &B256) -> B256 {
    let mut buf = [0u8; 65];
    buf[0] = 1;
    buf[1..33].copy_from_slice(left.as_slice());
    buf[33..].copy_from_slice(right.as_slice());
    keccak256(buf)
}

fn finish(domain: &[u8], n: u64, top: &B256) -> B256 {
    let mut buf = Vec::with_capacity(domain.len() + 40);
    buf.extend_from_slice(domain);
    buf.extend_from_slice(&n.to_be_bytes());
    buf.extend_from_slice(top.as_slice());
    keccak256(buf)
}

/// Count-bound Merkle root over ordered leaves. The last node of an odd level is
/// paired with itself; the empty tree has an all-zero top.
pub fn commitment_root(domain: &[u8], leaves: &[B256]) -> B256 {
    let top = if leaves.is_empty() {
        B256::ZERO
    } else {
        let mut level = leaves.to_vec();
        while level.len() > 1 {
            level = level.chunks(2).map(|p| node(&p[0], p.get(1).unwrap_or(&p[0]))).collect();
        }
        level[0]
    };
    finish(domain, leaves.len() as u64, &top)
}

/// Bottom-up siblings for `leaves[index]`. A node that is the last element of an
/// odd-length level is paired with itself and emits no sibling. `None` if out of range.
pub fn commitment_proof(leaves: &[B256], index: usize) -> Option<Vec<B256>> {
    if index >= leaves.len() {
        return None;
    }
    let mut siblings = Vec::new();
    let mut level = leaves.to_vec();
    let mut idx = index;
    while level.len() > 1 {
        let odd_last = level.len() % 2 == 1 && idx == level.len() - 1;
        if !odd_last {
            siblings.push(level[idx ^ 1]);
        }
        level = level.chunks(2).map(|p| node(&p[0], p.get(1).unwrap_or(&p[0]))).collect();
        idx /= 2;
    }
    Some(siblings)
}

/// Verify an inclusion proof produced by [`commitment_proof`] against a root from
/// [`commitment_root`]. Level lengths are recomputed from `n`; every sibling must be used.
pub fn verify_commitment_proof(
    domain: &[u8],
    n: u64,
    index: u64,
    leaf: B256,
    siblings: &[B256],
    root: B256,
) -> bool {
    if index >= n {
        return false;
    }
    let mut len = n;
    let mut idx = index;
    let mut acc = leaf;
    let mut rest = siblings.iter();
    while len > 1 {
        if len % 2 == 1 && idx == len - 1 {
            acc = node(&acc, &acc);
        } else {
            let Some(sibling) = rest.next() else {
                return false;
            };
            acc = if idx.is_multiple_of(2) { node(&acc, sibling) } else { node(sibling, &acc) };
        }
        idx /= 2;
        len = len.div_ceil(2);
    }
    rest.next().is_none() && finish(domain, n, &acc) == root
}

/// `keccak256(0x00 ‖ u64be(blockNumber) ‖ outputRoot)`.
pub fn output_leaf(block_number: u64, output_root: B256) -> B256 {
    let mut buf = [0u8; 41];
    buf[1..9].copy_from_slice(&block_number.to_be_bytes());
    buf[9..].copy_from_slice(output_root.as_slice());
    keccak256(buf)
}

/// `keccak256(0x00 ‖ u64be(blockNumber) ‖ u32be(renderedIndex) ‖ u8(kind) ‖ messageHash)`.
pub fn message_leaf(
    block_number: u64,
    rendered_index: u32,
    kind: MessageKind,
    message_hash: B256,
) -> B256 {
    let mut buf = [0u8; 46];
    buf[1..9].copy_from_slice(&block_number.to_be_bytes());
    buf[9..13].copy_from_slice(&rendered_index.to_be_bytes());
    buf[13] = kind as u8;
    buf[14..].copy_from_slice(message_hash.as_slice());
    keccak256(buf)
}

/// Interop payload hash of the `SentMessage` log that `replaySentMessage` emits:
/// `keccak256(topic0 ‖ topic1 ‖ topic2 ‖ topic3 ‖ abi.encode(sender, message))`.
pub fn export_message_hash(m: &replaySentMessageCall) -> B256 {
    let (topics, data) = super::render::sent_message_log(m);
    let mut buf = Vec::with_capacity(128 + data.len());
    for t in &topics {
        buf.extend_from_slice(t.as_slice());
    }
    buf.extend_from_slice(&data);
    keccak256(buf)
}

/// `keccak256(identifierWords ‖ payloadHash)`: the 192 argument bytes of
/// `validateMessage(Identifier,bytes32)` calldata (calldata without its selector).
pub fn import_message_hash(args: &[u8; 192]) -> B256 {
    keccak256(args)
}
