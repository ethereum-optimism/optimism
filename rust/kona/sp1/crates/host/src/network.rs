//! Utilities for network configuration and signer retrieval.

use std::env;

use anyhow::{Context, Result, anyhow, bail};
use sp1_sdk::{
    NetworkProver, ProverClient,
    network::{FulfillmentStrategy, NetworkMode, signer::NetworkSigner},
};

/// Parse a fulfillment strategy from a string.
pub fn parse_fulfillment_strategy(value: String) -> Result<FulfillmentStrategy> {
    match value.to_ascii_lowercase().as_str() {
        "reserved" => Ok(FulfillmentStrategy::Reserved),
        "hosted" => Ok(FulfillmentStrategy::Hosted),
        "auction" => Ok(FulfillmentStrategy::Auction),
        _ => bail!(
            "Invalid fulfillment strategy '{value}': must be 'reserved', 'hosted', or 'auction'"
        ),
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
        // Network proving spends real money. Require the key at startup instead
        // of failing on the first proof request.
        let private_key =
            env::var("NETWORK_PRIVATE_KEY").ok().filter(|key| !key.trim().is_empty()).context(
                "NETWORK_PRIVATE_KEY must be set for network proving \
                 (or set USE_KMS_REQUESTER=true to sign requests with AWS KMS)",
            )?;
        let signer = NetworkSigner::local(&private_key)?;
        tracing::info!("Using local requester with address: {:?}", signer.address());

        signer
    };

    Ok(network_signer)
}

/// Build a network prover from `USE_KMS_REQUESTER`, using the provided fulfillment strategy.
pub async fn build_network_prover_from_env(strategy: FulfillmentStrategy) -> Result<NetworkProver> {
    let use_kms_requester = env::var("USE_KMS_REQUESTER")
        .unwrap_or_else(|_| "false".to_string())
        .parse::<bool>()
        .context("USE_KMS_REQUESTER must be true or false")?;
    let network_signer = get_network_signer(use_kms_requester).await?;

    let network_mode = match strategy {
        FulfillmentStrategy::Auction => NetworkMode::Mainnet,
        FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved => NetworkMode::Reserved,
        _ => bail!("Fulfillment strategy must be 'reserved', 'hosted', or 'auction'"),
    };

    let prover =
        ProverClient::builder().network_for(network_mode).signer(network_signer).build().await;

    Ok(prover)
}
