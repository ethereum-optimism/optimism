//! Private projection relation runner: local fixtures, `--print-vkey`, and the v2 publication
//! request (`--publication-request`, spec-sound-profile §G.2). Fixture modes never submit a
//! network proof request.
use anyhow::{Context, Result, anyhow, ensure};
use clap::{Parser, ValueEnum};
use kona_genesis::RollupConfig;
use kona_protocol::{
    SpanBatch, SpanBatchElement,
    projection::{self, ProofVerifier, Sp1Verifier},
};
use kona_sp1_client_utils::private_projection::{
    PUBLIC_VALUES_LEN, PublicValues, RelationInput, Witness, execute,
};
use sp1_sdk::{Elf, Prover, ProverClient, SP1Stdin};
use std::{path::PathBuf, sync::Arc};

mod private_projection_host;
use private_projection_host::{ProverKind, envelope, program_vkey};

#[derive(Debug, Clone, Copy, ValueEnum)]
enum Mode {
    /// Native relation only.
    Native,
    /// Native relation, then an SP1 mock envelope (SP1 mock prover if the ELF exists, else the
    /// native mock envelope), checked with `kona_protocol`'s `Sp1Verifier`.
    Mock,
    /// SP1 executor on the guest ELF, no proof.
    Execute,
    /// Local CPU Groth16 proof (heavy; not run in CI).
    Prove,
}

#[derive(Debug, Parser)]
struct Args {
    /// Local fixture tuple: relation input, witness, admission-computed public values.
    #[arg(long)]
    fixture: Option<PathBuf>,
    /// Serve a v2 publication request from stdin; emit only its envelope on stdout.
    #[arg(long, conflicts_with_all = ["fixture", "print_vkey"])]
    publication_request: bool,
    /// Print the bytes32 program vkey (`program_vkey`) of `--elf` and exit.
    #[arg(long, requires = "elf", conflicts_with = "fixture")]
    print_vkey: bool,
    /// Fixture mode.
    #[arg(long, value_enum, default_value = "native")]
    mode: Mode,
    /// Private-projection guest ELF (execute/prove; mock when present; publication default is
    /// `$KONA_SP1_ELF_DIR/private-projection-elf`).
    #[arg(long)]
    elf: Option<PathBuf>,
    /// Also run corrupted inputs (execute: inside the guest) and require rejection.
    #[arg(long)]
    check_rejection: bool,
    /// Debugging only: on a relation failure, write the FULL PRIVATE WITNESS (private
    /// transactions, state and private data) in plaintext to
    /// `$KONA_SP1_PRIVATE_PROJECTION_DUMP_DIR`. Without this flag the variable is ignored.
    #[arg(long, requires = "publication_request")]
    allow_witness_dump: bool,
}

/// Admission's statement over a fixture's public data, independent of `execute`.
fn admission_statement(input: &RelationInput) -> Result<(RollupConfig, projection::Statement)> {
    let cfg: RollupConfig = serde_json::from_slice(&input.projection_config)?;
    let span = SpanBatch {
        batches: input
            .blocks
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b.timestamp,
                epoch_num: b.epoch,
                transactions: b.transactions.clone(),
            })
            .collect(),
        ..Default::default()
    };
    let recovery_hash = input.recovery.iter().rev().fold(Default::default(), |h, b| {
        use alloy_eips::Encodable2718;
        let txs: Vec<_> = b.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect();
        projection::recovery_step(h, b, &txs)
    });
    let statement = projection::validate_projection_range(
        &cfg,
        projection::ProjectionContext {
            parent_hash: input.parent_hash,
            l1_head: input.l1_head,
            continuation: projection::Continuation {
                anchor: input.anchor,
                output_root: input.anchor_output,
                recovery_hash,
            },
        },
        &span,
        &projection::StubVerifier,
    )?;
    Ok((cfg, statement))
}

