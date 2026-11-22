//! `spn-wallet`: fund and inspect PROVE accounts on the Succinct Prover Network (SPN).
//!
//! Built to fund `op-zk-proposer` SPN requesters. Those keys live in KMS behind op-signer and can
//! sign only SPN messages, so `fund` moves PROVE into any SPN account without the recipient
//! signing anything.
//!
//! ```text
//! spn-wallet balance  <ADDRESS>                                # SPN + (with L1_RPC_URL) L1 PROVE
//! spn-wallet deposit  --amount <PROVE>                         # L1 -> signer's SPN account
//! spn-wallet withdraw --amount <PROVE> --max-fee <PROVE>       # signer's SPN account -> L1
//! spn-wallet fund --to <ADDRESS> --amount <PROVE> --max-fee <PROVE> [--from-l1]
//! spn-wallet history <ADDRESS> | requests <ADDRESS> | request <ID> [--bids] | transaction <HASH>
//! ```
//!
//! Write commands print what they would send and sign nothing unless `--broadcast` is set. They
//! sign with the `PRIVATE_KEY` env var or a GCP Cloud KMS key (`--kms-key`), never both. The
//! PROVE side is always Ethereum mainnet, also for devnets that settle on Sepolia.

mod l1;
mod plan;
mod signer;
mod spn;

use std::time::{Duration, UNIX_EPOCH};

use alloy_primitives::{
    Address, B256, I256, U256,
    utils::{ParseUnits, Unit, parse_units},
};
use alloy_provider::DynProvider;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, bail};
use clap::{Args, Parser, Subcommand};
use prost::Message;
use sp1_sdk::network::proto::types::{
    BalanceOperation, ExecutionStatus, FulfillmentStatus, ProofRequest,
};

use plan::{Deposit, SpnQuote, Step};
use signer::Funder;
use spn::{Spn, TxParams};

const DEPOSIT_TIMEOUT: Duration = Duration::from_secs(15 * 60);
const TRANSFER_TIMEOUT: Duration = Duration::from_secs(5 * 60);
const BIDS_SHOWN: u32 = 20;

#[derive(Parser)]
#[command(
    version,
    about,
    after_help = "Signing commands read the private key from the PRIVATE_KEY env var, or use \
                  --kms-key (KMS_KEY) with gcloud Application Default Credentials."
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

#[derive(Args)]
struct Signing {
    /// GCP Cloud KMS key version to sign with instead of `PRIVATE_KEY`, in op-signer's `key:`
    /// format: projects/P/locations/L/keyRings/R/cryptoKeys/K/cryptoKeyVersions/V.
    #[arg(long, env = "KMS_KEY")]
    kms_key: Option<String>,
    /// Sign and send. Without it the command prints what it would send and signs nothing.
    #[arg(long)]
    broadcast: bool,
}

