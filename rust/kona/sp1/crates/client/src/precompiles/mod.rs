//! [`PrecompileProvider`] for FPVM-accelerated OP Stack precompiles.

use alloc::{boxed::Box, format, string::String, vec::Vec};
use alloy_primitives::{Address, Bytes};
use op_revm::{
    OpSpecId,
    precompiles::{fjord, granite, isthmus, jovian, karst},
};
use revm::{
    context::{Cfg, ContextTr},
    handler::{EthPrecompiles, PrecompileProvider},
    interpreter::{CallInputs, Gas, InstructionResult, InterpreterResult},
    precompile::{
        Precompile as PrecompileWithAddress, PrecompileError, PrecompileOutput, Precompiles, bn254,
        kzg_point_evaluation, secp256k1, secp256r1,
    },
    primitives::{AddressSet, hardfork::SpecId},
};

mod custom;
pub use custom::CustomCrypto;

mod factory;
pub use factory::ZkvmOpEvmFactory;

/// Get the ZKVM-accelerated precompiles.
fn get_precompiles() -> Vec<PrecompileWithAddress> {
    vec![
        bn254::add::ISTANBUL,
        bn254::mul::ISTANBUL,
        bn254::pair::ISTANBUL,
        secp256k1::ECRECOVER,
        secp256r1::P256VERIFY,
        kzg_point_evaluation::POINT_EVALUATION,
    ]
}

/// Get the cycle tracker name for a precompile address.
/// Returns None if the address is not a tracked precompile.
#[cfg(target_os = "zkvm")]
#[inline]
fn get_precompile_tracker_name(address: &Address) -> Option<&'static str> {
    // Compare against actual precompile constants
    if *address == *bn254::add::ISTANBUL.address() {
        Some("bn-add")
    } else if *address == *bn254::mul::ISTANBUL.address() {
        Some("bn-mul")
    } else if *address == *bn254::pair::ISTANBUL.address() {
        Some("bn-pair")
    } else if *address == *secp256k1::ECRECOVER.address() {
        Some("ec-recover")
    } else if *address == *secp256r1::P256VERIFY.address() {
        Some("p256-verify")
    } else if *address == *kzg_point_evaluation::POINT_EVALUATION.address() {
        Some("kzg-eval")
    } else {
        None
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
        let precompiles = match spec {
            spec @ (OpSpecId::BEDROCK |
            OpSpecId::REGOLITH |
            OpSpecId::CANYON |
            OpSpecId::ECOTONE) => Precompiles::new(spec.into_eth_spec().into()),
            OpSpecId::FJORD => fjord(),
            OpSpecId::GRANITE | OpSpecId::HOLOCENE => granite(),
            OpSpecId::ISTHMUS => isthmus(),
            OpSpecId::JOVIAN => jovian(),
            OpSpecId::KARST | OpSpecId::INTEROP => karst(),
        };
        let mut precompiles_owned = precompiles.clone();
        precompiles_owned.extend(get_precompiles());
        let precompiles = Box::leak(Box::new(precompiles_owned));

        Self { inner: EthPrecompiles { precompiles, spec: SpecId::default() }, spec }
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
        let mut result = InterpreterResult {
            result: InstructionResult::Return,
            gas: Gas::new(inputs.gas_limit),
            output: Bytes::new(),
        };

        use revm::context::LocalContextTr;
        let input = match &inputs.input {
            revm::interpreter::CallInput::Bytes(bytes) => bytes.clone(),
            revm::interpreter::CallInput::SharedBuffer(range) => context
                .local()
                .shared_memory_buffer_slice(range.clone())
                .map(|b| Bytes::from(b.to_vec()))
                .unwrap_or_default(),
        };

        // Priority:
        // 1. If the precompile has an accelerated version, use that.
        // 2. If the precompile is not accelerated, use the default version.
        // 3. If the precompile is not found, return None.
        let output = if let Some(precompile) = self.inner.precompiles.get(&inputs.bytecode_address)
        {
            // Track cycles for accelerated precompiles
            #[cfg(target_os = "zkvm")]
            let tracker_name = get_precompile_tracker_name(&inputs.bytecode_address);

            #[cfg(target_os = "zkvm")]
            if let Some(name) = tracker_name {
                println!("cycle-tracker-report-start: precompile-{}", name);
            }

            let result = match precompile.execute(&input, inputs.gas_limit, inputs.reservoir) {
                Ok(output) => output,
                Err(PrecompileError::Fatal(e)) => return Err(e),
                Err(PrecompileError::FatalAny(e)) => return Err(format!("{e:?}")),
            };

            #[cfg(target_os = "zkvm")]
            if let Some(name) = tracker_name {
                println!("cycle-tracker-report-end: precompile-{}", name);
            }

            result
        } else {
            return Ok(None);
        };

        let output: PrecompileOutput = output;
        if output.is_halt() {
            result.result = if output.halt_reason().is_some_and(|r| r.is_oog()) {
                InstructionResult::PrecompileOOG
            } else {
                InstructionResult::PrecompileError
            };
        } else {
            let underflow = result.gas.record_regular_cost(output.gas_used);
            assert!(underflow, "Gas underflow is not possible");
            result.result = InstructionResult::Return;
            result.output = output.bytes;
        }

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
