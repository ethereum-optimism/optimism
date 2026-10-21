//! `spn-wallet`: check, fund, and bridge PROVE on the Succinct Prover Network (SPN).
//!
//! Built to fund devnet `op-zk-proposer` SPN requester accounts, whose keys live in KMS: `fund`
//! moves PROVE into any SPN account without the recipient signing anything.
//!
//! ```text
//! spn-wallet balance <ADDRESS>                       # SPN + (with L1_RPC_URL) mainnet balance
//! spn-wallet deposit  --amount <PROVE>               # mainnet -> signer's SPN account
//! spn-wallet withdraw --amount <PROVE>               # signer's SPN account -> mainnet
//! spn-wallet fund --to <ADDRESS> --amount <PROVE>    # SPN transfer; --from-l1 deposits first
//! ```
//!
//! The private key for signing commands is read only from the `PRIVATE_KEY` env var, so it never
//! appears in shell history or process command lines.
//!
//! This crate lives under `kona/sp1/crates` only temporarily, next to `op-zk-proposer`; it has no
//! dependents and can move freely.

mod l1;
mod spn;

use std::{str::FromStr, time::Duration};

use alloy_primitives::{
    Address, U256,
    utils::{ParseUnits, format_units, parse_units},
};
use alloy_signer_local::PrivateKeySigner;
use anyhow::{Context, Result, anyhow, bail, ensure};
use clap::{Parser, Subcommand};
use url::Url;

use spn::Spn;

const PRIVATE_KEY_ENV: &str = "PRIVATE_KEY";
const POLL_INTERVAL: Duration = Duration::from_secs(10);
const DEPOSIT_TIMEOUT: Duration = Duration::from_secs(15 * 60);
const TRANSFER_TIMEOUT: Duration = Duration::from_secs(5 * 60);

#[derive(Parser)]
#[command(
    version,
    about,
    after_help = "Signing commands read the private key from the PRIVATE_KEY env var."
)]
struct Cli {
    /// Ethereum mainnet RPC URL (needed for L1 balances and deposits).
    #[arg(long, env = "L1_RPC_URL", global = true)]
    l1_rpc: Option<Url>,
    /// SPN RPC URL.
    #[arg(long, env = "SPN_RPC_URL", global = true, default_value = spn::MAINNET_SPN_RPC)]
    spn_rpc: String,
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Show the PROVE balance of an address on SPN and (with --l1-rpc) on Ethereum mainnet.
    Balance { address: Address },
    /// Bridge PROVE from Ethereum mainnet into the signer's own SPN account.
    Deposit {
        /// Amount in PROVE (decimal, e.g. 1.5).
        #[arg(long, value_parser = parse_prove)]
        amount: U256,
    },
    /// Withdraw PROVE from the signer's SPN account back to the same address on mainnet.
    Withdraw {
        /// Amount in PROVE (decimal, e.g. 1.5).
        #[arg(long, value_parser = parse_prove)]
        amount: U256,
    },
    /// Fund any SPN account from the signer; the recipient does not need to do anything.
    Fund {
        /// Recipient SPN account.
        #[arg(long)]
        to: Address,
        /// Amount in PROVE (decimal, e.g. 1.5) the recipient receives.
        #[arg(long, value_parser = parse_prove)]
        amount: U256,
        /// Source the funds from the signer's mainnet PROVE: deposit amount + transfer fee into
        /// the signer's SPN account first, then transfer.
        #[arg(long)]
        from_l1: bool,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Command::Balance { address } => {
            let spn = Spn::connect(&cli.spn_rpc).await?;
            println!("address: {address}");
            println!("SPN: {}", fmt_prove(spn.balance(address).await?));
            match cli.l1_rpc {
                Some(url) => {
                    let provider = l1::read_provider(url);
                    println!("L1:  {}", fmt_prove(l1::prove_balance(&provider, address).await?));
                }
                None => println!("L1:  skipped (no --l1-rpc)"),
            }
        }
        Command::Deposit { amount } => {
            let signer = load_signer()?;
            let l1_rpc = require_l1_rpc(cli.l1_rpc)?;
            let spn = Spn::connect(&cli.spn_rpc).await?;
            deposit_and_wait(&spn, l1_rpc, signer, amount).await?;
        }
        Command::Withdraw { amount } => {
            let signer = load_signer()?;
            let spn = Spn::connect(&cli.spn_rpc).await?;
            let params = spn.withdraw_params().await?;
            let total = amount + params.fee;
            let balance = spn.balance(signer.address()).await?;
            ensure!(
                balance >= total,
                "SPN balance of {} is {}, need {} (amount + withdraw fee {})",
                signer.address(),
                fmt_prove(balance),
                fmt_prove(total),
                fmt_prove(params.fee)
            );
            let hash = spn.withdraw(&signer, amount, &params).await?;
            println!("withdraw submitted: {hash}");
            println!("amount: {}", fmt_prove(amount));
            println!("fee:    {}", fmt_prove(params.fee));
            println!(
                "{} is paid out to {} on L1 when the auctioneer settles the next SPN block; \
                 check with `spn-wallet balance`.",
                fmt_prove(amount),
                signer.address()
            );
        }
        Command::Fund { to, amount, from_l1 } => {
            let signer = load_signer()?;
            let l1_rpc = if from_l1 { Some(require_l1_rpc(cli.l1_rpc)?) } else { None };
            let spn = Spn::connect(&cli.spn_rpc).await?;
            let params = spn.transfer_params().await?;
            let total = amount + params.fee;
            if let Some(l1_rpc) = l1_rpc {
                deposit_and_wait(&spn, l1_rpc, signer.clone(), total).await?;
            }
            let balance = spn.balance(signer.address()).await?;
            ensure!(
                balance >= total,
                "SPN balance of {} is {}, need {} (amount + transfer fee {})",
                signer.address(),
                fmt_prove(balance),
                fmt_prove(total),
                fmt_prove(params.fee)
            );
            let before = spn.balance(to).await?;
            let hash = spn.transfer(&signer, to, amount, &params).await?;
            println!("transfer submitted: {hash} (fee {})", fmt_prove(params.fee));
            let Some(now) =
                wait_for_spn_balance(&spn, to, before + amount, TRANSFER_TIMEOUT).await?
            else {
                bail!(
                    "transfer {hash} was submitted but the SPN balance of {to} has not grown by {} \
                     after {}m; recheck with `spn-wallet balance {to}`",
                    fmt_prove(amount),
                    TRANSFER_TIMEOUT.as_secs() / 60
                );
            };
            println!("{to} SPN balance: {}", fmt_prove(now));
        }
    }
    Ok(())
}

