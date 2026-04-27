//! Signal types for the `kona-derive` pipeline.
//!
//! Signals are the primary method of communication between the pipeline driver
//! and the [`DerivationPipeline`]. The pipeline receives [`Signal`]s and
//! dispatches to stages via [`Stage`] methods.
//!
//! [`DerivationPipeline`]: crate::DerivationPipeline
//! [`Stage`]: crate::Stage

use kona_protocol::{BlockInfo, L2BlockInfo};

/// A signal to send to the pipeline.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(clippy::large_enum_variant)]
pub enum Signal {
    /// Reset the pipeline.
    Reset(ResetSignal),
    /// Hardfork Activation.
    Activation(ActivationSignal),
    /// Flush the currently active channel.
    FlushChannel,
    /// Provide a new L1 block to the L1 traversal stage.
    ProvideBlock(BlockInfo),
}

impl core::fmt::Display for Signal {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Reset(_) => write!(f, "reset"),
            Self::Activation(_) => write!(f, "activation"),
            Self::FlushChannel => write!(f, "flush_channel"),
            Self::ProvideBlock(_) => write!(f, "provide_block"),
        }
    }
}

/// A pipeline reset signal.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct ResetSignal {
    /// The L2 safe head to reset to.
    pub l2_safe_head: L2BlockInfo,
}

/// A pipeline hardfork activation signal.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct ActivationSignal {
    /// The L2 safe head to reset to.
    pub l2_safe_head: L2BlockInfo,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_reset_signal() {
        let signal = ResetSignal::default();
        assert_eq!(Signal::Reset(signal), Signal::Reset(signal));
    }

    #[test]
    fn test_activation_signal() {
        let signal = ActivationSignal::default();
        assert_eq!(Signal::Activation(signal), Signal::Activation(signal));
    }
}
