//! Contains the [`SystemConfig`] type.

use crate::{
    CONFIG_UPDATE_TOPIC, RollupConfig, SystemConfigLog, SystemConfigUpdateError,
    SystemConfigUpdateKind,
};
use alloy_consensus::{Eip658Value, Receipt};
use alloy_primitives::{Address, B64, Log, U256};

/// System configuration.
#[derive(Debug, Copy, Clone, Default, Hash, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct SystemConfig {
    /// Batcher address
    #[cfg_attr(feature = "serde", serde(rename = "batcherAddr"))]
    pub batcher_address: Address,
    /// Fee overhead value
    #[cfg_attr(feature = "serde", serde(serialize_with = "serialize_u256_full"))]
    pub overhead: U256,
    /// Fee scalar value
    #[cfg_attr(feature = "serde", serde(serialize_with = "serialize_u256_full"))]
    pub scalar: U256,
    /// Gas limit value
    pub gas_limit: u64,
    /// Base fee scalar value
    pub base_fee_scalar: Option<u64>,
    /// Blob base fee scalar value
    pub blob_base_fee_scalar: Option<u64>,
    /// EIP-1559 denominator
    pub eip1559_denominator: Option<u32>,
    /// EIP-1559 elasticity
    pub eip1559_elasticity: Option<u32>,
    /// The operator fee scalar (isthmus hardfork)
    pub operator_fee_scalar: Option<u32>,
    /// The operator fee constant (isthmus hardfork)
    pub operator_fee_constant: Option<u64>,
    /// Min base fee (jovian hardfork)
    /// Note: according to the [spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/system-config.md#initialization), as long as the MinBaseFee is not
    /// explicitly set, the default value (`0`) will be systematically applied.
    pub min_base_fee: Option<u64>,
    /// DA footprint gas scalar (Jovian hardfork)
    /// Note: according to the [spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/system-config.md#initialization), as long as the DAFootprintGasScalar is not
    /// explicitly set, the default value (`400`) will be systematically applied.
    pub da_footprint_gas_scalar: Option<u16>,
}

/// Custom EIP-1559 parameter decoding is needed here for holocene encoding.
///
/// This is used by the Optimism monorepo [here][here].
///
/// [here]: https://github.com/ethereum-optimism/optimism/blob/cf28bffc7d880292794f53bb76bfc4df7898307b/op-service/eth/types.go#L519
#[cfg(feature = "serde")]
impl<'a> serde::Deserialize<'a> for SystemConfig {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'a>,
    {
        use alloy_primitives::B256;
        // An alias struct that is identical to `SystemConfig`.
        // We use the alias to decode the eip1559 params as their u32 values.
        #[derive(serde::Deserialize)]
        #[serde(rename_all = "camelCase")]
        #[serde(deny_unknown_fields)]
        struct SystemConfigAlias {
            #[serde(rename = "batcherAddress", alias = "batcherAddr")]
            batcher_address: Address,
            overhead: U256,
            scalar: U256,
            gas_limit: u64,
            base_fee_scalar: Option<u64>,
            blob_base_fee_scalar: Option<u64>,
            eip1559_params: Option<B64>,
            eip1559_denominator: Option<u32>,
            eip1559_elasticity: Option<u32>,
            operator_fee_params: Option<B256>,
            operator_fee_scalar: Option<u32>,
            operator_fee_constant: Option<u64>,
            min_base_fee: Option<u64>,
            da_footprint_gas_scalar: Option<u16>,
        }

        let mut alias = SystemConfigAlias::deserialize(deserializer)?;
        if let Some(params) = alias.eip1559_params {
            alias.eip1559_denominator =
                Some(u32::from_be_bytes(params.as_slice().get(0..4).unwrap().try_into().unwrap()));
            alias.eip1559_elasticity =
                Some(u32::from_be_bytes(params.as_slice().get(4..8).unwrap().try_into().unwrap()));
        }
        if let Some(params) = alias.operator_fee_params {
            alias.operator_fee_scalar = Some(u32::from_be_bytes(
                params.as_slice().get(20..24).unwrap().try_into().unwrap(),
            ));
            alias.operator_fee_constant = Some(u64::from_be_bytes(
                params.as_slice().get(24..32).unwrap().try_into().unwrap(),
            ));
        }

