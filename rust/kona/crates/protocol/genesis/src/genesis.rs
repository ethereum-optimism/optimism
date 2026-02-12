//! Genesis types.

use alloy_eips::eip1898::BlockNumHash;

use crate::SystemConfig;

/// Chain genesis information.
#[derive(Debug, Copy, Clone, Default, Hash, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct ChainGenesis {
    /// L1 genesis block
    pub l1: BlockNumHash,
    /// L2 genesis block
    pub l2: BlockNumHash,
    /// Timestamp of the L2 genesis block
    pub l2_time: u64,
    /// Optional System configuration
    pub system_config: Option<SystemConfig>,
}

#[cfg(feature = "arbitrary")]
impl<'a> arbitrary::Arbitrary<'a> for ChainGenesis {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        let system_config = Option::<SystemConfig>::arbitrary(u)?;
        let l1_num_hash = BlockNumHash {
            number: u64::arbitrary(u)?,
            hash: alloy_primitives::B256::arbitrary(u)?,
        };
        let l2_num_hash = BlockNumHash {
            number: u64::arbitrary(u)?,
            hash: alloy_primitives::B256::arbitrary(u)?,
        };
        Ok(Self { l1: l1_num_hash, l2: l2_num_hash, l2_time: u.arbitrary()?, system_config })
    }
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;
    use alloy_primitives::{address, b256, uint};

    const fn ref_genesis() -> ChainGenesis {
        ChainGenesis {
            l1: BlockNumHash {
                hash: b256!("438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108"),
                number: 17422590,
            },
            l2: BlockNumHash {
                hash: b256!("dbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3"),
                number: 105235063,
            },
            l2_time: 1686068903,
            system_config: Some(SystemConfig {
                batcher_address: address!("6887246668a3b87F54DeB3b94Ba47a6f63F32985"),
                overhead: uint!(0xbc_U256),
                scalar: uint!(0xa6fe0_U256),
                gas_limit: 30000000,
                base_fee_scalar: None,
                blob_base_fee_scalar: None,
                eip1559_denominator: None,
                eip1559_elasticity: None,
                operator_fee_scalar: None,
                operator_fee_constant: None,
                min_base_fee: None,
                da_footprint_gas_scalar: Some(10),
            }),
        }
    }

    #[test]
    fn test_genesis_serde() {
        let genesis_str = r#"{
            "l1": {
              "hash": "0x438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108",
              "number": 17422590
            },
            "l2": {
              "hash": "0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3",
              "number": 105235063
            },
            "l2_time": 1686068903,
            "system_config": {
              "batcherAddress": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
              "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
              "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
              "gasLimit": 30000000,
              "daFootprintGasScalar": 10
            }
          }"#;
        let genesis: ChainGenesis = serde_json::from_str(genesis_str).unwrap();
        assert_eq!(genesis, ref_genesis());
    }

    #[test]
    fn test_genesis_unknown_field_json() {
        let raw: &str = r#"{
            "l1": {
              "hash": "0x438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108",
              "number": 17422590
            },
            "l2": {
              "hash": "0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3",
              "number": 105235063
            },
            "l2_time": 1686068903,
            "system_config": {
              "batcherAddress": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
              "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
              "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
              "gasLimit": 30000000
            },
            "unknown_field": "unknown"
        }"#;

        let err = serde_json::from_str::<ChainGenesis>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }
}
