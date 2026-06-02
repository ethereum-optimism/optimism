//! Prints Interop activation deposits and upgrade gas; paired with the Go `interop-deposits-dump`
//! command for cross-language diffing.

use alloy_primitives::{TxKind, hex};
use kona_hardforks::{Hardforks, Interop};

fn main() {
    for activate in [false, true] {
        let gas = Hardforks::INTEROP.upgrade_gas_for_activation(activate);
        println!("activate={}", activate);
        println!("gas=0x{:016x}", gas);
        for (i, tx) in Interop::deposits(activate).iter().enumerate() {
            let to = match tx.to {
                TxKind::Call(addr) => format!("0x{}", hex::encode(addr.as_slice())),
                TxKind::Create => "create".to_string(),
            };
            println!("--- tx {} ---", i);
            println!("source_hash=0x{}", hex::encode(tx.source_hash.as_slice()));
            println!("from=0x{}", hex::encode(tx.from.as_slice()));
            println!("to={}", to);
            println!("mint=0x{:032x}", tx.mint);
            println!("value=0x{:064x}", tx.value);
            println!("gas_limit=0x{:016x}", tx.gas_limit);
            println!("is_system_tx={}", tx.is_system_transaction);
            println!("data=0x{}", hex::encode(&tx.input));
        }
    }
}
