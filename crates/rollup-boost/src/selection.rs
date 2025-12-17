use crate::payload::{OpExecutionPayloadEnvelope, PayloadSource};
use serde::{Deserialize, Serialize};

/// Defines the strategy for choosing between the builder block and the L2 client block
/// during block production.
#[derive(Serialize, Deserialize, Debug, Copy, Clone, PartialEq, clap::ValueEnum)]
pub enum BlockSelectionPolicy {
    /// Selects the block based on gas usage.
    ///
    /// If the builder block uses less than 10% of the gas used by the L2 client block,
    /// the L2 block is selected instead. This prevents propagation of valid but empty
    /// builder blocks and mitigates issues where the builder is not receiving enough
    /// transactions due to networking or peering failures.
    GasUsed,
}

impl BlockSelectionPolicy {
    pub fn select_block(
        &self,
        builder_payload: OpExecutionPayloadEnvelope,
        l2_payload: OpExecutionPayloadEnvelope,
    ) -> (OpExecutionPayloadEnvelope, PayloadSource) {
        match self {
            BlockSelectionPolicy::GasUsed => {
                let builder_gas = builder_payload.gas_used() as f64;
                let l2_gas = l2_payload.gas_used() as f64;

                // Select the L2 block if the builder block uses less than 10% of the gas.
                // This avoids selecting empty or severely underfilled blocks,
                if builder_gas < l2_gas * 0.1 {
                    (l2_payload, PayloadSource::L2)
                } else {
                    (builder_payload, PayloadSource::Builder)
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV4;

    #[test]
    fn test_gas_used_policy_select_l2_block() -> eyre::Result<()> {
        let execution_payload = r#"{"executionPayload":{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0","withdrawalsRoot":"0x123400000000000000000000000000000000000000000000000000000000babe"},"blockValue":"0x0","blobsBundle":{"commitments":[],"proofs":[],"blobs":[]},"shouldOverrideBuilder":false,"parentBeaconBlockRoot":"0xdead00000000000000000000000000000000000000000000000000000000beef","executionRequests":["0xdeadbeef"]}"#;
        let mut builder_payload: OpExecutionPayloadEnvelopeV4 =
            serde_json::from_str(execution_payload)?;
        let mut l2_payload = builder_payload.clone();

        let gas_used = 1000000000;
        l2_payload
            .execution_payload
            .payload_inner
            .payload_inner
            .payload_inner
            .gas_used = gas_used;

        builder_payload
            .execution_payload
            .payload_inner
            .payload_inner
            .payload_inner
            .gas_used = (gas_used as f64 * 0.09) as u64;

        let builder_payload = OpExecutionPayloadEnvelope::V4(builder_payload);
        let l2_payload = OpExecutionPayloadEnvelope::V4(l2_payload);

        let selected_payload =
            BlockSelectionPolicy::GasUsed.select_block(builder_payload, l2_payload);

        assert_eq!(selected_payload.1, PayloadSource::L2);
        Ok(())
    }

    #[test]
    fn test_gas_used_policy_select_builder_block() -> eyre::Result<()> {
        let execution_payload = r#"{"executionPayload":{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0","withdrawalsRoot":"0x123400000000000000000000000000000000000000000000000000000000babe"},"blockValue":"0x0","blobsBundle":{"commitments":[],"proofs":[],"blobs":[]},"shouldOverrideBuilder":false,"parentBeaconBlockRoot":"0xdead00000000000000000000000000000000000000000000000000000000beef","executionRequests":["0xdeadbeef"]}"#;
        let mut builder_payload: OpExecutionPayloadEnvelopeV4 =
            serde_json::from_str(execution_payload)?;
        let mut l2_payload = builder_payload.clone();

        let gas_used = 1000000000;
        l2_payload
            .execution_payload
            .payload_inner
            .payload_inner
            .payload_inner
            .gas_used = gas_used;

        builder_payload
            .execution_payload
            .payload_inner
            .payload_inner
            .payload_inner
            .gas_used = (gas_used as f64 * 0.1) as u64;

        let builder_payload = OpExecutionPayloadEnvelope::V4(builder_payload);
        let l2_payload = OpExecutionPayloadEnvelope::V4(l2_payload);

        let selected_payload =
            BlockSelectionPolicy::GasUsed.select_block(builder_payload, l2_payload);

        assert_eq!(selected_payload.1, PayloadSource::Builder);
        Ok(())
    }
}