#[derive(Args)]
struct Page {
    /// Rows per page.
    #[arg(long, default_value_t = 20, value_parser = clap::value_parser!(u32).range(1..=i64::from(spn::MAX_PAGE)))]
    limit: u32,
    /// Page number, starting at 1.
    #[arg(long, default_value_t = 1, value_parser = clap::value_parser!(u32).range(1..))]
    page: u32,
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
        #[command(flatten)]
        signing: Signing,
    },
    /// Withdraw PROVE from the signer's SPN account back to the same address on mainnet.
    Withdraw {
        /// Amount in PROVE (decimal, e.g. 1.5).
        #[arg(long, value_parser = parse_prove)]
        amount: U256,
        /// Highest SPN fee, in PROVE, to accept. Required with --broadcast.
        #[arg(long, value_parser = parse_prove, required_if_eq("broadcast", "true"))]
        max_fee: Option<U256>,
        #[command(flatten)]
        signing: Signing,
    },
    /// Fund any SPN account from the signer; the recipient does not need to do anything.
    Fund {
        /// Recipient SPN account.
        #[arg(long)]
        to: Address,
        /// Amount in PROVE (decimal, e.g. 1.5) the recipient receives.
        #[arg(long, value_parser = parse_prove)]
        amount: U256,
        /// Highest SPN transfer fee, in PROVE, to accept. Required with --broadcast.
        #[arg(long, value_parser = parse_prove, required_if_eq("broadcast", "true"))]
        max_fee: Option<U256>,
        /// Source the funds from the signer's mainnet PROVE: deposit amount + transfer fee into
        /// the signer's SPN account first, then transfer.
        #[arg(long)]
        from_l1: bool,
        #[command(flatten)]
        signing: Signing,
    },
    /// List an address's SPN balance changes, newest first.
    History {
        address: Address,
        #[command(flatten)]
        page: Page,
    },
    /// List proof requests made by an SPN requester, newest first.
    Requests {
        requester: Address,
        /// Only requests in this fulfillment status (REQUESTED, ASSIGNED, FULFILLED,
        /// UNFULFILLABLE, REVERTED, EXPIRED).
        #[arg(long, value_parser = parse_status)]
        status: Option<FulfillmentStatus>,
        #[command(flatten)]
        page: Page,
    },
    /// Show one proof request.
    Request {
        id: B256,
        /// Also list the latest bids.
        #[arg(long)]
        bids: bool,
    },
    /// Show an SPN transaction. SPN accepted it if it is found; `history` shows whether it
    /// applied.
    Transaction { hash: B256 },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    let spn = Spn::connect(&cli.spn_rpc).await?;
    match cli.command {
        Command::Balance { address } => balance(&spn, cli.l1_rpc, address).await,
        Command::Deposit { amount, signing } => {
            let funder = load_funder(&signing).await?;
            let wallet = signing.broadcast.then(|| funder.wallet());
            let l1 = l1::connect(require_l1_rpc(cli.l1_rpc)?, wallet).await?;
            let deposit = plan::deposit(amount, &l1::quote(&l1, funder.address()).await?)?;
            if !signing.broadcast {
                preview_deposit(&deposit);
                preview_done();
                return Ok(());
            }
            execute_deposit(&spn, &l1, &funder, &deposit, None).await
        }
        Command::Withdraw { amount, max_fee, signing } => {
            let funder = load_funder(&signing).await?;
            let params = spn.withdraw_params().await?;
            let domain = spn.domain().await?;
            let quote = spn_quote(&spn, &funder, &params).await?;
            plan::withdraw(amount, &quote, max_fee)?;
            let body = spn.withdraw_body(funder.address(), amount, &params, &domain).await?;
            println!("withdraw {} (fee {})", fmt_prove(amount), fmt_prove(params.fee));
            print_spn_tx(funder.address(), body.nonce, &params);
            if !signing.broadcast {
                preview_done();
                return Ok(());
            }
            let signature = funder.sign_message(&body.encode_to_vec()).await?;
            let hash = spn.submit_withdraw(funder.address(), body, signature).await?;
            println!("withdraw submitted: {hash}");
            println!(
                "{} is paid out to {} on L1 when the auctioneer settles the next SPN block; \
                 check with `spn-wallet history {}`.",
                fmt_prove(amount),
                funder.address(),
                funder.address()
            );
            Ok(())
        }
        Command::Fund { to, amount, max_fee, from_l1, signing } => {
            let funder = load_funder(&signing).await?;
            // Everything the transfer needs from SPN, and the fee cap, is settled before any L1
            // transaction; after a deposit only the nonce is read again.
            let params = spn.transfer_params().await?;
            let domain = spn.domain().await?;
            let quote = spn_quote(&spn, &funder, &params).await?;
            let l1 = match from_l1 {
                true => {
                    let wallet = signing.broadcast.then(|| funder.wallet());
                    let l1 = l1::connect(require_l1_rpc(cli.l1_rpc)?, wallet).await?;
                    let l1_quote = l1::quote(&l1, funder.address()).await?;
                    Some((l1, l1_quote))
                }
                false => None,
            };
            let steps =
                plan::fund(to, amount, &quote, max_fee, l1.as_ref().map(|(_, quote)| quote))?;
            // After a --from-l1 deposit lands, the PROVE sits in the signer's SPN account; this
            // command moves it on if the transfer is never sent.
            let resume = (from_l1 && signing.broadcast).then(|| {
                let mut cmd =
                    format!("spn-wallet fund --to {to} --amount {} --broadcast", fmt_units(amount));
                if let Some(cap) = max_fee {
                    cmd += &format!(" --max-fee {}", fmt_units(cap));
                }
                if let Some(key) = &signing.kms_key {
                    cmd += &format!(" --kms-key {key}");
                }
                if cli.spn_rpc != spn::MAINNET_SPN_RPC {
                    cmd += &format!(" --spn-rpc {}", cli.spn_rpc);
                }
                cmd
            });
            for step in &steps {
                match step {
                    Step::Deposit(deposit) if !signing.broadcast => preview_deposit(deposit),
                    Step::Deposit(deposit) => {
                        let (provider, _) =
                            l1.as_ref().expect("plan::fund deposits only with an L1 quote");
                        execute_deposit(&spn, provider, &funder, deposit, resume.as_deref())
                            .await?;
                    }
                    Step::Transfer { to, amount } => {
                        let tx = TransferTx {
                            to: *to,
                            amount: *amount,
                            params: &params,
                            domain: &domain,
                        };
                        transfer(&spn, &funder, &tx, signing.broadcast, resume.as_deref()).await?;
                    }
                }
            }
            if !signing.broadcast {
                preview_done();
            }
            Ok(())
        }
        Command::History { address, page } => {
            for log in spn.balance_logs(address, None, None, page.limit, page.page).await? {
                let operation = operation_name(log.operation);
                println!(
                    "{}  {operation:<16}  {:>30}  tx {}",
                    fmt_time(log.created_at),
                    fmt_signed_prove(&log.amount)?,
                    B256::try_from(log.tx_hash.as_slice()).context("SPN returned a bad tx hash")?
                );
            }
            Ok(())
        }
        Command::Requests { requester, status, page } => {
            for request in spn.proof_requests(requester, status, page.limit, page.page).await? {
                print_request_line(&request)?;
            }
            Ok(())
        }
        Command::Request { id, bids } => {
            let Some(request) = spn.proof_request(id).await? else {
                bail!("SPN has no proof request {id}");
            };
            print_request(&request)?;
            if bids {
                println!("bids (latest {BIDS_SHOWN}):");
                for bid in spn.bids(id, BIDS_SHOWN).await? {
                    println!(
                        "  {}  {}  {}{}",
                        fmt_time(bid.created_at),
                        Address::try_from(bid.bidder.as_slice())
                            .context("SPN returned a bad bidder")?,
                        fmt_prove(spn::parse_amount(&bid.amount)?),
                        bid.bidder_name.map(|name| format!("  ({name})")).unwrap_or_default()
                    );
                }
            }
            Ok(())
        }
        Command::Transaction { hash } => {
            let Some(tx) = spn.transaction(hash).await? else {
                bail!("SPN has no transaction {hash}");
            };
            println!("tx:      {hash}");
            println!(
                "sender:  {}",
                Address::try_from(tx.sender.as_slice()).context("SPN returned a bad sender")?
            );
            println!("nonce:   {}", tx.nonce);
            println!("created: {}", fmt_time(tx.created_at));
            if let Some(name) = tx.name {
                println!("name:    {name}");
            }
            if let Some(id) = tx.request_id {
                println!("request: 0x{}", alloy_primitives::hex::encode(id));
            }
            println!(
                "SPN accepted this transaction; `spn-wallet history` shows whether it applied."
            );
            Ok(())
        }
    }
}

