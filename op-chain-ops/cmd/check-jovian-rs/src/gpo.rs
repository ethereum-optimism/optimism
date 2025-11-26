use alloy::{network::Ethereum, primitives::address, providers::Provider, sol};
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

pub async fn check<T: Provider<Ethereum>>(provider: &T) -> Result<(), Box<dyn Error>> {
    // Instantiate the contract instance.
    let gas_price_oracle_address = address!("0x420000000000000000000000000000000000000F");
    let gpo = GasPriceOracle::new(gas_price_oracle_address, provider);

    info!("Calling GPO.isJovian()...");
    let is_jovian = gpo.isJovian().call().await;
    match is_jovian {
        Ok(true) => info!("Gas Price Oracle is Jovian"),
        Ok(false) => error!("Gas Price Oracle is not Jovian"),
        Err(e) => error!(
            "Error calling GPO.isJovian() (network is likely not yet upgraded): {}",
            e
        ),
    }
    Ok(())
}
