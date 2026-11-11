//! op-reth runs with state masking off: it defaults the window to 0 where it owns the engine
//! defaults and rejects a nonzero window where it can only observe the config it is launched with.

use reth_node_api::{AddOnsContext, FullNodeComponents, PrimitivesTy, TreeConfig};
use reth_node_builder::rpc::EngineValidatorBuilder;
use reth_node_core::args::{DefaultEngineValues, EngineArgs};
use reth_storage_overlay::OverlayManager;
use reth_tracing::tracing::info;

/// Startup error for a nonzero effective state masking window.
pub const STATE_MASKING_UNSUPPORTED: &str =
    "--engine.num-state-masking-blocks must be 0 on op-reth: state masking is not supported";

/// The engine defaults were locked with a nonzero state masking window before
/// [`init_op_engine_defaults`] ran, so every `EngineArgs` built from them masks state.
#[derive(Clone, Copy, Debug, Eq, PartialEq, thiserror::Error)]
#[error(
    "engine defaults were locked with --engine.num-state-masking-blocks {num_state_masking_blocks} \
     before op-reth could default state masking off; a node launched with this default is rejected"
)]
pub struct EngineDefaultsLockedMasked {
    /// The locked default window.
    pub num_state_masking_blocks: u64,
}

/// Defaults `--engine.num-state-masking-blocks` to 0 for every [`EngineArgs`] built from here on.
///
/// The engine defaults are a process-global that locks on first use, so this must run before the
/// first `EngineArgs` or `NodeConfig` is built. A default locked earlier is kept and reported
/// when it masks; any node launched with it is rejected.
pub fn init_op_engine_defaults() -> Result<(), EngineDefaultsLockedMasked> {
    if DefaultEngineValues::default().with_num_state_masking_blocks(0).try_init().is_ok() {
        return Ok(());
    }
    match EngineArgs::default().num_state_masking_blocks() {
        0 => Ok(()),
        num_state_masking_blocks => Err(EngineDefaultsLockedMasked { num_state_masking_blocks }),
    }
}

/// Rejects a nonzero state masking window; op-reth does not support state masking.
pub fn ensure_no_state_masking(num_state_masking_blocks: u64) -> eyre::Result<()> {
    eyre::ensure!(num_state_masking_blocks == 0, STATE_MASKING_UNSUPPORTED);
    Ok(())
}

/// Builds the engine tree validator with `inner`, rejecting a [`TreeConfig`] that masks state.
#[derive(Clone, Debug)]
pub struct NoStateMaskingValidatorBuilder<EVB> {
    inner: EVB,
}

impl<EVB> NoStateMaskingValidatorBuilder<EVB> {
    /// Wraps `inner`.
    pub const fn new(inner: EVB) -> Self {
        Self { inner }
    }
}

impl<N, EVB> EngineValidatorBuilder<N> for NoStateMaskingValidatorBuilder<EVB>
where
    N: FullNodeComponents,
    EVB: EngineValidatorBuilder<N>,
{
    type EngineValidator = EVB::EngineValidator;

    async fn build_tree_validator(
        self,
        ctx: &AddOnsContext<'_, N>,
        tree_config: TreeConfig,
        overlay_manager: OverlayManager<PrimitivesTy<N::Types>>,
    ) -> eyre::Result<Self::EngineValidator> {
        ensure_no_state_masking(tree_config.num_state_masking_blocks())?;
        info!(target: "reth::cli", "Engine tree config has state masking off");
        self.inner.build_tree_validator(ctx, tree_config, overlay_manager).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case::masking_off(0, true)]
    #[case::masking_on(30, false)]
    fn state_masking_guard(#[case] num_state_masking_blocks: u64, #[case] accepted: bool) {
        assert_eq!(ensure_no_state_masking(num_state_masking_blocks).is_ok(), accepted);
    }

    #[test]
    fn init_op_engine_defaults_turns_state_masking_off() {
        init_op_engine_defaults().unwrap();

        assert_eq!(EngineArgs::default().num_state_masking_blocks(), 0);
    }
}
