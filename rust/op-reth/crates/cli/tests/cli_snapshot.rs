//! Snapshot test of the complete op-reth CLI surface.
//!
//! Renders the full clap command tree (every subcommand, every argument, including hidden ones)
//! into a deterministic text form and compares it against the checked-in snapshot at
//! `tests/snapshots/cli.snap`. Any change to the CLI surface — a new upstream reth flag after a
//! pin bump, a renamed alias, a changed default — shows up as a diff in the snapshot and has to
//! be acknowledged by regenerating it:
//!
//! ```text
//! UPDATE_SNAPSHOT=1 cargo nextest run -p reth-optimism-cli --all-features cli_surface_snapshot
//! ```
//!
//! The rendering is hand-rolled (instead of `--help` output or a snapshot crate) so that it is
//! independent of terminal width, help-text wrapping, and extra dependencies. Machine-specific
//! content (platform-derived paths, build metadata embedded in defaults) is normalized to stable
//! placeholders, see [`normalize_default`]. Only clap's *short* help (`Arg::get_help`) is
//! rendered, never the long help: long help texts embed environment- and registry-derived
//! content (most notably `--chain`, whose long help lists every built-in superchain chain and
//! would churn on registry updates unrelated to the CLI surface).
//!
//! The test is gated on the `dev` feature because the `dev`-only `test-vectors` subcommand is
//! part of the rendered surface. CI runs the workspace test suite with `--all-features`
//! (the `rust-tests` CircleCI job), which enables it.
#![cfg(feature = "dev")]

use clap::CommandFactory;
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser};
use reth_optimism_node::args::RollupArgs;
use std::{env, fmt::Write as _, fs, path::Path};

/// Path of the checked-in snapshot file.
const SNAPSHOT_PATH: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/snapshots/cli.snap");

/// The binary name used as the root of every rendered subcommand path.
///
/// The real binary installs `op-reth` version metadata before parsing; in tests the clap command
/// still carries the upstream default name, so the root label is pinned here instead of read
/// from the command.
const ROOT_NAME: &str = "op-reth";

/// Long option names whose default values embed build- or machine-specific data
/// (client version strings, CPU counts). Their defaults are normalized to a placeholder.
const MACHINE_SPECIFIC_DEFAULTS: &[&str] =
    &["identity", "builder.extradata", "rpc.max-tracing-requests"];

