//! Response to safe head request

use alloy_eips::BlockNumHash;

/// The safe head response.
///
/// <https://github.com/ethereum-optimism/optimism/blob/77c91d09eaa44d2c53bec60eb89c5c55737bc325/op-service/eth/output.go#L19-L22>
/// Note: the optimism "eth.BlockID" type is number,hash <https://github.com/ethereum-optimism/optimism/blob/77c91d09eaa44d2c53bec60eb89c5c55737bc325/op-service/eth/id.go#L10-L13>
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SafeHeadResponse {
    /// The L1 block.
    pub l1_block: BlockNumHash,
    /// The safe head.
    pub safe_head: BlockNumHash,
}

#[cfg(test)]
mod tests {
    use super::*;

    // <https://github.com/alloy-rs/op-alloy/issues/155>
    #[test]
    fn test_safe_head_response() {
        let s = r#"{"l1Block":{"hash":"0x7de331305c2bb3e5642a2adcb9c003cc67cefc7b05a3da5a6a4b12cf3af15407","number":6834391},"safeHead":{"hash":"0xa5e5ec1ade7d6fef209f73861bf0080950cde74c4b0c07823983eb5225e282a8","number":18266679}}"#;
        let _response: SafeHeadResponse = serde_json::from_str(s).unwrap();
    }
}
