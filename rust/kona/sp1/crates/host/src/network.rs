//! Utilities for network configuration and signer retrieval.

use std::env;

use anyhow::{Context, Result, anyhow};
use sp1_sdk::network::{FulfillmentStrategy, NetworkMode, signer::NetworkSigner};

/// Parse a fulfillment strategy from a string.
pub fn parse_fulfillment_strategy(value: String) -> FulfillmentStrategy {
    match value.to_ascii_lowercase().as_str() {
        "reserved" => FulfillmentStrategy::Reserved,
        "hosted" => FulfillmentStrategy::Hosted,
        "auction" => FulfillmentStrategy::Auction,
        _ => FulfillmentStrategy::UnspecifiedFulfillmentStrategy,
    }
}

/// Try to determine the network mode from the provided fulfillment strategies.
pub fn determine_network_mode(
    range_proof_strategy: FulfillmentStrategy,
    agg_proof_strategy: FulfillmentStrategy,
) -> Result<NetworkMode> {
    match (range_proof_strategy, agg_proof_strategy) {
        (FulfillmentStrategy::Auction, FulfillmentStrategy::Auction) => Ok(NetworkMode::Mainnet),
        (
            FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved,
            FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved,
        ) => Ok(NetworkMode::Reserved),
        (FulfillmentStrategy::UnspecifiedFulfillmentStrategy, _) |
        (_, FulfillmentStrategy::UnspecifiedFulfillmentStrategy) => {
            Err(anyhow!("The range and agg fulfillment Strategies must be specified"))
        }
        _ => Err(anyhow!(
            "The range fulfillment Strategy '{}' and agg fulfillment Strategy '{}' are incompatible",
            range_proof_strategy.as_str_name().to_ascii_lowercase(),
            agg_proof_strategy.as_str_name().to_ascii_lowercase()
        )),
    }
}

/// Compute the network signer using the `NETWORK_PRIVATE_KEY` env var.
/// If the `use_kms_requester` parameter is set to `true`, the `NETWORK_PRIVATE_KEY` env var
/// must be set with a key ARN.
pub async fn get_network_signer(use_kms_requester: bool) -> Result<NetworkSigner> {
    let network_signer = if use_kms_requester {
        // If using KMS, NETWORK_PRIVATE_KEY should be a KMS key ARN.
        let kms_key_arn = env::var("NETWORK_PRIVATE_KEY")
            .context("NETWORK_PRIVATE_KEY must be set when USE_KMS_REQUESTER is true")?;
        let signer = NetworkSigner::aws_kms(&kms_key_arn).await?;
        tracing::info!("Using KMS requester with address: {:?}", signer.address());

        signer
    } else {
        // Otherwise, use a private key with a default value to avoid errors in mock mode.
        let private_key = env::var("NETWORK_PRIVATE_KEY").unwrap_or_else(|_| {
            tracing::warn!(
                "Using default NETWORK_PRIVATE_KEY of 0x01. This is only valid in mock mode."
            );
            "0x0000000000000000000000000000000000000000000000000000000000000001".to_string()
        });
        let signer = NetworkSigner::local(&private_key)?;
        tracing::info!("Using local requester with address: {:?}", signer.address());

        signer
    };

    Ok(network_signer)
}
