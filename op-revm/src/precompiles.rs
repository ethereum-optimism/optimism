//! Contains Optimism specific precompiles.
use crate::OpSpecId;
use revm::{
    context::Cfg,
    context_interface::ContextTr,
    handler::{EthPrecompiles, PrecompileProvider},
    interpreter::{CallInputs, InterpreterResult},
    precompile::{
        self, bn254, secp256r1, Precompile, PrecompileError, PrecompileId, PrecompileResult,
        Precompiles,
    },
    primitives::{hardfork::SpecId, Address, OnceLock},
};
use std::{boxed::Box, string::String};

/// Optimism precompile provider
#[derive(Debug, Clone)]
pub struct OpPrecompiles {
    /// Inner precompile provider is same as Ethereums.
    inner: EthPrecompiles,
    /// Spec id of the precompile provider.
    spec: OpSpecId,
}

impl OpPrecompiles {
    /// Create a new precompile provider with the given OpSpec.
    #[inline]
    pub fn new_with_spec(spec: OpSpecId) -> Self {
        let precompiles = match spec {
            spec @ (OpSpecId::BEDROCK
            | OpSpecId::REGOLITH
            | OpSpecId::CANYON
            | OpSpecId::ECOTONE) => Precompiles::new(spec.into_eth_spec().into()),
            OpSpecId::FJORD => fjord(),
            OpSpecId::GRANITE | OpSpecId::HOLOCENE => granite(),
            OpSpecId::ISTHMUS => isthmus(),
            OpSpecId::INTEROP | OpSpecId::OSAKA | OpSpecId::JOVIAN => jovian(),
        };

        Self {
            inner: EthPrecompiles {
                precompiles,
                spec: SpecId::default(),
            },
            spec,
        }
    }

    /// Precompiles getter.
    #[inline]
    pub fn precompiles(&self) -> &'static Precompiles {
        self.inner.precompiles
    }
}

/// Returns precompiles for Fjord spec.
pub fn fjord() -> &'static Precompiles {
    static INSTANCE: OnceLock<Precompiles> = OnceLock::new();
    INSTANCE.get_or_init(|| {
        let mut precompiles = Precompiles::cancun().clone();
        // RIP-7212: secp256r1 P256verify
        precompiles.extend([secp256r1::P256VERIFY]);
        precompiles
    })
}

/// Returns precompiles for Granite spec.
pub fn granite() -> &'static Precompiles {
    static INSTANCE: OnceLock<Precompiles> = OnceLock::new();
    INSTANCE.get_or_init(|| {
        let mut precompiles = fjord().clone();
        // Restrict bn254Pairing input size
        precompiles.extend([bn254_pair::GRANITE]);
        precompiles
    })
}

/// Returns precompiles for isthmus spec.
pub fn isthmus() -> &'static Precompiles {
    static INSTANCE: OnceLock<Precompiles> = OnceLock::new();
    INSTANCE.get_or_init(|| {
        let mut precompiles = granite().clone();
        // Prague bls12 precompiles
        precompiles.extend(precompile::bls12_381::precompiles());
        // Isthmus bls12 precompile modifications
        precompiles.extend([
            bls12_381::ISTHMUS_G1_MSM,
            bls12_381::ISTHMUS_G2_MSM,
            bls12_381::ISTHMUS_PAIRING,
        ]);
        precompiles
    })
}

/// Returns precompiles for jovian spec.
pub fn jovian() -> &'static Precompiles {
    static INSTANCE: OnceLock<Precompiles> = OnceLock::new();
    INSTANCE.get_or_init(|| {
        let mut precompiles = isthmus().clone();

        let mut to_remove = Precompiles::default();
        to_remove.extend([
            bn254::pair::ISTANBUL,
            bls12_381::ISTHMUS_G1_MSM,
            bls12_381::ISTHMUS_G2_MSM,
            bls12_381::ISTHMUS_PAIRING,
        ]);

        // Replace the 4 variable-input precompiles with Jovian versions (reduced limits)
        precompiles.difference(&to_remove);

        precompiles.extend([
            bn254_pair::JOVIAN,
            bls12_381::JOVIAN_G1_MSM,
            bls12_381::JOVIAN_G2_MSM,
            bls12_381::JOVIAN_PAIRING,
        ]);

        precompiles
    })
}

