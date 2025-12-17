use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus};
use alloy_rpc_types_eth::{Block, BlockNumberOrTag};
use jsonrpsee::core::async_trait;
use op_alloy_rpc_types_engine::OpPayloadAttributes;

use crate::ClientResult;
use crate::payload::{NewPayload, OpExecutionPayloadEnvelope, PayloadVersion};

#[async_trait]
pub trait EngineApiExt: Send + Sync + 'static {
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> ClientResult<ForkchoiceUpdated>;

    async fn new_payload(&self, new_payload: NewPayload) -> ClientResult<PayloadStatus>;

    async fn get_payload(
        &self,
        payload_id: PayloadId,
        version: PayloadVersion,
    ) -> ClientResult<OpExecutionPayloadEnvelope>;

    async fn get_block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> ClientResult<Block>;
}
