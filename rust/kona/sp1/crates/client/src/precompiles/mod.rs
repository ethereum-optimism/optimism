//! [`PrecompileProvider`] for FPVM-accelerated OP Stack precompiles.

use alloc::{format, string::String};
use alloy_primitives::Address;
use op_revm::{OpSpecId, precompiles::OpPrecompiles};
#[cfg(target_os = "zkvm")]
use revm::precompile::PrecompileId;
use revm::{
    context::{Cfg, ContextTr, LocalContextTr},
    context_interface::JournalTr,
    handler::{EthPrecompiles, PrecompileProvider, precompile_output_to_interpreter_result},
    interpreter::{CallInput, CallInputs, InterpreterResult},
    precompile::PrecompileError,
    primitives::AddressSet,
};

mod custom;
pub use custom::CustomCrypto;

mod factory;
pub use factory::ZkvmOpEvmFactory;

/// Tracker names for accelerated precompiles.
pub mod cycle_tracker {
    /// Individual tracker names without the prefix.
    pub mod names {
        /// BN254 add tracker name.
        pub const BN_ADD: &str = "bn-add";
        /// BN254 mul tracker name.
        pub const BN_MUL: &str = "bn-mul";
        /// BN254 pairing tracker name.
        pub const BN_PAIR: &str = "bn-pair";
        /// ECRECOVER tracker name.
        pub const EC_RECOVER: &str = "ec-recover";
        /// P256 verify tracker name.
        pub const P256_VERIFY: &str = "p256-verify";
        /// KZG eval tracker name.
        pub const KZG_EVAL: &str = "kzg-eval";
    }

    /// Full cycle tracker keys with the `precompile-` prefix.
    pub mod keys {
        /// BN254 add cycle-tracker key.
        pub const BN_ADD: &str = "precompile-bn-add";
        /// BN254 mul cycle-tracker key.
        pub const BN_MUL: &str = "precompile-bn-mul";
        /// BN254 pairing cycle-tracker key.
        pub const BN_PAIR: &str = "precompile-bn-pair";
        /// ECRECOVER cycle-tracker key.
        pub const EC_RECOVER: &str = "precompile-ec-recover";
        /// P256 verify cycle-tracker key.
        pub const P256_VERIFY: &str = "precompile-p256-verify";
        /// KZG eval cycle-tracker key.
        pub const KZG_EVAL: &str = "precompile-kzg-eval";
    }
}

/// Get the cycle tracker name for a precompile by its ID.
/// Returns None if the precompile is not accelerated/tracked.
#[cfg(target_os = "zkvm")]
#[inline]
fn get_precompile_tracker_name(id: &PrecompileId) -> Option<&'static str> {
    match id {
        PrecompileId::Bn254Add => Some(cycle_tracker::names::BN_ADD),
        PrecompileId::Bn254Mul => Some(cycle_tracker::names::BN_MUL),
        PrecompileId::Bn254Pairing => Some(cycle_tracker::names::BN_PAIR),
        PrecompileId::EcRec => Some(cycle_tracker::names::EC_RECOVER),
        PrecompileId::P256Verify => Some(cycle_tracker::names::P256_VERIFY),
        PrecompileId::KzgPointEvaluation => Some(cycle_tracker::names::KZG_EVAL),
        _ => None,
    }
}

/// The ZKVM-cycle-tracking precompiles.
#[derive(Debug)]
pub struct OpZkvmPrecompiles {
    /// The default [`EthPrecompiles`] provider.
    inner: EthPrecompiles,
    /// The [`OpSpecId`] of the precompiles.
    spec: OpSpecId,
}

impl OpZkvmPrecompiles {
    /// Create a new precompile provider with the given [`OpSpecId`].
    #[inline]
    pub fn new_with_spec(spec: OpSpecId) -> Self {
        let precompiles = OpPrecompiles::new_with_spec(spec).precompiles();

        Self { inner: EthPrecompiles { precompiles, spec: spec.into_eth_spec() }, spec }
    }
}

