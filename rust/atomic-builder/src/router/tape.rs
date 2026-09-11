//! Scratch storage for executing real, self-only tape getters.
use super::{Chain, Error, Tape, abi::Identifier};
use crate::{Paused, Reply};
use revm::{
    Database,
    context_interface::{ContextTr, JournalTr},
    primitives::{U256, keccak256},
};

pub(super) fn quote<DB: Database + Clone>(
    paused: &Paused<DB>,
    chain: &Chain<DB>,
    tape: &Tape,
) -> Result<Reply, Error<DB>>
where
    DB::Error: Clone,
{
    let mut context = chain.context.clone();
    let spec = (*context.cfg().spec()).into();
    context.journal_mut().set_spec_id(spec);
    context
        .journal_mut()
        .load_account_with_code(chain.router)
        .map_err(|e| Error::Core(crate::Error::Evm(e.into())))?;
    // Layout is asserted against the compiled router artifact in integration tests.
    let mut slots = vec![
        (U256::from(8), U256::from(tape.witnesses.len())),
        (U256::from(9), U256::from(tape.calls.len())),
    ];
    let witnesses = U256::from_be_bytes(keccak256(U256::from(8).to_be_bytes::<32>()).0);
    for (i, witness) in tape.witnesses.iter().enumerate() {
        let base = witnesses.wrapping_add(U256::from(i * 7));
        identifier(&mut slots, base, &witness.identifier);
        slots.push((base + U256::from(5), U256::from(witness.success)));
        bytes(&mut slots, base + U256::from(6), &witness.returnData);
    }
    let calls = U256::from_be_bytes(keccak256(U256::from(9).to_be_bytes::<32>()).0);
    for (i, call) in tape.calls.iter().enumerate() {
        let base = calls.wrapping_add(U256::from(i * 9));
        identifier(&mut slots, base, &call.identifier);
        slots.push((base + U256::from(5), call.sequence));
        slots.push((base + U256::from(6), U256::from_be_slice(call.sender.as_slice())));
        slots.push((base + U256::from(7), U256::from_be_slice(call.target.as_slice())));
        bytes(&mut slots, base + U256::from(8), &call.data);
    }
    let mut callbacks = std::collections::BTreeMap::<U256, Vec<_>>::new();
    for item in &tape.callbacks {
        callbacks.entry(item.waitingSequence).or_default().push(&item.call);
    }
    for (sequence, calls) in callbacks {
        let root = U256::from_be_bytes(
            keccak256([sequence.to_be_bytes::<32>(), U256::from(15).to_be_bytes::<32>()].concat())
                .0,
        );
        slots.push((root, U256::from(calls.len())));
        let start = U256::from_be_bytes(keccak256(root.to_be_bytes::<32>()).0);
        for (i, call) in calls.iter().enumerate() {
            let base = start.wrapping_add(U256::from(i * 9));
            identifier(&mut slots, base, &call.identifier);
            slots.push((base + U256::from(5), call.sequence));
            slots.push((base + U256::from(6), U256::from_be_slice(call.sender.as_slice())));
            slots.push((base + U256::from(7), U256::from_be_slice(call.target.as_slice())));
            bytes(&mut slots, base + U256::from(8), &call.data);
        }
    }
    // callbackFailure and nested are packed in slot 16. Scratch getters observe the
    // preloaded callback status, never the application's pending state.
    slots.push((
        U256::from(16),
        U256::from(256 + u16::from(tape.callbacks.iter().any(|c| !c.success))),
    ));
    slots.push((U256::from(18), U256::from(tape.callbacks.len())));
    identifier(&mut slots, U256::from(10), &tape.completion);
    for (key, value) in slots {
        context
            .journal_mut()
            .sstore(chain.router, key, value)
            .map_err(|e| Error::Core(crate::Error::Evm(e.into())))?;
    }
    paused.quote(context).map_err(Error::Core)
}
fn identifier(slots: &mut Vec<(U256, U256)>, base: U256, id: &Identifier) {
    for (i, word) in [
        U256::from_be_slice(id.origin.as_slice()),
        id.blockNumber,
        id.logIndex,
        id.timestamp,
        id.chainId,
    ]
    .into_iter()
    .enumerate()
    {
        slots.push((base + U256::from(i), word));
    }
}
fn bytes(slots: &mut Vec<(U256, U256)>, slot: U256, data: &[u8]) {
    if data.len() < 32 {
        let mut value = [0; 32];
        value[..data.len()].copy_from_slice(data);
        value[31] = (data.len() * 2) as u8;
        slots.push((slot, U256::from_be_bytes(value)));
    } else {
        slots.push((slot, U256::from(data.len() * 2 + 1)));
        let base = U256::from_be_bytes(keccak256(slot.to_be_bytes::<32>()).0);
        for (i, chunk) in data.chunks(32).enumerate() {
            let mut value = [0; 32];
            value[..chunk.len()].copy_from_slice(chunk);
            slots.push((base.wrapping_add(U256::from(i)), U256::from_be_bytes(value)));
        }
    }
}
