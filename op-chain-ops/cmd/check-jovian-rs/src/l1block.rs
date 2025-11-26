use alloy::{network::Ethereum, primitives::address, providers::Provider, sol};
use log::{error, info};
use std::error::Error;

// Generate the contract bindings for the ERC20 interface.
sol! {
   // The `rpc` attribute enables contract interaction via the provider.
   #[sol(rpc)]
   contract L1Block {
        function daFootprintGasScalar() public view returns (uint16);
   }
}

pub async fn check<T: Provider<Ethereum>>(provider: &T) -> Result<(), Box<dyn Error>> {
    // Instantiate the contract instance.
    let l1block_address = address!("0x4200000000000000000000000000000000000015");
    let l1block = L1Block::new(l1block_address, provider);

    info!("Calling L1Block.DaFootprintGasScalar()...");
    let scalar = l1block.daFootprintGasScalar().call().await;
    match scalar {
        Ok(scalar) => info!("L1Block.DaFootprintGasScalar() returned {}", scalar),
        Err(e) => error!(
            "Error calling L1Block.DaFootprintGasScalar() (network is likely not yet upgraded): {}",
            e
        ),
    }
    Ok(())
}