impl<CTX> PrecompileProvider<CTX> for OpZkvmPrecompiles
where
    CTX: ContextTr<Cfg: Cfg<Spec = OpSpecId>>,
{
    type Output = InterpreterResult;

    #[inline]
    fn set_spec(&mut self, spec: <CTX::Cfg as Cfg>::Spec) -> bool {
        if spec == self.spec {
            return false;
        }
        *self = Self::new_with_spec(spec);
        true
    }

    #[inline]
    fn run(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
    ) -> Result<Option<Self::Output>, String> {
        let Some(precompile) = self.inner.precompiles.get(&inputs.bytecode_address) else {
            return Ok(None);
        };

        #[cfg(target_os = "zkvm")]
        let tracker_name = get_precompile_tracker_name(precompile.id());

        let output = {
            let shared_buffer;
            let input_bytes = match &inputs.input {
                CallInput::Bytes(bytes) => bytes.as_ref(),
                CallInput::SharedBuffer(range) => {
                    shared_buffer = context.local().shared_memory_buffer_slice(range.clone());
                    shared_buffer.as_deref().unwrap_or(&[])
                }
            };

            #[cfg(target_os = "zkvm")]
            if let Some(name) = tracker_name {
                println!("cycle-tracker-report-start: precompile-{}", name);
            }

            let output = precompile
                .execute(input_bytes, inputs.gas_limit, inputs.reservoir)
                .map_err(|e| match e {
                    PrecompileError::Fatal(e) => e,
                    PrecompileError::FatalAny(e) => format!("{e:?}"),
                })?;

            #[cfg(target_os = "zkvm")]
            if let Some(name) = tracker_name {
                println!("cycle-tracker-report-end: precompile-{}", name);
            }

            output
        };

        if let Some(halt_reason) = output.halt_reason() &&
            !halt_reason.is_oog() &&
            context.journal().depth() == 1
        {
            context.local_mut().set_precompile_error_context(halt_reason.to_string());
        }

        let result = precompile_output_to_interpreter_result(output, inputs.gas_limit);
        Ok(Some(result))
    }

    #[inline]
    fn warm_addresses(&self) -> &AddressSet {
        self.inner.warm_addresses()
    }

    #[inline]
    fn contains(&self, address: &Address) -> bool {
        self.inner.contains(address)
    }
}

#[cfg(test)]
mod tests {
    use alloc::vec;

    use op_revm::{DefaultOp, precompiles::bn254_pair};
    use revm::{
        Context,
        bytecode::Bytecode,
        context::CfgEnv,
        handler::PrecompileProvider,
        interpreter::{CallInput, CallInputs, CallScheme, CallValue, InstructionResult},
        precompile::{PrecompileHalt, PrecompileStatus, bn254},
        primitives::{B256, U256},
    };

    use super::*;

    #[test]
    fn karst_bn254_pairing_uses_fork_input_cap() {
        let precompiles = OpZkvmPrecompiles::new_with_spec(OpSpecId::KARST);
        let bn254_pair_precompile =
            precompiles.inner.precompiles.get(&bn254::pair::ADDRESS).unwrap();

        let bad_input_len = bn254_pair::KARST_MAX_INPUT_SIZE + 1;
        assert!(bad_input_len < bn254_pair::JOVIAN_MAX_INPUT_SIZE);
        let input = vec![0u8; bad_input_len];

        let res = bn254_pair_precompile.execute(&input, u64::MAX, 0).unwrap();
        assert!(matches!(res.status, PrecompileStatus::Halt(PrecompileHalt::Bn254PairLength)));
    }

    #[test]
    fn karst_bn254_pairing_wrapper_preserves_halt_gas_semantics() {
        let bad_input_len = bn254_pair::KARST_MAX_INPUT_SIZE + 1;
        let gas_limit = 1_000_000;
        let mut context = Context::op().with_cfg(CfgEnv::new_with_spec(OpSpecId::KARST));
        let mut precompiles = OpZkvmPrecompiles::new_with_spec(OpSpecId::KARST);

        let result = precompiles
            .run(
                &mut context,
                &CallInputs {
                    input: CallInput::Bytes(vec![0u8; bad_input_len].into()),
                    return_memory_offset: 0..0,
                    gas_limit,
                    reservoir: 0,
                    bytecode_address: bn254::pair::ADDRESS,
                    known_bytecode: (B256::ZERO, Bytecode::default()),
                    target_address: bn254::pair::ADDRESS,
                    caller: Address::ZERO,
                    value: CallValue::Transfer(U256::ZERO),
                    scheme: CallScheme::Call,
                    is_static: false,
                    charged_new_account_state_gas: false,
                },
            )
            .unwrap()
            .expect("bn254 pairing address should be a precompile");

        assert_eq!(result.result, InstructionResult::PrecompileError);
        assert_eq!(result.gas.remaining(), 0, "non-OOG precompile halt must spend all gas");
        assert_eq!(result.gas.total_gas_spent(), gas_limit);
        assert!(result.output.is_empty());
    }
}
