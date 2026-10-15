//! EVM bytecode of the contracts the live tests deploy, and the storage slots they write.

use alloy_primitives::{Address, B256, Bytes, U256};

/// Storage slot every call to the block-hash recorder sets to one.
pub(super) const CALLED_SLOT: B256 = B256::ZERO;

/// Storage slot a call to the block-hash recorder with calldata sets to `BLOCKHASH(calldata)`.
pub(super) const BLOCK_HASH_SLOT: B256 = B256::with_last_byte(1);

/// Runtime code of the block-hash recorder: `slot 0 = 1`, then
/// `slot 1 = BLOCKHASH(calldata[0..32])` unless the call has no calldata.
///
/// ```text
/// PUSH1 1 PUSH1 0 SSTORE
/// CALLDATASIZE ISZERO PUSH1 0x11 JUMPI
/// PUSH1 0 CALLDATALOAD BLOCKHASH PUSH1 1 SSTORE
/// JUMPDEST STOP
/// ```
pub(super) const BLOCK_HASH_RECORDER_RUNTIME: [u8; 19] = [
    0x60, 0x01, 0x60, 0x00, 0x55, 0x36, 0x15, 0x60, 0x11, 0x57, 0x60, 0x00, 0x35, 0x40, 0x60, 0x01,
    0x55, 0x5b, 0x00,
];

/// Number of storage slots the slot writer takes from calldata.
pub(super) const SLOT_WRITER_SLOTS: usize = 8;

/// Runtime code of the slot writer: a call with calldata stores its first eight 32-byte words
/// into `slots`, in order; a call without calldata self-destructs to `beneficiary`.
///
/// ```text
/// CALLDATASIZE ISZERO PUSH1 <selfdestruct> JUMPI
/// PUSH1 0x00 CALLDATALOAD PUSH1 slots[0] SSTORE
/// ...
/// PUSH1 0xe0 CALLDATALOAD PUSH1 slots[7] SSTORE
/// STOP
/// JUMPDEST PUSH20 beneficiary SELFDESTRUCT
/// ```
pub(super) fn slot_writer_runtime(beneficiary: Address, slots: [u8; SLOT_WRITER_SLOTS]) -> Vec<u8> {
    let mut store_slots = Vec::new();
    for (word, slot) in slots.into_iter().enumerate() {
        store_slots.extend_from_slice(&[0x60, word as u8 * 32, 0x35, 0x60, slot, 0x55]);
    }
    store_slots.push(0x00);
    let dispatch_len = 5;
    let selfdestruct_at = u8::try_from(dispatch_len + store_slots.len()).expect("short code");
    let mut code = vec![0x36, 0x15, 0x60, selfdestruct_at, 0x57];
    code.extend_from_slice(&store_slots);
    code.push(0x5b);
    code.push(0x73);
    code.extend_from_slice(beneficiary.as_slice());
    code.push(0xff);
    code
}

/// Number of storage slots [`create_and_selfdestruct_initcode`] writes before self-destructing.
pub(super) const CREATE_AND_SELFDESTRUCT_SLOTS: u8 = 4;

/// Initcode that stores seven into four slots and then self-destructs to `beneficiary` within
/// the creating transaction, which under EIP-6780 removes the account with its storage.
///
/// ```text
/// PUSH1 7 PUSH1 0 SSTORE
/// ...
/// PUSH1 7 PUSH1 3 SSTORE
/// PUSH20 beneficiary SELFDESTRUCT
/// ```
pub(super) fn create_and_selfdestruct_initcode(beneficiary: Address) -> Bytes {
    let mut code = Vec::new();
    for slot in 0..CREATE_AND_SELFDESTRUCT_SLOTS {
        code.extend_from_slice(&[0x60, 0x07, 0x60, slot, 0x55]);
    }
    code.push(0x73);
    code.extend_from_slice(beneficiary.as_slice());
    code.push(0xff);
    code.into()
}

/// Initcode deploying `runtime`, which follows the twelve initcode bytes:
/// `PUSH1 len PUSH1 12 PUSH1 0 CODECOPY PUSH1 len PUSH1 0 RETURN`.
pub(super) fn initcode_returning(runtime: &[u8]) -> Bytes {
    let runtime_len = u8::try_from(runtime.len()).expect("runtime fits one PUSH1");
    let mut initcode =
        vec![0x60, runtime_len, 0x60, 0x0c, 0x60, 0x00, 0x39, 0x60, runtime_len, 0x60, 0x00, 0xf3];
    initcode.extend_from_slice(runtime);
    initcode.into()
}

/// Calldata for the slot writer: the value of each of its slots, in the order of its `slots`.
pub(super) fn slot_writer_calldata(values: [u64; SLOT_WRITER_SLOTS]) -> Bytes {
    values
        .iter()
        .flat_map(|value| U256::from(*value).to_be_bytes::<32>())
        .collect::<Vec<u8>>()
        .into()
}

/// Calldata for the block-hash recorder: the number of the block to read.
pub(super) fn block_number_calldata(number: u64) -> Bytes {
    Bytes::copy_from_slice(&U256::from(number).to_be_bytes::<32>())
}
