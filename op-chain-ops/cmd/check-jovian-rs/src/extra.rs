use alloy::{consensus::BlockHeader, network::Ethereum, providers::Provider};
use log::info;
use std::error::Error;

pub async fn check<T: Provider<Ethereum>>(provider: &T) -> Result<(), Box<dyn Error>> {
    let block = provider
        .get_block_by_number(alloy::eips::BlockNumberOrTag::Finalized)
        .await?
        .unwrap();
    let extra_data = block.header.extra_data();
    info!("Extra data: {:?}", extra_data);
    Ok(())
}