impl<CTX> PrecompileProvider<CTX> for OpPrecompiles
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
        self.inner.run(context, inputs)
    }

    #[inline]
    fn warm_addresses(&self) -> Box<impl Iterator<Item = Address>> {
        self.inner.warm_addresses()
    }

    #[inline]
    fn contains(&self, address: &Address) -> bool {
        self.inner.contains(address)
    }
}

impl Default for OpPrecompiles {
    fn default() -> Self {
        Self::new_with_spec(OpSpecId::JOVIAN)
    }
}

/// Bn254 pair precompile.
pub mod bn254_pair {
    use super::*;

    /// Max input size for the bn254 pair precompile.
    pub const GRANITE_MAX_INPUT_SIZE: usize = 112687;
    /// Bn254 pair precompile.
    pub const GRANITE: Precompile = Precompile::new(
        PrecompileId::Bn254Pairing,
        bn254::pair::ADDRESS,
        run_pair_granite,
    );

    /// Run the bn254 pair precompile with Optimism input limit.
    pub fn run_pair_granite(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > GRANITE_MAX_INPUT_SIZE {
            return Err(PrecompileError::Bn254PairLength);
        }
        bn254::run_pair(
            input,
            bn254::pair::ISTANBUL_PAIR_PER_POINT,
            bn254::pair::ISTANBUL_PAIR_BASE,
            gas_limit,
        )
    }

    /// Max input size for the bn254 pair precompile.
    pub const JOVIAN_MAX_INPUT_SIZE: usize = 81_984;
    /// Bn254 pair precompile.
    pub const JOVIAN: Precompile = Precompile::new(
        PrecompileId::Bn254Pairing,
        bn254::pair::ADDRESS,
        run_pair_jovian,
    );

    /// Run the bn254 pair precompile with Optimism input limit.
    pub fn run_pair_jovian(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > JOVIAN_MAX_INPUT_SIZE {
            return Err(PrecompileError::Bn254PairLength);
        }
        bn254::run_pair(
            input,
            bn254::pair::ISTANBUL_PAIR_PER_POINT,
            bn254::pair::ISTANBUL_PAIR_BASE,
            gas_limit,
        )
    }
}

/// Bls12_381 precompile.
pub mod bls12_381 {
    use super::*;
    use revm::precompile::bls12_381_const::{G1_MSM_ADDRESS, G2_MSM_ADDRESS, PAIRING_ADDRESS};

    /// Max input size for the g1 msm precompile.
    pub const ISTHMUS_G1_MSM_MAX_INPUT_SIZE: usize = 513760;

    /// The maximum input size for the BLS12-381 g1 msm operation after the Jovian Hardfork.
    pub const JOVIAN_G1_MSM_MAX_INPUT_SIZE: usize = 288_960;

    /// Max input size for the g2 msm precompile.
    pub const ISTHMUS_G2_MSM_MAX_INPUT_SIZE: usize = 488448;

    /// Max input size for the g2 msm precompile after the Jovian Hardfork.
    pub const JOVIAN_G2_MSM_MAX_INPUT_SIZE: usize = 278_784;

    /// Max input size for the pairing precompile.
    pub const ISTHMUS_PAIRING_MAX_INPUT_SIZE: usize = 235008;

    /// Max input size for the pairing precompile after the Jovian Hardfork.
    pub const JOVIAN_PAIRING_MAX_INPUT_SIZE: usize = 156_672;

    /// G1 msm precompile.
    pub const ISTHMUS_G1_MSM: Precompile =
        Precompile::new(PrecompileId::Bls12G1Msm, G1_MSM_ADDRESS, run_g1_msm_isthmus);
    /// G2 msm precompile.
    pub const ISTHMUS_G2_MSM: Precompile =
        Precompile::new(PrecompileId::Bls12G2Msm, G2_MSM_ADDRESS, run_g2_msm_isthmus);
    /// Pairing precompile.
    pub const ISTHMUS_PAIRING: Precompile = Precompile::new(
        PrecompileId::Bls12Pairing,
        PAIRING_ADDRESS,
        run_pair_isthmus,
    );