/// Deposits `amount` from L1 and waits until SPN credits the signer's account.
async fn deposit_and_wait(
    spn: &Spn,
    l1_rpc: Url,
    signer: PrivateKeySigner,
    amount: U256,
) -> Result<()> {
    let from = signer.address();
    let before = spn.balance(from).await?;
    l1::deposit(&l1::signing_provider(l1_rpc, signer), from, amount).await?;
    println!("waiting for SPN to credit {} to {from}...", fmt_prove(amount));
    let Some(now) = wait_for_spn_balance(spn, from, before + amount, DEPOSIT_TIMEOUT).await? else {
        bail!(
            "the L1 deposit succeeded but SPN has not credited {from} after {}m; the credit is \
             still pending, recheck with `spn-wallet balance {from}`",
            DEPOSIT_TIMEOUT.as_secs() / 60
        );
    };
    println!("{from} SPN balance: {}", fmt_prove(now));
    Ok(())
}

/// Polls until the SPN balance of `address` reaches `target`; `None` on timeout.
async fn wait_for_spn_balance(
    spn: &Spn,
    address: Address,
    target: U256,
    timeout: Duration,
) -> Result<Option<U256>> {
    let deadline = tokio::time::Instant::now() + timeout;
    loop {
        let balance = spn.balance(address).await?;
        if balance >= target {
            return Ok(Some(balance));
        }
        if tokio::time::Instant::now() >= deadline {
            return Ok(None);
        }
        tokio::time::sleep(POLL_INTERVAL).await;
    }
}

fn load_signer() -> Result<PrivateKeySigner> {
    let key = std::env::var(PRIVATE_KEY_ENV).unwrap_or_default();
    if key.trim().is_empty() {
        bail!("{PRIVATE_KEY_ENV} env var is required for this command");
    }
    // Do not surface the parse error: it must never risk echoing key material.
    let signer = PrivateKeySigner::from_str(key.trim())
        .map_err(|_| anyhow!("{PRIVATE_KEY_ENV} is not a valid hex private key"))?;
    println!("signer: {}", signer.address());
    Ok(signer)
}

fn require_l1_rpc(url: Option<Url>) -> Result<Url> {
    url.context("--l1-rpc (or L1_RPC_URL) is required for this command")
}

fn parse_prove(s: &str) -> Result<U256, String> {
    match parse_units(s, 18).map_err(|e| e.to_string())? {
        ParseUnits::U256(v) => Ok(v),
        ParseUnits::I256(_) => Err("amount must not be negative".into()),
    }
}

pub(crate) fn fmt_prove(wei: U256) -> String {
    let units = format_units(wei, 18).expect("18 decimals is always valid");
    let units = units.trim_end_matches('0').trim_end_matches('.');
    format!("{units} PROVE ({wei} wei)")
}