        Ok(Self {
            batcher_address: alias.batcher_address,
            overhead: alias.overhead,
            scalar: alias.scalar,
            gas_limit: alias.gas_limit,
            base_fee_scalar: alias.base_fee_scalar,
            blob_base_fee_scalar: alias.blob_base_fee_scalar,
            eip1559_denominator: alias.eip1559_denominator,
            eip1559_elasticity: alias.eip1559_elasticity,
            operator_fee_scalar: alias.operator_fee_scalar,
            operator_fee_constant: alias.operator_fee_constant,
            min_base_fee: alias.min_base_fee,
            da_footprint_gas_scalar: alias.da_footprint_gas_scalar,
        })
    }
}

impl SystemConfig {
    /// Filters all L1 receipts to find config updates and applies the config updates.
    ///
    /// Returns `true` if any config updates were applied, `false` otherwise.
    pub fn update_with_receipts(
        &mut self,
        receipts: &[Receipt],
        l1_system_config_address: Address,
        ecotone_active: bool,
    ) -> Result<bool, SystemConfigUpdateError> {
        let mut updated = false;
        for receipt in receipts {
            if Eip658Value::Eip658(false) == receipt.status {
                continue;
            }

            receipt.logs.iter().try_for_each(|log| {
                let topics = log.topics();
                if log.address == l1_system_config_address &&
                    !topics.is_empty() &&
                    topics[0] == CONFIG_UPDATE_TOPIC
                {
                    // Safety: Error is bubbled up by the trailing `?`
                    self.process_config_update_log(log, ecotone_active)?;
                    updated = true;
                }
                Ok::<(), SystemConfigUpdateError>(())
            })?;
        }
        Ok(updated)
    }

    /// Returns the eip1559 parameters from a [SystemConfig] encoded as a [B64].
    pub fn eip_1559_params(
        &self,
        rollup_config: &RollupConfig,
        parent_timestamp: u64,
        next_timestamp: u64,
    ) -> Option<B64> {
        let is_holocene = rollup_config.is_holocene_active(next_timestamp);

        // For the first holocene block, a zero'd out B64 is returned to signal the
        // execution layer to use the canyon base fee parameters. Else, the system
        // config's eip1559 parameters are encoded as a B64.
        if is_holocene && !rollup_config.is_holocene_active(parent_timestamp) {
            Some(B64::ZERO)
        } else {
            is_holocene.then_some(B64::from_slice(
                &[
                    self.eip1559_denominator.unwrap_or_default().to_be_bytes(),
                    self.eip1559_elasticity.unwrap_or_default().to_be_bytes(),
                ]
                .concat(),
            ))
        }
    }

    /// Decodes an EVM log entry emitted by the system config contract and applies it as a
    /// [SystemConfig] change.
    ///
    /// Parse log data for:
    ///
    /// ```text
    /// event ConfigUpdate(
    ///    uint256 indexed version,
    ///    UpdateType indexed updateType,
    ///    bytes data
    /// );
    /// ```
    fn process_config_update_log(
        &mut self,
        log: &Log,
        ecotone_active: bool,
    ) -> Result<SystemConfigUpdateKind, SystemConfigUpdateError> {
        // Construct the system config log from the log.
        let log = SystemConfigLog::new(log.clone(), ecotone_active);

        // Construct the update type from the log.
        let update = log.build()?;

        // Apply the update to the system config.
        update.apply(self);

        // Return the update type.
        Ok(update.kind())
    }
}

