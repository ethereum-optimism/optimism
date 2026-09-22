//! Local-only private projection relation runner. Never submits a network proof request.
use anyhow::{Result, ensure};
use clap::{Parser, ValueEnum};
use kona_sp1_client_utils::private_projection::{PublicInputs, PublicOutputs, Witness, execute};
use sp1_sdk::{Elf, Prover, ProverClient, ProvingKey, SP1PublicValues, SP1Stdin};
use std::{path::PathBuf, sync::Arc};

#[derive(Debug, Clone, Copy, ValueEnum)]
enum Mode {
    Native,
    Mock,
    Execute,
    Prove,
}

#[derive(Debug, Parser)]
struct Args {
    /// Local fixture tuple: public inputs, private witness, independently expected journal.
    #[arg(long)]
    fixture: PathBuf,
    /// Native by default; mock explicitly emits placeholder bytes after native validation.
    #[arg(long, value_enum, default_value = "native")]
    mode: Mode,
    /// Private-projection guest ELF, required for execute/prove.
    #[arg(long)]
    elf: Option<PathBuf>,
    /// Also execute a deliberately corrupted witness inside the guest and require rejection.
    #[arg(long)]
    check_rejection: bool,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    // Fixed process initialization, matching the guest and existing range executor.
    ensure!(
        revm::precompile::install_crypto(
            kona_sp1_client_utils::precompiles::CustomCrypto::default()
        ),
        "crypto backend already initialized"
    );
    let (inputs, witness, expected): (PublicInputs, Witness, PublicOutputs) =
        serde_json::from_slice(&std::fs::read(args.fixture)?)?;
    // Match existing mock lifecycle tests: real native execution precedes placeholder proof bytes.
    let native = execute(&inputs, &witness)?;
    ensure!(native == expected, "native journal differs from expected public inputs");
    match args.mode {
        Mode::Native => println!("native private projection execution passed"),
        Mode::Mock => {
            let proof = b"kona-sp1-mock-private-projection-proof-v1";
            println!(
                "native execution passed; mock proof only ({} placeholder bytes)",
                proof.len()
            );
        }
        Mode::Execute | Mode::Prove => {
            let path = args.elf.ok_or_else(|| anyhow::anyhow!("--elf is required"))?;
            let elf = Elf::Dynamic(Arc::from(std::fs::read(path)?));
            let rejection_elf = elf.clone();
            let mut stdin = SP1Stdin::new();
            stdin.write_vec(serde_json::to_vec(&(&inputs, &witness))?);
            // Explicit CPU backend: SP1_PROVER/network credentials cannot select paid proving.
            let client = ProverClient::builder().cpu().build().await;
            let values = if matches!(args.mode, Mode::Execute) {
                let (values, report) = client.execute(elf, stdin).await?;
                ensure!(report.exit_code == 0, "guest execution exited unsuccessfully");
                values
            } else {
                let pk = client.setup(elf).await?;
                let proof = client.prove(&pk, stdin).await?;
                client.verify(&proof, pk.verifying_key(), Some(sp1_sdk::StatusCode::SUCCESS))?;
                let mut tampered = proof.clone();
                let mut wrong = expected.clone();
                wrong.context_hash[0] ^= 1;
                tampered.public_values = SP1PublicValues::new();
                tampered.public_values.write(&wrong);
                ensure!(
                    client
                        .verify(&tampered, pk.verifying_key(), Some(sp1_sdk::StatusCode::SUCCESS))
                        .is_err(),
                    "verifier accepted altered public inputs"
                );
                proof.public_values
            };
            let mut expected_values = SP1PublicValues::new();
            expected_values.write(&expected);
            ensure!(
                values.as_slice() == expected_values.as_slice(),
                "SP1 journal differs from expected public inputs"
            );
            if args.check_rejection {
                let mut invalid = witness.clone();
                invalid.anchor_header.state_root[0] ^= 1;
                let mut bad_stdin = SP1Stdin::new();
                bad_stdin.write_vec(serde_json::to_vec(&(&inputs, &invalid))?);
                if let Ok((values, report)) = client.execute(rejection_elf, bad_stdin).await {
                    ensure!(
                        report.exit_code != 0 && values.as_slice().is_empty(),
                        "guest accepted corrupt private anchor"
                    );
                }
            }
            println!("SP1 {:?} passed; public journal matches native execution", args.mode);
        }
    }
    Ok(())
}