async fn balance(spn: &Spn, l1_rpc: Option<Url>, address: Address) -> Result<()> {
    println!("address: {address}");
    println!("SPN: {}", fmt_prove(spn.balance(address).await?));
    match l1_rpc {
        Some(url) => {
            let provider = l1::connect(url, None).await?;
            println!("L1:  {}", fmt_prove(l1::prove_balance(&provider, address).await?));
        }
        None => println!("L1:  skipped (no --l1-rpc)"),
    }
    Ok(())
}

async fn load_funder(signing: &Signing) -> Result<Funder> {
    let funder = Funder::load(signing.kms_key.as_deref()).await?;
    match &funder {
        Funder::Local(_) => println!("signer: {} (PRIVATE_KEY)", funder.address()),
        Funder::Kms(_) => println!("signer: {} (KMS)", funder.address()),
    }
    Ok(funder)
}

async fn spn_quote(spn: &Spn, funder: &Funder, params: &TxParams) -> Result<SpnQuote> {
    Ok(SpnQuote { fee: params.fee, balance: spn.balance(funder.address()).await? })
}

fn preview_deposit(&Deposit { amount, approve }: &Deposit) {
    println!("deposit {} from L1 into the signer's SPN account", fmt_prove(amount));
    if approve {
        println!("  tx to PROVE {}: {}", l1::PROVE, l1::approve_calldata(amount));
    }
    println!("  tx to VApp  {}: {}", l1::VAPP, l1::deposit_calldata(amount));
}