/// Corruptions every relation must reject: a wrong anchor, an L1 witness not hanging off
/// `l1Head`, and an omitted export replay.
fn rejections(
    input: &RelationInput,
    witness: &Witness,
) -> Vec<(&'static str, RelationInput, Witness)> {
    let mut anchor = witness.clone();
    anchor.anchor_header.state_root.0[0] ^= 1;
    let mut l1 = witness.clone();
    l1.preimages.remove(&input.l1_head);
    let mut omitted = input.clone();
    omitted.blocks[0].transactions.remove(2);
    vec![
        ("private anchor output", input.clone(), anchor),
        ("L1 header chain from l1Head", input.clone(), l1),
        ("projection messages differ", omitted, witness.clone()),
    ]
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    if args.print_vkey {
        let elf = args.elf.as_deref().expect("clap requires --elf");
        println!("{}", program_vkey(elf).await?);
        return Ok(());
    }
    // Fixed process initialization, matching the guest and existing range executor.
    ensure!(
        revm::precompile::install_crypto(
            kona_sp1_client_utils::precompiles::CustomCrypto::default()
        ),
        "crypto backend already initialized"
    );
    if args.publication_request {
        return private_projection_host::publish(args.elf.as_deref(), args.allow_witness_dump)
            .await;
    }
    let path = args
        .fixture
        .ok_or_else(|| anyhow!("--fixture, --publication-request or --print-vkey required"))?;
    let (input, witness, expected): (RelationInput, Witness, alloy_primitives::Bytes) =
        serde_json::from_slice(&std::fs::read(&path)?)
            .with_context(|| format!("fixture {}", path.display()))?;
    let expected: PublicValues = expected
        .as_ref()
        .try_into()
        .map_err(|_| anyhow!("fixture public values must be {PUBLIC_VALUES_LEN} bytes"))?;
    let (cfg, statement) = admission_statement(&input)?;
    ensure!(
        projection::public_values(&statement) == expected,
        "fixture public values are not admission's"
    );
    let native = execute(&input, &witness)?;
    ensure!(native == expected, "native public values differ from admission's");
    if args.check_rejection {
        for (reason, bad_input, bad_witness) in rejections(&input, &witness) {
            let err = execute(&bad_input, &bad_witness).err().map(|e| format!("{e:#}"));
            ensure!(
                err.as_deref().is_some_and(|e| e.contains(reason)),
                "native relation did not reject ({reason}): {err:?}"
            );
        }
    }
    let payload = serde_json::to_vec(&(&input, &witness))?;
    match args.mode {
        Mode::Native => println!("native private projection execution passed"),
        Mode::Mock => {
            let mut profile = cfg.private_projection.clone().expect("validated above");
            let elf = args.elf.filter(|p| p.exists());
            let (kind, env) = if let Some(elf) = elf.as_deref() {
                profile.program_vkey = program_vkey(elf).await?;
                let env =
                    envelope(ProverKind::Mock, profile.program_vkey, Some(elf), payload, &native)
                        .await?;
                ("SP1 mock prover", env)
            } else {
                let env =
                    envelope(ProverKind::NativeMock, profile.program_vkey, None, payload, &native)
                        .await?;
                ("native mock envelope (no ELF)", env)
            };
            Sp1Verifier { profile: &profile, allow_mock: true }
                .verify(&statement, &env)
                .map_err(|e| anyhow!("Sp1Verifier rejected the mock envelope: {e}"))?;
            ensure!(
                Sp1Verifier { profile: &profile, allow_mock: false }
                    .verify(&statement, &env)
                    .is_err(),
                "Sp1Verifier accepted a mock envelope with mock proofs disabled"
            );
            let mut forged = env.clone();
            let messages_root = forged.len() - PUBLIC_VALUES_LEN + 19 * 32;
            forged[messages_root] ^= 1;
            ensure!(
                Sp1Verifier { profile: &profile, allow_mock: true }
                    .verify(&statement, &forged)
                    .is_err(),
                "Sp1Verifier accepted a forged messagesRoot"
            );
            println!(
                "{kind}: {}-byte envelope accepted by Sp1Verifier with mock proofs enabled, \
                 rejected with them disabled and with a forged word",
                env.len()
            );
        }
        Mode::Execute | Mode::Prove => {
            let path = args.elf.ok_or_else(|| anyhow::anyhow!("--elf is required"))?;
            let bytes = std::fs::read(&path)?;
            let elf = Elf::Dynamic(Arc::from(bytes.clone()));
            // Explicit CPU backend: SP1_PROVER/network credentials cannot select paid proving.
            let client = ProverClient::builder().cpu().build().await;
            let mut stdin = SP1Stdin::new();
            stdin.write_vec(payload.clone());
            let (values, report) = client.execute(elf.clone(), stdin).await?;
            eprintln!(
                "guest cycles: {} (tracked: {:?})",
                report.total_instruction_count(),
                report.cycle_tracker
            );
            ensure!(report.exit_code == 0, "guest execution exited unsuccessfully");
            ensure!(values.as_slice() == native.as_slice(), "SP1 public values differ from native");
            if matches!(args.mode, Mode::Prove) {
                let vkey = program_vkey(&path).await?;
                let env = envelope(ProverKind::Cpu, vkey, Some(&path), payload, &native).await?;
                println!("Groth16 envelope: {} bytes", env.len());
            }
            if args.check_rejection {
                for (reason, bad_input, bad_witness) in rejections(&input, &witness) {
                    let mut bad_stdin = SP1Stdin::new();
                    bad_stdin.write_vec(serde_json::to_vec(&(&bad_input, &bad_witness))?);
                    if let Ok((values, report)) =
                        client.execute(Elf::Dynamic(Arc::from(bytes.clone())), bad_stdin).await
                    {
                        ensure!(
                            report.exit_code != 0 && values.as_slice().is_empty(),
                            "guest accepted a corrupted input ({reason})"
                        );
                    }
                }
            }
            println!("SP1 {:?} passed; public values match native execution", args.mode);
        }
    }
    Ok(())
}
