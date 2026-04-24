// Pedantic/nursery lints from workspace-level clippy config are largely stylistic;
// reth upstream maintains its own curated lint set. Allow the cosmetic/architectural-cost
// categories here so real issues stay visible.
#![allow(clippy::cast_lossless)]
#![allow(clippy::cast_possible_truncation)]
#![allow(clippy::cast_possible_wrap)]
#![allow(clippy::cast_precision_loss)]
#![allow(clippy::cast_sign_loss)]
#![allow(clippy::default_trait_access)]
#![allow(clippy::doc_markdown)]
#![allow(clippy::elidable_lifetime_names)]
#![allow(clippy::fallible_impl_from)]
#![allow(clippy::float_cmp)]
#![allow(clippy::future_not_send)]
#![allow(clippy::ignore_without_reason)]
#![allow(clippy::ignored_unit_patterns)]
#![allow(clippy::inconsistent_struct_constructor)]
#![allow(clippy::inline_always)]
#![allow(clippy::items_after_statements)]
#![allow(clippy::large_futures)]
#![allow(clippy::large_stack_arrays)]
#![allow(clippy::large_stack_frames)]
#![allow(clippy::manual_let_else)]
#![allow(clippy::map_unwrap_or)]
#![allow(clippy::match_wildcard_for_single_variants)]
#![allow(clippy::mismatching_type_param_order)]
#![allow(clippy::missing_const_for_fn)]
#![allow(clippy::missing_errors_doc)]
#![allow(clippy::missing_fields_in_debug)]
#![allow(clippy::missing_panics_doc)]
#![allow(clippy::must_use_candidate)]
#![allow(clippy::needless_pass_by_value)]
#![allow(clippy::needless_raw_string_hashes)]
#![allow(clippy::non_std_lazy_statics)]
#![allow(clippy::redundant_closure_for_method_calls)]
#![allow(clippy::redundant_pub_crate)]
#![allow(clippy::ref_option)]
#![allow(clippy::return_self_not_must_use)]
#![allow(clippy::semicolon_if_nothing_returned)]
#![allow(clippy::significant_drop_tightening)]
#![allow(clippy::similar_names)]
#![allow(clippy::single_match_else)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::struct_field_names)]
#![allow(clippy::too_long_first_doc_paragraph)]
#![allow(clippy::too_many_lines)]
#![allow(clippy::unchecked_time_subtraction)]
#![allow(clippy::uninlined_format_args)]
#![allow(clippy::unnecessary_semicolon)]
#![allow(clippy::unnecessary_wraps)]
#![allow(clippy::unreadable_literal)]
#![allow(clippy::unused_async)]
#![allow(clippy::unused_self)]
#![allow(clippy::use_self)]
#![allow(clippy::used_underscore_binding)]
#![allow(clippy::wildcard_imports)]

//! Optimism's payload builder implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![allow(clippy::useless_let_if_seq)]

extern crate alloc;

pub mod builder;
pub use builder::OpPayloadBuilder;
pub mod error;
pub mod payload;
use op_alloy_rpc_types_engine::OpExecutionData;
pub use payload::{
    OpBuiltPayload, OpExecData, OpPayloadAttributes, OpPayloadAttrs, OpPayloadBuilderAttributes,
    payload_id_optimism,
};
mod traits;
use reth_optimism_primitives::OpPrimitives;
use reth_payload_primitives::{BuiltPayload, PayloadTypes};
use reth_primitives_traits::{Block, NodePrimitives, SealedBlock};
pub use traits::*;
pub mod validator;
pub use validator::OpExecutionPayloadValidator;

pub mod config;

// Implement `ConfigureEngineEvm<OpExecData>` by delegating to the `OpExecutionData` implementation.
// This must live here because `OpExecData` is defined in this crate (orphan rules).
impl<ChainSpec, N, R> reth_evm::ConfigureEngineEvm<OpExecData>
    for reth_optimism_evm::OpEvmConfig<ChainSpec, N, R>
where
    N: NodePrimitives,
    R: Send + Sync + Unpin + Clone + 'static,
    ChainSpec: Send + Sync + Unpin + Clone + 'static,
    Self: reth_evm::ConfigureEngineEvm<OpExecutionData>,
{
    fn evm_env_for_payload(
        &self,
        payload: &OpExecData,
    ) -> Result<reth_evm::EvmEnvFor<Self>, <Self as reth_evm::ConfigureEvm>::Error> {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::evm_env_for_payload(self, &payload.0)
    }

    fn context_for_payload<'a>(
        &self,
        payload: &'a OpExecData,
    ) -> Result<reth_evm::ExecutionCtxFor<'a, Self>, <Self as reth_evm::ConfigureEvm>::Error> {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::context_for_payload(self, &payload.0)
    }

    fn tx_iterator_for_payload(
        &self,
        payload: &OpExecData,
    ) -> Result<impl reth_evm::ExecutableTxIterator<Self>, <Self as reth_evm::ConfigureEvm>::Error>
    {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::tx_iterator_for_payload(self, &payload.0)
    }
}

/// ZST that aggregates Optimism [`PayloadTypes`].
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
#[non_exhaustive]
pub struct OpPayloadTypes<N: NodePrimitives = OpPrimitives>(core::marker::PhantomData<N>);

impl<N: NodePrimitives> PayloadTypes for OpPayloadTypes<N>
where
    OpBuiltPayload<N>: BuiltPayload,
{
    type ExecutionData = crate::payload::OpExecData;
    type BuiltPayload = OpBuiltPayload<N>;
    type PayloadAttributes = crate::payload::OpPayloadAttrs;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        crate::payload::OpExecData::from(OpExecutionData::from_block_unchecked(
            block.hash(),
            &block.into_block().into_ethereum_block(),
        ))
    }
}