/// Sends the deposit and waits until SPN credits the signer. Nothing else spends from the
/// signer's account meanwhile, so a balance delta confirms the credit.
/// `resume` is the command that moves the credited PROVE on when this deposit is the first step
/// of `fund --from-l1`.
async fn execute_deposit(
    spn: &Spn,
    provider: &DynProvider,
    funder: &Funder,
    &Deposit { amount, approve }: &Deposit,
    resume: Option<&str>,
) -> Result<()> {
    let from = funder.address();
    let before = spn.balance(from).await?;
    l1::deposit(provider, amount, approve).await?;
    let pending = || {
        format!(
            "the L1 deposit succeeded but SPN has not confirmed crediting {from}; recheck with \
             `spn-wallet balance {from}`{}",
            resume.map(|cmd| format!(", then finish with `{cmd}`")).unwrap_or_default()
        )
    };
    println!("waiting for SPN to credit {} to {from}...", fmt_prove(amount));
    let credited =
        spn.wait_for_balance(from, before + amount, DEPOSIT_TIMEOUT).await.with_context(pending)?;
    let Some(now) = credited else {
        bail!("{} (waited {}m)", pending(), DEPOSIT_TIMEOUT.as_secs() / 60);
    };
    println!("{from} SPN balance: {}", fmt_prove(now));
    Ok(())
}

/// What a transfer sends: recipient, amount, and the SPN fee parameters and domain it is signed
/// under.
struct TransferTx<'a> {
    to: Address,
    amount: U256,
    params: &'a TxParams,
    domain: &'a [u8],
}

/// Prints the transfer; with `broadcast`, signs, submits, and waits for the recipient's
/// `TRANSFER_IN` log. `resume` is added to failures before submission, when the PROVE already sits
/// in the signer's SPN account.
async fn transfer(
    spn: &Spn,
    funder: &Funder,
    tx: &TransferTx<'_>,
    broadcast: bool,
    resume: Option<&str>,
) -> Result<()> {
    let TransferTx { to, amount, params, domain } = *tx;
    let from = funder.address();
    let unsent = |err: anyhow::Error| match resume {
        Some(cmd) => err.context(format!(
            "the transfer was not sent and the PROVE is in {from}'s SPN account; finish with \
             `{cmd}`"
        )),
        None => err,
    };
    let body = spn.transfer_body(from, to, amount, params, domain).await.map_err(unsent)?;
    println!("transfer {} to {to} (fee {})", fmt_prove(amount), fmt_prove(params.fee));
    print_spn_tx(from, body.nonce, params);
    if !broadcast {
        return Ok(());
    }
    let signature = funder.sign_message(&body.encode_to_vec()).await.map_err(unsent)?;
    let submitted_at = spn::unix_now();
    let hash = spn.submit_transfer(from, body, signature).await?;
    println!("transfer submitted: {hash}");
    let landed = spn
        .wait_for_transfer_in(to, hash, submitted_at, TRANSFER_TIMEOUT)
        .await
        .with_context(|| {
            format!(
                "transfer {hash} was submitted; check `spn-wallet history {to}` before sending \
                 again"
            )
        })?;
    if !landed {
        bail!(
            "transfer {hash} was submitted but {to} has no TRANSFER_IN for it after {}m; check \
             `spn-wallet transaction {hash}` and `spn-wallet history {to}`",
            TRANSFER_TIMEOUT.as_secs() / 60
        );
    }
    // The transfer has landed; a failed balance read must not turn that into an error.
    let balance = spn.balance(to).await.map_or_else(|_| "?".to_string(), fmt_prove);
    println!("{to} received {}; SPN balance: {balance}", fmt_prove(amount));
    Ok(())
}

fn print_spn_tx(from: Address, nonce: u64, params: &TxParams) {
    println!("  from {from}, nonce {nonce}, auctioneer {}", params.auctioneer);
}

fn preview_done() {
    println!("preview only: nothing was signed; rerun with --broadcast to send");
}

