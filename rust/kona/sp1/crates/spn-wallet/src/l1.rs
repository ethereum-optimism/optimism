//! Ethereum mainnet side: the PROVE token and the `SuccinctVApp` bridge contract.

use alloy_network::ReceiptResponse;
use alloy_primitives::{Address, U256, address};
use alloy_provider::{DynProvider, Provider, ProviderBuilder};
use alloy_signer_local::PrivateKeySigner;
use alloy_sol_types::sol;
use anyhow::{Context, Result, ensure};
use url::Url;

use crate::fmt_prove;

/// PROVE ERC-20 (succinctlabs/network `contracts/deployments/1.json`).
pub(crate) const PROVE: Address = address!("0x6BEF15D938d4E72056AC92Ea4bDD0D76B1C4ad29");
/// `SuccinctVApp` proxy: SPN's L1 settlement contract that holds deposited PROVE.
pub(crate) const VAPP: Address = address!("0x5Ad5Bc4B18f7c173DcE17A57682Cb0Dc8788951F");

sol! {
    #[sol(rpc)]
    interface IERC20 {
        function balanceOf(address account) external view returns (uint256);
        function allowance(address owner, address spender) external view returns (uint256);
        function approve(address spender, uint256 amount) external returns (bool);
    }

    #[sol(rpc)]
    interface ISuccinctVApp {
        function deposit(uint256 amount) external returns (uint64 receipt);
        function minDepositAmount() external view returns (uint256);
    }
}

pub(crate) fn read_provider(url: Url) -> DynProvider {
    ProviderBuilder::new().connect_http(url).erased()
}

pub(crate) fn signing_provider(url: Url, signer: PrivateKeySigner) -> DynProvider {
    ProviderBuilder::new().wallet(signer).connect_http(url).erased()
}

pub(crate) async fn prove_balance(provider: &DynProvider, account: Address) -> Result<U256> {
    IERC20::new(PROVE, provider).balanceOf(account).call().await.context("PROVE.balanceOf")
}

/// Approves (if needed) and deposits `amount` PROVE into the `VApp`. SPN credits `from`, the
/// transaction sender; the contract has no way to deposit on behalf of another address.
pub(crate) async fn deposit(provider: &DynProvider, from: Address, amount: U256) -> Result<()> {
    let prove = IERC20::new(PROVE, provider);
    let vapp = ISuccinctVApp::new(VAPP, provider);

    let min = vapp.minDepositAmount().call().await.context("VApp.minDepositAmount")?;
    ensure!(
        amount >= min,
        "deposit of {} is below VApp.minDepositAmount ({})",
        fmt_prove(amount),
        fmt_prove(min)
    );
    let balance = prove_balance(provider, from).await?;
    ensure!(
        balance >= amount,
        "L1 PROVE balance of {from} is {}, need {}",
        fmt_prove(balance),
        fmt_prove(amount)
    );

    let allowance = prove.allowance(from, VAPP).call().await.context("PROVE.allowance")?;
    if allowance < amount {
        let receipt = prove
            .approve(VAPP, amount)
            .send()
            .await
            .context("sending PROVE.approve")?
            .get_receipt()
            .await
            .context("waiting for PROVE.approve receipt")?;
        ensure!(receipt.status(), "PROVE.approve reverted: {:?}", receipt.transaction_hash());
        println!("approve: {:?}", receipt.transaction_hash());
    }

    let receipt = vapp
        .deposit(amount)
        .send()
        .await
        .context("sending VApp.deposit")?
        .get_receipt()
        .await
        .context("waiting for VApp.deposit receipt")?;
    ensure!(receipt.status(), "VApp.deposit reverted: {:?}", receipt.transaction_hash());
    println!("deposit: {:?}", receipt.transaction_hash());
    Ok(())
}
