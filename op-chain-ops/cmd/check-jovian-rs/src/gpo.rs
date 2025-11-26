use alloy::{primitives::address, providers::ProviderBuilder, sol};
use log::{error, info};
use std::error::Error;

// Generate the contract bindings for the ERC20 interface.
sol! {
   // The `rpc` attribute enables contract interaction via the provider.
   #[sol(rpc)]
   contract GasPriceOracle {
        function isJovian() public view returns (bool);
   }
}

pub async fn check_gpo(l2_url: &str) -> Result<(), Box<dyn Error>> {
    info!("Constructing provider for {l2_url}...");
    // Initialize the provider.
    let provider = ProviderBuilder::new().connect(l2_url).await?;

    // Instantiate the contract instance.
    let gas_price_oracle_address = address!("0x420000000000000000000000000000000000000F");
    let gpo = GasPriceOracle::new(gas_price_oracle_address, provider);

    let is_jovian = gpo.isJovian().call().await?;
    info!("Calling GPO.isJovian()...");

    if is_jovian {
        info!("Gas Price Oracle is Jovian");
        Ok(())
    } else {
        error!("Gas Price Oracle is not Jovian");
        Err("Gas Price Oracle is not Jovian".into())
    }
}