/// Compatibility helper function to serialize a [`U256`] as a [`B256`].
///
/// [`B256`]: alloy_primitives::B256
#[cfg(feature = "serde")]
fn serialize_u256_full<S>(ts: &U256, ser: S) -> Result<S::Ok, S::Error>
where
    S: serde::Serializer,
{
    use serde::Serialize;

    alloy_primitives::B256::from(ts.to_be_bytes::<32>()).serialize(ser)
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, HardForkConfig};
    use alloc::vec;
    use alloy_primitives::{B256, LogData, address, b256, hex};

    #[test]
    #[cfg(feature = "serde")]
    fn test_system_config_da_footprint_gas_scalar() {
        let raw = r#"{
        "batcherAddress": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
          "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
          "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
          "gasLimit": 30000000,
          "eip1559Params": "0x000000ab000000cd",
          "daFootprintGasScalar": 10
        }"#;
        let system_config: SystemConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(system_config.da_footprint_gas_scalar, Some(10), "da_footprint_gas_scalar");
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_system_config_eip1559_params() {
        let raw = r#"{
          "batcherAddress": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
          "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
          "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
          "gasLimit": 30000000,
          "eip1559Params": "0x000000ab000000cd"
        }"#;
        let system_config: SystemConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(system_config.eip1559_denominator, Some(0xab_u32), "eip1559_denominator");
        assert_eq!(system_config.eip1559_elasticity, Some(0xcd_u32), "eip1559_elasticity");
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_system_config_serde() {
        let raw = r#"{
          "batcherAddr": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
          "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
          "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
          "gasLimit": 30000000
        }"#;
        let expected = SystemConfig {
            batcher_address: address!("6887246668a3b87F54DeB3b94Ba47a6f63F32985"),
            overhead: U256::from(0xbc),
            scalar: U256::from(0xa6fe0),
            gas_limit: 30000000,
            ..Default::default()
        };

        let deserialized: SystemConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(deserialized, expected);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_system_config_unknown_field() {
        let raw = r#"{
          "batcherAddr": "0x6887246668a3b87F54DeB3b94Ba47a6f63F32985",
          "overhead": "0x00000000000000000000000000000000000000000000000000000000000000bc",
          "scalar": "0x00000000000000000000000000000000000000000000000000000000000a6fe0",
          "gasLimit": 30000000,
          "unknown": 0
        }"#;
        let err = serde_json::from_str::<SystemConfig>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }

    #[test]
    #[cfg(feature = "arbitrary")]
    fn test_arbitrary_system_config() {
        use arbitrary::Arbitrary;
        use rand::Rng;
        let mut bytes = [0u8; 1024];
        rand::rng().fill(bytes.as_mut_slice());
        SystemConfig::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap();
    }

    #[test]
    fn test_eip_1559_params_from_system_config_none() {
        let rollup_config = RollupConfig::default();
        let sys_config = SystemConfig::default();
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 0), None);
    }

    #[test]
    fn test_eip_1559_params_from_system_config_some() {
        let rollup_config = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let sys_config = SystemConfig {
            eip1559_denominator: Some(1),
            eip1559_elasticity: None,
            ..Default::default()
        };
        let expected = Some(B64::from_slice(&[1u32.to_be_bytes(), 0u32.to_be_bytes()].concat()));
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 0), expected);
    }

    #[test]
    fn test_eip_1559_params_from_system_config() {
        let rollup_config = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let sys_config = SystemConfig {
            eip1559_denominator: Some(1),
            eip1559_elasticity: Some(2),
            ..Default::default()
        };
        let expected = Some(B64::from_slice(&[1u32.to_be_bytes(), 2u32.to_be_bytes()].concat()));
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 0), expected);
    }

    #[test]
    fn test_default_eip_1559_params_from_system_config() {
        let rollup_config = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let sys_config = SystemConfig {
            eip1559_denominator: None,
            eip1559_elasticity: None,
            ..Default::default()
        };
        let expected = Some(B64::ZERO);
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 0), expected);
    }

    #[test]
    fn test_default_eip_1559_params_from_system_config_pre_holocene() {
        let rollup_config = RollupConfig::default();
        let sys_config = SystemConfig {
            eip1559_denominator: Some(1),
            eip1559_elasticity: Some(2),
            ..Default::default()
        };
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 0), None);
    }

    #[test]
    fn test_default_eip_1559_params_first_block_holocene() {
        let rollup_config = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        };
        let sys_config = SystemConfig {
            eip1559_denominator: Some(1),
            eip1559_elasticity: Some(2),
            ..Default::default()
        };
        assert_eq!(sys_config.eip_1559_params(&rollup_config, 0, 2), Some(B64::ZERO));
    }

    #[test]
    fn test_system_config_update_with_receipts_unchanged() {
        let mut system_config = SystemConfig::default();
        let receipts = vec![];
        let l1_system_config_address = Address::ZERO;
        let ecotone_active = false;

        let updated = system_config
            .update_with_receipts(&receipts, l1_system_config_address, ecotone_active)
            .unwrap();
        assert!(!updated);

        assert_eq!(system_config, SystemConfig::default());
    }

    #[test]
    fn test_system_config_update_with_receipts_batcher_address() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000000");
        let mut system_config = SystemConfig::default();
        let l1_system_config_address = Address::ZERO;
        let ecotone_active = false;

        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        let receipt = Receipt {
            logs: vec![update_log],
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 0,
        };

        let updated = system_config
            .update_with_receipts(&[receipt], l1_system_config_address, ecotone_active)
            .unwrap();
        assert!(updated);

        assert_eq!(
            system_config.batcher_address,
            address!("000000000000000000000000000000000000bEEF"),
        );
    }

    #[test]
    fn test_system_config_update_batcher_log() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000000");

        let mut system_config = SystemConfig::default();

        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        // Update the batcher address.
        system_config.process_config_update_log(&update_log, false).unwrap();

        assert_eq!(
            system_config.batcher_address,
            address!("000000000000000000000000000000000000bEEF")
        );
    }

    #[test]
    fn test_system_config_update_gas_config_log() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000001");

        let mut system_config = SystemConfig::default();

        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000babe000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        // Update the batcher address.
        system_config.process_config_update_log(&update_log, false).unwrap();

        assert_eq!(system_config.overhead, U256::from(0xbabe));
        assert_eq!(system_config.scalar, U256::from(0xbeef));
    }

    #[test]
    fn test_system_config_update_gas_config_log_ecotone() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000001");

        let mut system_config = SystemConfig::default();

        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000babe000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        // Update the gas limit.
        system_config.process_config_update_log(&update_log, true).unwrap();

        assert_eq!(system_config.overhead, U256::from(0));
        assert_eq!(system_config.scalar, U256::from(0xbeef));
    }

    #[test]
    fn test_system_config_update_gas_limit_log() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000002");

        let mut system_config = SystemConfig::default();

        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        // Update the gas limit.
        system_config.process_config_update_log(&update_log, false).unwrap();

        assert_eq!(system_config.gas_limit, 0xbeef_u64);
    }

    #[test]
    fn test_system_config_update_eip1559_params_log() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000004");

        let mut system_config = SystemConfig::default();
        let update_log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        // Update the EIP-1559 parameters.
        system_config.process_config_update_log(&update_log, false).unwrap();

        assert_eq!(system_config.eip1559_denominator, Some(0xbabe_u32));
        assert_eq!(system_config.eip1559_elasticity, Some(0xbeef_u32));
    }

    #[test]
    fn test_system_config_update_operator_fee_log() {
        const UPDATE_TYPE: B256 =
            b256!("0000000000000000000000000000000000000000000000000000000000000005");

        let mut system_config = SystemConfig::default();
        let update_log  = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    UPDATE_TYPE,
                ],
                hex!("0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000babe000000000000beef").into()
            )
        };

        // Update the operator fee.
        system_config.process_config_update_log(&update_log, false).unwrap();

        assert_eq!(system_config.operator_fee_scalar, Some(0xbabe_u32));
        assert_eq!(system_config.operator_fee_constant, Some(0xbeef_u64));
    }
}