    /// G1 msm precompile after the Jovian Hardfork.
    pub const JOVIAN_G1_MSM: Precompile =
        Precompile::new(PrecompileId::Bls12G1Msm, G1_MSM_ADDRESS, run_g1_msm_jovian);
    /// G2 msm precompile after the Jovian Hardfork.
    pub const JOVIAN_G2_MSM: Precompile =
        Precompile::new(PrecompileId::Bls12G2Msm, G2_MSM_ADDRESS, run_g2_msm_jovian);
    /// Pairing precompile after the Jovian Hardfork.
    pub const JOVIAN_PAIRING: Precompile =
        Precompile::new(PrecompileId::Bls12Pairing, PAIRING_ADDRESS, run_pair_jovian);

    /// Run the g1 msm precompile with Optimism input limit.
    pub fn run_g1_msm_isthmus(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > ISTHMUS_G1_MSM_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "G1MSM input length too long for OP Stack input size limitation after the Isthmus Hardfork".into(),
            ));
        }
        precompile::bls12_381::g1_msm::g1_msm(input, gas_limit)
    }

    /// Run the g1 msm precompile with Optimism input limit.
    pub fn run_g1_msm_jovian(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > JOVIAN_G1_MSM_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "G1MSM input length too long for OP Stack input size limitation after the Jovian Hardfork".into(),
            ));
        }
        precompile::bls12_381::g1_msm::g1_msm(input, gas_limit)
    }

    /// Run the g2 msm precompile with Optimism input limit.
    pub fn run_g2_msm_isthmus(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > ISTHMUS_G2_MSM_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "G2MSM input length too long for OP Stack input size limitation".into(),
            ));
        }
        precompile::bls12_381::g2_msm::g2_msm(input, gas_limit)
    }

    /// Run the g2 msm precompile with Optimism input limit after the Jovian Hardfork.
    pub fn run_g2_msm_jovian(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > JOVIAN_G2_MSM_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "G2MSM input length too long for OP Stack input size limitation after the Jovian Hardfork".into(),
            ));
        }
        precompile::bls12_381::g2_msm::g2_msm(input, gas_limit)
    }

    /// Run the pairing precompile with Optimism input limit.
    pub fn run_pair_isthmus(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > ISTHMUS_PAIRING_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "Pairing input length too long for OP Stack input size limitation".into(),
            ));
        }
        precompile::bls12_381::pairing::pairing(input, gas_limit)
    }

    /// Run the pairing precompile with Optimism input limit after the Jovian Hardfork.
    pub fn run_pair_jovian(input: &[u8], gas_limit: u64) -> PrecompileResult {
        if input.len() > JOVIAN_PAIRING_MAX_INPUT_SIZE {
            return Err(PrecompileError::Other(
                "Pairing input length too long for OP Stack input size limitation after the Jovian Hardfork".into(),
            ));
        }
        precompile::bls12_381::pairing::pairing(input, gas_limit)
    }
}

#[cfg(test)]
mod tests {
    use crate::precompiles::bls12_381::{
        run_g1_msm_isthmus, run_g1_msm_jovian, run_g2_msm_isthmus, run_g2_msm_jovian,
        ISTHMUS_G1_MSM_MAX_INPUT_SIZE, ISTHMUS_G2_MSM_MAX_INPUT_SIZE,
        ISTHMUS_PAIRING_MAX_INPUT_SIZE, JOVIAN_G1_MSM_MAX_INPUT_SIZE, JOVIAN_G2_MSM_MAX_INPUT_SIZE,
        JOVIAN_PAIRING_MAX_INPUT_SIZE,
    };

    use super::*;
    use revm::{
        precompile::{bls12_381_const, PrecompileError},
        primitives::{hex, Bytes},
    };
    use std::vec;