/// Environment variables that point at machine-specific base directories. Any default value that
/// contains one of these paths is normalized to a placeholder.
const PATH_ENV_VARS: &[&str] =
    &["XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "HOME", "USERPROFILE"];

#[test]
fn cli_surface_snapshot() {
    let cmd = Cli::<OpChainSpecParser, RollupArgs>::command();
    let rendered = render_snapshot(&cmd);

    let path = Path::new(SNAPSHOT_PATH);
    if env::var("UPDATE_SNAPSHOT").is_ok_and(|v| v == "1") {
        fs::write(path, &rendered)
            .unwrap_or_else(|e| panic!("failed to write snapshot to {}: {e}", path.display()));
        println!("snapshot updated: {}", path.display());
        return;
    }

    let expected = fs::read_to_string(path).unwrap_or_default();
    if expected == rendered {
        return;
    }

    let (line_no, expected_line, actual_line) = first_diff_line(&expected, &rendered);

    // Print the full generated snapshot so it can be recovered from CI logs even when the
    // checked-in file is badly out of date (e.g. on the very first run after a reth bump).
    println!("----- BEGIN GENERATED CLI SNAPSHOT ({}) -----", SNAPSHOT_PATH);
    println!("{rendered}");
    println!("----- END GENERATED CLI SNAPSHOT -----");

    if let Some(tmpdir) = option_env!("CARGO_TARGET_TMPDIR") {
        let new_path = Path::new(tmpdir).join("cli.snap.new");
        if fs::write(&new_path, &rendered).is_ok() {
            println!("full generated snapshot also written to: {}", new_path.display());
        }
    }

    panic!(
        "op-reth CLI surface changed: snapshot mismatch at line {line_no}.\n\
         expected: {expected_line}\n\
         actual:   {actual_line}\n\
         \n\
         If this change is intentional, regenerate the snapshot with\n\
         \n\
         UPDATE_SNAPSHOT=1 cargo nextest run -p reth-optimism-cli --all-features cli_surface_snapshot\n\
         \n\
         or copy the block between the BEGIN/END markers in this test's stdout into\n\
         rust/op-reth/crates/cli/tests/snapshots/cli.snap."
    );
}

/// Renders the complete command tree into the snapshot text.
fn render_snapshot(cmd: &clap::Command) -> String {
    let mut out = String::new();
    out.push_str(
        "# op-reth CLI surface snapshot.\n\
         # One block per subcommand path; arguments in clap definition order, hidden ones\n\
         # included. Platform- and build-specific defaults are normalized to placeholders.\n\
         # Regenerate with:\n\
         #   UPDATE_SNAPSHOT=1 cargo nextest run -p reth-optimism-cli --all-features cli_surface_snapshot\n\
         \n",
    );
    render_command(&mut out, ROOT_NAME, cmd);
    let mut out = out.trim_end().to_string();
    out.push('\n');
    out
}

/// Renders one command block, then recurses into its subcommands.
fn render_command(out: &mut String, path: &str, cmd: &clap::Command) {
    writeln!(out, "== {path}").unwrap();
    if let Some(about) = cmd.get_about() {
        writeln!(out, "about: {}", escape(&about.to_string())).unwrap();
    }
    for arg in cmd.get_arguments() {
        // `help` and `version` are clap built-ins, not part of the surface under our control.
        if matches!(arg.get_id().as_str(), "help" | "version") {
            continue;
        }
        writeln!(out, "{}", render_arg(arg)).unwrap();
    }
    writeln!(out).unwrap();
    for sub in cmd.get_subcommands() {
        if sub.get_name() == "help" {
            continue;
        }
        render_command(out, &format!("{path} {}", sub.get_name()), sub);
    }
}

/// Renders a single argument as one line with a fixed field order.
fn render_arg(arg: &clap::Arg) -> String {
    let mut s = String::from("arg: ");

    let takes_value = matches!(arg.get_action(), clap::ArgAction::Set | clap::ArgAction::Append);

    match (arg.get_short(), arg.get_long()) {
        (Some(short), Some(long)) => write!(s, "-{short}/--{long}").unwrap(),
        (None, Some(long)) => write!(s, "--{long}").unwrap(),
        (Some(short), None) => write!(s, "-{short}").unwrap(),
        (None, None) => write!(s, "[positional: {}]", arg.get_id()).unwrap(),
    }

    if takes_value {
        let value_names = match arg.get_value_names() {
            Some(names) if !names.is_empty() => {
                names.iter().map(|n| n.to_string()).collect::<Vec<_>>().join(" ")
            }
            _ => arg.get_id().as_str().to_uppercase(),
        };
        write!(s, " <{value_names}>").unwrap();
    }

    if let Some(aliases) = arg.get_all_aliases() &&
        !aliases.is_empty()
    {
        write!(s, " [aliases: {}]", aliases.join(", ")).unwrap();
    }
    let short_aliases = arg.get_all_short_aliases().unwrap_or_default();
    if !short_aliases.is_empty() {
        let rendered: Vec<String> = short_aliases.iter().map(|c| format!("-{c}")).collect();
        write!(s, " [short-aliases: {}]", rendered.join(", ")).unwrap();
    }

    if let Some(env_var) = arg.get_env() {
        write!(s, " [env: {}]", env_var.to_string_lossy()).unwrap();
    }

    let defaults: Vec<String> = arg
        .get_default_values()
        .iter()
        .map(|v| normalize_default(arg.get_long(), &v.to_string_lossy()))
        .collect();
    if !defaults.is_empty() {
        write!(s, " [default: {}]", defaults.join(", ")).unwrap();
    }

    if takes_value {
        let possible: Vec<String> =
            arg.get_possible_values().iter().map(|p| p.get_name().to_string()).collect();
        if !possible.is_empty() {
            write!(s, " [possible: {}]", possible.join(", ")).unwrap();
        }
    }

    if arg.is_required_set() {
        s.push_str(" [required]");
    }
    if arg.is_global_set() {
        s.push_str(" [global]");
    }
    if arg.is_hide_set() {
        s.push_str(" [hidden]");
    }

    if let Some(help) = arg.get_help() {
        write!(s, " help: {}", escape(&help.to_string())).unwrap();
    }

    s
}

/// Replaces machine-specific default values with stable placeholders.
fn normalize_default(long: Option<&str>, value: &str) -> String {
    if let Some(long) = long &&
        MACHINE_SPECIFIC_DEFAULTS.contains(&long)
    {
        return "<MACHINE_SPECIFIC>".to_string();
    }
    for var in PATH_ENV_VARS {
        if let Ok(dir) = env::var(var) &&
            dir.len() > 1 &&
            value.contains(&dir)
        {
            return "<PLATFORM_PATH>".to_string();
        }
    }
    value.to_string()
}

/// Escapes newlines so every rendered element stays on a single line.
fn escape(s: &str) -> String {
    s.replace('\n', "\\n")
}

/// Returns the 1-based line number and contents of the first differing line.
fn first_diff_line<'a>(expected: &'a str, actual: &'a str) -> (usize, &'a str, &'a str) {
    let mut left = expected.lines();
    let mut right = actual.lines();
    let mut line_no = 0usize;
    loop {
        line_no += 1;
        match (left.next(), right.next()) {
            (None, None) => return (line_no, "<missing>", "<missing>"),
            (l, r) if l != r => {
                return (line_no, l.unwrap_or("<missing>"), r.unwrap_or("<missing>"));
            }
            _ => {}
        }
    }
}
