//! Ethereum mainnet side: the PROVE token and the `SuccinctVApp` bridge contract.
//!
//! SPN settles on Ethereum mainnet for every chain it proves, including Sepolia devnets.

use alloy_network::{EthereumWallet, ReceiptResponse};
use alloy_primitives::{Address, Bytes, U256, address};
use alloy_provider::{DynProvider, Provider, ProviderBuilder};
use alloy_sol_types::{SolCall, sol};
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, ensure};

use crate::plan::L1Quote;

/// PROVE ERC-20 (succinctlabs/network `contracts/deployments/1.json`).
pub(crate) const PROVE: Address = address!("0x6BEF15D938d4E72056AC92Ea4bDD0D76B1C4ad29");
/// `SuccinctVApp` proxy: SPN's L1 settlement contract that holds deposited PROVE.
pub(crate) const VAPP: Address = address!("0x5Ad5Bc4B18f7c173DcE17A57682Cb0Dc8788951F");
const MAINNET_CHAIN_ID: u64 = 1;

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

/// Connects to the L1 RPC and checks that it serves Ethereum mainnet, where PROVE and the `VApp`
/// live. A Sepolia URL would send the approve and deposit to addresses with no code.
pub(crate) async fn connect(url: Url, wallet: Option<EthereumWallet>) -> Result<DynProvider> {
    let provider = match wallet {
        Some(wallet) => ProviderBuilder::new().wallet(wallet).connect_http(url).erased(),
        None => ProviderBuilder::new().connect_http(url).erased(),
    };
    let chain_id = provider.get_chain_id().await.context("L1 eth_chainId")?;
    ensure!(
        chain_id == MAINNET_CHAIN_ID,
        "L1 RPC serves chain {chain_id}; PROVE and the SPN bridge live on Ethereum mainnet \
         (chain 1), also for devnets that settle on Sepolia"
    );
    Ok(provider)
}

pub(crate) async fn prove_balance(provider: &DynProvider, account: Address) -> Result<U256> {
    IERC20::new(PROVE, provider).balanceOf(account).call().await.context("PROVE.balanceOf")
}

pub(crate) async fn quote(provider: &DynProvider, from: Address) -> Result<L1Quote> {
    let prove = IERC20::new(PROVE, provider);
    let vapp = ISuccinctVApp::new(VAPP, provider);
    Ok(L1Quote {
        balance: prove_balance(provider, from).await?,
        allowance: prove.allowance(from, VAPP).call().await.context("PROVE.allowance")?,
        min_deposit: vapp.minDepositAmount().call().await.context("VApp.minDepositAmount")?,
    })
}

pub(crate) fn approve_calldata(amount: U256) -> Bytes {
    IERC20::approveCall { spender: VAPP, amount }.abi_encode().into()
}

pub(crate) fn deposit_calldata(amount: U256) -> Bytes {
    ISuccinctVApp::depositCall { amount }.abi_encode().into()
}

/// Sends `PROVE.approve(VApp, amount)` when `approve` is set, then `VApp.deposit(amount)`, and
/// waits for both receipts. SPN credits the transaction sender; the `VApp` cannot deposit on behalf
/// of another address.
pub(crate) async fn deposit(provider: &DynProvider, amount: U256, approve: bool) -> Result<()> {
    if approve {
        let receipt = IERC20::new(PROVE, provider)
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
    let receipt = ISuccinctVApp::new(VAPP, provider)
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