    #[test]
    fn test_bn254_pair() {
        let input = hex::decode(
            "\
      1c76476f4def4bb94541d57ebba1193381ffa7aa76ada664dd31c16024c43f59\
      3034dd2920f673e204fee2811c678745fc819b55d3e9d294e45c9b03a76aef41\
      209dd15ebff5d46c4bd888e51a93cf99a7329636c63514396b4a452003a35bf7\
      04bf11ca01483bfa8b34b43561848d28905960114c8ac04049af4b6315a41678\
      2bb8324af6cfc93537a2ad1a445cfd0ca2a71acd7ac41fadbf933c2a51be344d\
      120a2a4cf30c1bf9845f20c6fe39e07ea2cce61f0c9bb048165fe5e4de877550\
      111e129f1cf1097710d41c4ac70fcdfa5ba2023c6ff1cbeac322de49d1b6df7c\
      2032c61a830e3c17286de9462bf242fca2883585b93870a73853face6a6bf411\
      198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c2\
      1800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed\
      090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b\
      12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa",
        )
        .unwrap();
        let expected =
            hex::decode("0000000000000000000000000000000000000000000000000000000000000001")
                .unwrap();
        let outcome = bn254_pair::run_pair_granite(&input, 260_000).unwrap();
        assert_eq!(outcome.bytes, expected);

        // Invalid input length
        let input = hex::decode(
            "\
          1111111111111111111111111111111111111111111111111111111111111111\
          1111111111111111111111111111111111111111111111111111111111111111\
          111111111111111111111111111111\
      ",
        )
        .unwrap();

        let res = bn254_pair::run_pair_granite(&input, 260_000);
        assert!(matches!(res, Err(PrecompileError::Bn254PairLength)));

        // Valid input length shorter than 112687
        let input = vec![1u8; 586 * bn254::PAIR_ELEMENT_LEN];
        let res = bn254_pair::run_pair_granite(&input, 260_000);
        assert!(matches!(res, Err(PrecompileError::OutOfGas)));

        // Input length longer than 112687
        let input = vec![1u8; 587 * bn254::PAIR_ELEMENT_LEN];
        let res = bn254_pair::run_pair_granite(&input, 260_000);
        assert!(matches!(res, Err(PrecompileError::Bn254PairLength)));
    }

    #[test]
    fn test_accelerated_bn254_pairing_jovian() {
        const TEST_INPUT: [u8; 384] = hex!(
            "2cf44499d5d27bb186308b7af7af02ac5bc9eeb6a3d147c186b21fb1b76e18da2c0f001f52110ccfe69108924926e45f0b0c868df0e7bde1fe16d3242dc715f61fb19bb476f6b9e44e2a32234da8212f61cd63919354bc06aef31e3cfaff3ebc22606845ff186793914e03e21df544c34ffe2f2f3504de8a79d9159eca2d98d92bd368e28381e8eccb5fa81fc26cf3f048eea9abfdd85d7ed3ab3698d63e4f902fe02e47887507adf0ff1743cbac6ba291e66f59be6bd763950bb16041a0a85e000000000000000000000000000000000000000000000000000000000000000130644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd451971ff0471b09fa93caaf13cbf443c1aede09cc4328f5a62aad45f40ec133eb4091058a3141822985733cbdddfed0fd8d6c104e9e9eff40bf5abfef9ab163bc72a23af9a5ce2ba2796c1f4e453a370eb0af8c212d9dc9acd8fc02c2e907baea223a8eb0b0996252cb548a4487da97b02422ebc0e834613f954de6c7e0afdc1fc"
        );
        const EXPECTED_OUTPUT: [u8; 32] =
            hex!("0000000000000000000000000000000000000000000000000000000000000001");

        let res = bn254_pair::run_pair_jovian(TEST_INPUT.as_ref(), u64::MAX);
        assert!(matches!(res, Ok(outcome) if **outcome.bytes == EXPECTED_OUTPUT));
    }

    #[test]
    fn test_accelerated_bn254_pairing_bad_input_len_jovian() {
        let input = [0u8; bn254_pair::JOVIAN_MAX_INPUT_SIZE + 1];
        let res = bn254_pair::run_pair_jovian(&input, u64::MAX);
        assert!(matches!(res, Err(PrecompileError::Bn254PairLength)));
    }

    #[test]
    fn test_get_jovian_precompile_with_bad_input_len() {
        let precompiles = OpPrecompiles::new_with_spec(OpSpecId::JOVIAN);
        let bn254_pair_precompile = precompiles
            .precompiles()
            .get(&bn254::pair::ADDRESS)
            .unwrap();

        let mut bad_input_len = bn254_pair::JOVIAN_MAX_INPUT_SIZE + 1;
        assert!(bad_input_len < bn254_pair::GRANITE_MAX_INPUT_SIZE);
        let input = vec![0u8; bad_input_len];

        let res = bn254_pair_precompile.execute(&input, u64::MAX);
        assert!(matches!(res, Err(PrecompileError::Bn254PairLength)));

        let bls12_381_g1_msm_precompile = precompiles
            .precompiles()
            .get(&bls12_381_const::G1_MSM_ADDRESS)
            .unwrap();
        bad_input_len = bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE + 1;
        assert!(bad_input_len < bls12_381::ISTHMUS_G1_MSM_MAX_INPUT_SIZE);
        let input = vec![0u8; bad_input_len];
        let res = bls12_381_g1_msm_precompile.execute(&input, u64::MAX);
        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );

        let bls12_381_g2_msm_precompile = precompiles
            .precompiles()
            .get(&bls12_381_const::G2_MSM_ADDRESS)
            .unwrap();
        bad_input_len = bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE + 1;
        assert!(bad_input_len < bls12_381::ISTHMUS_G2_MSM_MAX_INPUT_SIZE);
        let input = vec![0u8; bad_input_len];
        let res = bls12_381_g2_msm_precompile.execute(&input, u64::MAX);
        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );

        let bls12_381_pairing_precompile = precompiles
            .precompiles()
            .get(&bls12_381_const::PAIRING_ADDRESS)
            .unwrap();
        bad_input_len = bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE + 1;
        assert!(bad_input_len < bls12_381::ISTHMUS_PAIRING_MAX_INPUT_SIZE);
        let input = vec![0u8; bad_input_len];
        let res = bls12_381_pairing_precompile.execute(&input, u64::MAX);
        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }

    #[test]
    fn test_cancun_precompiles_in_fjord() {
        // additional to cancun, fjord has p256verify
        assert_eq!(fjord().difference(Precompiles::cancun()).len(), 1)
    }

    #[test]
    fn test_cancun_precompiles_in_granite() {
        // granite has p256verify (fjord)
        // granite has modification of cancun's bn254 pair (doesn't count as new precompile)
        assert_eq!(granite().difference(Precompiles::cancun()).len(), 1)
    }

    #[test]
    fn test_prague_precompiles_in_isthmus() {
        let new_prague_precompiles = Precompiles::prague().difference(Precompiles::cancun());

        // isthmus contains all precompiles that were new in prague, without modifications
        assert!(new_prague_precompiles.difference(isthmus()).is_empty())
    }

    #[test]
    fn test_prague_precompiles_in_jovian() {
        let new_prague_precompiles = Precompiles::prague().difference(Precompiles::cancun());

        // jovian contains all precompiles that were new in prague, without modifications
        assert!(new_prague_precompiles.difference(jovian()).is_empty())
    }

    /// All the addresses of the precompiles in isthmus should be in jovian
    #[test]
    fn test_isthmus_precompiles_in_jovian() {
        let new_isthmus_precompiles = isthmus().difference(Precompiles::cancun());

        // jovian contains all precompiles that were new in isthmus, without modifications
        assert!(new_isthmus_precompiles.difference(jovian()).is_empty())
    }

    #[test]
    fn test_default_precompiles_is_latest() {
        let latest = OpPrecompiles::new_with_spec(OpSpecId::default())
            .inner
            .precompiles;
        let default = OpPrecompiles::default().inner.precompiles;
        assert_eq!(latest.len(), default.len());

        let intersection = default.intersection(latest);
        assert_eq!(intersection.len(), latest.len())
    }

    #[test]
    fn test_g1_isthmus_max_size() {
        let oversized_input = vec![0u8; ISTHMUS_G1_MSM_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = run_g1_msm_isthmus(&input, 260_000);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }

    #[test]
    fn test_g1_jovian_max_size() {
        let oversized_input = vec![0u8; JOVIAN_G1_MSM_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = run_g1_msm_jovian(&input, u64::MAX);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }
    #[test]
    fn test_g2_isthmus_max_size() {
        let oversized_input = vec![0u8; ISTHMUS_G2_MSM_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = run_g2_msm_isthmus(&input, 260_000);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }
    #[test]
    fn test_g2_jovian_max_size() {
        let oversized_input = vec![0u8; JOVIAN_G2_MSM_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = run_g2_msm_jovian(&input, u64::MAX);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }
    #[test]
    fn test_pair_isthmus_max_size() {
        let oversized_input = vec![0u8; ISTHMUS_PAIRING_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = bls12_381::run_pair_isthmus(&input, 260_000);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }
    #[test]
    fn test_pair_jovian_max_size() {
        let oversized_input = vec![0u8; JOVIAN_PAIRING_MAX_INPUT_SIZE + 1];
        let input = Bytes::from(oversized_input);

        let res = bls12_381::run_pair_jovian(&input, u64::MAX);

        assert!(
            matches!(res, Err(PrecompileError::Other(msg)) if msg.contains("input length too long"))
        );
    }
}
