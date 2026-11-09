//! Test-only SP1 guest: one OP deposit that runs a TSTORE loop until it is out of gas, executed
//! with the same revm / op-revm / foldhash code that the super-range guest uses for L2 blocks.
//!
//! Input: the guest hash seed, then `(gas_limit, top_limb, step_limb)`. The contract stores the
//! keys `(top_limb << 192) + (i << (64 * step_limb))` for i = 0, 1, ... Output: the gas used.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use op_revm::{DefaultOp, OpBuilder, OpSpecId, OpTransaction};
use revm::{
    ExecuteEvm,
    bytecode::Bytecode,
    context::{CfgEnv, Context, TxEnv},
    database::InMemoryDB,
    primitives::{Address, B256, Bytes, TxKind},
    state::AccountInfo,
};

const TARGET: Address = Address::new([0xAA; 20]);
const CALLER: Address = Address::new([0x11; 20]);
const TSTORES_PER_LOOP: usize = 64;

/// `PUSH32 base; JUMPDEST; 64 * (DUP1 DUP1 TSTORE PUSH32 step ADD); PUSH1 0x21 JUMP` with
/// `base = top_limb << 192` and `step = 1 << (64 * step_limb)`. Each store costs 112 gas; the loop
/// runs until out of gas.
fn runtime(top_limb: u64, step_limb: u64) -> Vec<u8> {
    assert!(step_limb < 4, "step_limb {step_limb} is not a 64-bit word index");
    let mut base = [0u8; 32];
    base[..8].copy_from_slice(&top_limb.to_be_bytes());
    let mut step = [0u8; 32];
    step[31 - 8 * step_limb as usize] = 1;

    let mut code = vec![0x7F];
    code.extend_from_slice(&base);
    code.push(0x5B);
    for _ in 0..TSTORES_PER_LOOP {
        code.extend_from_slice(&[0x80, 0x80, 0x5D, 0x7F]);
        code.extend_from_slice(&step);
        code.push(0x01);
    }
    code.extend_from_slice(&[0x60, 0x21, 0x56]);
    code
}

/// Entrypoint to the hash-flooding check program.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);
    foldhash::install_global_seed(sp1_zkvm::io::read::<u64>());

    let (gas_limit, top_limb, step_limb) = sp1_zkvm::io::read::<(u64, u64, u64)>();

    let mut db = InMemoryDB::default();
    db.insert_account_info(
        TARGET,
        AccountInfo::default()
            .with_code(Bytecode::new_raw(Bytes::from(runtime(top_limb, step_limb)))),
    );
    // A zero-value deposit, as an ordinary L1 account submits through the OptimismPortal.
    let tx = OpTransaction::builder()
        .base(TxEnv::builder().caller(CALLER).kind(TxKind::Call(TARGET)).gas_limit(gas_limit))
        .source_hash(B256::with_last_byte(1))
        .build_fill();
    let mut evm =
        Context::op().with_cfg(CfgEnv::new_with_spec(OpSpecId::KARST)).with_db(db).build_op();

    let result = evm.transact(tx).expect("deposit executes");
    assert!(!result.result.is_success(), "the loop must end out of gas");
    sp1_zkvm::io::commit(&result.result.tx_gas_used());
}