fn print_request_line(request: &ProofRequest) -> Result<()> {
    println!(
        "{}  {}  {:<13}  {:<17}  cost {}",
        fmt_time(request.created_at),
        B256::try_from(request.request_id.as_slice()).context("SPN returned a bad request id")?,
        fulfillment_name(request.fulfillment_status),
        execution_name(request.execution_status),
        request
            .deduction_amount
            .as_deref()
            .map_or(Ok("-".into()), |a| spn::parse_amount(a).map(fmt_prove))?
    );
    Ok(())
}

fn print_request(request: &ProofRequest) -> Result<()> {
    let id =
        B256::try_from(request.request_id.as_slice()).context("SPN returned a bad request id")?;
    println!("request:     {id}");
    println!(
        "requester:   {}",
        Address::try_from(request.requester.as_slice()).context("SPN returned a bad requester")?
    );
    println!("fulfillment: {}", fulfillment_name(request.fulfillment_status));
    println!("execution:   {}", execution_name(request.execution_status));
    println!("created:     {}", fmt_time(request.created_at));
    println!("deadline:    {}", fmt_time(request.deadline));
    if let Some(fulfiller) = &request.fulfiller {
        println!(
            "fulfiller:   {}",
            Address::try_from(fulfiller.as_slice()).context("SPN returned a bad fulfiller")?
        );
    }
    if let Some(gas_used) = request.gas_used {
        println!("gas used:    {gas_used} of {}", request.gas_limit);
    }
    if let Some(cost) = &request.deduction_amount {
        println!("cost:        {}", fmt_prove(spn::parse_amount(cost)?));
    }
    Ok(())
}

fn fulfillment_name(status: i32) -> String {
    FulfillmentStatus::try_from(status)
        .map_or_else(|_| format!("UNKNOWN({status})"), |s| s.as_str_name().to_string())
}

fn execution_name(status: i32) -> String {
    ExecutionStatus::try_from(status)
        .map_or_else(|_| format!("UNKNOWN({status})"), |s| s.as_str_name().to_string())
}

fn operation_name(operation: i32) -> String {
    BalanceOperation::try_from(operation)
        .map_or_else(|_| format!("UNKNOWN({operation})"), |op| op.as_str_name().to_string())
}

fn require_l1_rpc(url: Option<Url>) -> Result<Url> {
    url.context("--l1-rpc (or L1_RPC_URL) is required for this command")
}

/// Parses decimal PROVE into wei. `parse_units` drops digits past the 18th decimal, so those
/// are rejected rather than silently rounded.
fn parse_prove(s: &str) -> Result<U256, String> {
    if s.split_once('.').is_some_and(|(_, fraction)| fraction.len() > 18) {
        return Err(format!("{s} has more than 18 decimals, PROVE's smallest unit"));
    }
    match parse_units(s, 18).map_err(|e| e.to_string())? {
        ParseUnits::U256(v) => Ok(v),
        ParseUnits::I256(_) => Err("amount must not be negative".into()),
    }
}

fn parse_status(s: &str) -> Result<FulfillmentStatus, String> {
    FulfillmentStatus::from_str_name(&s.to_ascii_uppercase())
        .ok_or_else(|| format!("unknown fulfillment status {s:?}"))
}

/// Decimal PROVE without trailing zeros, as `parse_prove` accepts it.
fn fmt_units(wei: impl Into<ParseUnits>) -> String {
    let units = wei.into().format_units(Unit::ETHER);
    units.trim_end_matches('0').trim_end_matches('.').to_string()
}

pub(crate) fn fmt_prove(wei: U256) -> String {
    format!("{} PROVE ({wei} wei)", fmt_units(wei))
}

/// Balance-log amounts are signed: SPN records debits as negative values.
fn fmt_signed_prove(amount: &str) -> Result<String> {
    let wei = I256::from_dec_str(amount)
        .with_context(|| format!("SPN returned an invalid amount {amount:?}"))?;
    Ok(format!("{} PROVE", fmt_units(wei)))
}

fn fmt_time(unix_secs: u64) -> String {
    humantime::format_rfc3339_seconds(UNIX_EPOCH + Duration::from_secs(unix_secs)).to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_prove_is_exact_to_the_wei() {
        assert_eq!(parse_prove("1.5"), Ok(U256::from(1_500_000_000_000_000_000u128)));
        assert_eq!(parse_prove("0.000000000000000001"), Ok(U256::from(1)));
        assert!(parse_prove("1.0000000000000000001").is_err());
        assert!(parse_prove("-1").is_err());
    }
}
