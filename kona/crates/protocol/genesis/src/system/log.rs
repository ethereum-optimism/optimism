//! Contains the [`SystemConfigLog`].

use alloy_primitives::Log;

use crate::{
    BatcherUpdate, CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC, Eip1559Update,
    GasConfigUpdate, GasLimitUpdate, LogProcessingError, OperatorFeeUpdate, SystemConfigUpdate,
    SystemConfigUpdateError, SystemConfigUpdateKind, UnsafeBlockSignerUpdate,
    updates::{DaFootprintGasScalarUpdate, MinBaseFeeUpdate},
};

/// The system config log is an EVM log entry emitted
/// by the system contract to update the system config.
///
/// The log data is formatted as follows:
/// ```text
/// event ConfigUpdate(
///    uint256 indexed version,
///    UpdateType indexed updateType,
///    bytes data
/// );
/// ```
#[derive(Debug, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct SystemConfigLog {
    /// The log.
    pub log: Log,
    /// Whether ecotone is active.
    pub ecotone_active: bool,
}

impl SystemConfigLog {
    /// Constructs a new system config update.
    pub const fn new(log: Log, ecotone_active: bool) -> Self {
        Self { log, ecotone_active }
    }

    /// Validate the log topic.
    pub fn validate_topic(&self) -> Result<(), LogProcessingError> {
        if self.log.topics().len() < 3 {
            return Err(LogProcessingError::InvalidTopicLen(self.log.topics().len()));
        }
        if self.log.topics()[0] != CONFIG_UPDATE_TOPIC {
            return Err(LogProcessingError::InvalidTopic);
        }
        Ok(())
    }

    /// Validate the config update version.
    pub fn validate_version(&self) -> Result<(), LogProcessingError> {
        let version = self.log.topics()[1];
        if version != CONFIG_UPDATE_EVENT_VERSION_0 {
            return Err(LogProcessingError::UnsupportedVersion(version));
        }
        Ok(())
    }

    /// Extracts the update type from the log.
    pub fn update_type(&self) -> Result<SystemConfigUpdateKind, SystemConfigUpdateError> {
        if self.log.topics().len() < 3 {
            return Err(LogProcessingError::InvalidTopicLen(self.log.topics().len()).into());
        }
        let topic = self.log.topics()[2];
        let topic_bytes = <&[u8; 8]>::try_from(&topic.as_slice()[24..])
            .map_err(|_| LogProcessingError::UpdateTypeDecodingError)?;
        let ty = u64::from_be_bytes(*topic_bytes);
        ty.try_into().map_err(|_| {
            SystemConfigUpdateError::LogProcessing(
                LogProcessingError::InvalidSystemConfigUpdateType(ty),
            )
        })
    }

    /// Builds the [`SystemConfigUpdate`] from the log.
    pub fn build(&self) -> Result<SystemConfigUpdate, SystemConfigUpdateError> {
        self.validate_topic()?;
        self.validate_version()?;
        match self.update_type()? {
            SystemConfigUpdateKind::Batcher => {
                let update = BatcherUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::Batcher(update))
            }
            SystemConfigUpdateKind::GasConfig => {
                let update = GasConfigUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::GasConfig(update))
            }
            SystemConfigUpdateKind::GasLimit => {
                let update = GasLimitUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::GasLimit(update))
            }
            SystemConfigUpdateKind::Eip1559 => {
                let update = Eip1559Update::try_from(self)?;
                Ok(SystemConfigUpdate::Eip1559(update))
            }
            SystemConfigUpdateKind::OperatorFee => {
                let update = OperatorFeeUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::OperatorFee(update))
            }
            SystemConfigUpdateKind::UnsafeBlockSigner => {
                let update = UnsafeBlockSignerUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::UnsafeBlockSigner(update))
            }
            SystemConfigUpdateKind::MinBaseFee => {
                let update = MinBaseFeeUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::MinBaseFee(update))
            }
            SystemConfigUpdateKind::DaFootprintGasScalar => {
                let update = DaFootprintGasScalarUpdate::try_from(self)?;
                Ok(SystemConfigUpdate::DaFootprintGasScalar(update))
            }
        }
    }
}
