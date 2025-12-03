use alloy::{consensus::BlockHeader, network::Ethereum, providers::Provider};
use log::{info, warn};
use op_alloy_consensus::decode_jovian_extra_data;

use std::error::Error;

pub async fn check<T: Provider<Ethereum>>(provider: &T) -> Result<(), Box<dyn Error>> {
    let block = provider
        .get_block_by_number(alloy::eips::BlockNumberOrTag::Finalized)
        .await?
        .unwrap();
    let extra_data = block.header.extra_data();
    info!("Extra data: {:?}", extra_data);

    match decode_jovian_extra_data(&extra_data) {
        Ok(data) => {
            info!(
                "Decoded extra data: elasticity:{}, denominator:{}, minBaseFee:{}",
                data.0, data.1, data.2
            );
        }
        Err(err) => {
            return Err(Box::new(err));
        }
    }

    match block.header.blob_gas_used() {
        Some(0) => {
            warn!("Zero Blob gas used, inconclusive");
        }
        Some(d) => {
            info!("Nonzero Blob gas used: {}", d);
        }
        None => {
            return Err(Box::<dyn Error>::from("Nil blobGasUsed".to_string()));
        }
    }

    Ok(())
}
