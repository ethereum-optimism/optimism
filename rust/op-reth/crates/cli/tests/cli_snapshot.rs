//! Snapshot test of the complete op-reth CLI surface.
//!
//! Renders the full clap command tree (every subcommand, every argument, including hidden ones)
//! into a deterministic text form and compares it against the checked-in snapshot at
//! `tests/snapshots/cli.snap`. Any change to the CLI surface — a new upstream reth flag after a
//! pin bump, a renamed alias, a changed default, or changed parsing behavior — shows up as a diff
//! in the snapshot and has to be acknowledged by regenerating it:
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
//! (the `rust-tests` `CircleCI` job), which enables it.
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
    render_command_metadata(out, cmd);
    if let Some(about) = cmd.get_about() {
        writeln!(out, "about: {}", escape(&about.to_string())).unwrap();
    }
    for arg in cmd.get_arguments() {
        // `help` and `version` are clap built-ins, not part of the surface under our control.
        if matches!(arg.get_id().as_str(), "help" | "version") {
            continue;
        }
        writeln!(out, "{}", render_arg(cmd, arg)).unwrap();
    }
    writeln!(out).unwrap();
    for sub in cmd.get_subcommands() {
        if sub.get_name() == "help" {
            continue;
        }
        render_command(out, &format!("{path} {}", sub.get_name()), sub);
    }
}

/// Renders aliases, flag forms, and visibility for a command.
fn render_command_metadata(out: &mut String, cmd: &clap::Command) {
    let mut metadata = String::from("command:");

    let aliases: Vec<&str> = cmd.get_all_aliases().collect();
    if !aliases.is_empty() {
        write!(metadata, " [aliases: {}]", aliases.join(", ")).unwrap();
    }
    let visible_aliases: Vec<&str> = cmd.get_visible_aliases().collect();
    if !visible_aliases.is_empty() {
        write!(metadata, " [visible-aliases: {}]", visible_aliases.join(", ")).unwrap();
    }

    if let Some(short) = cmd.get_short_flag() {
        write!(metadata, " [short-flag: -{short}]").unwrap();
    }
    let short_aliases: Vec<String> =
        cmd.get_all_short_flag_aliases().map(|alias| format!("-{alias}")).collect();
    if !short_aliases.is_empty() {
        write!(metadata, " [short-flag-aliases: {}]", short_aliases.join(", ")).unwrap();
    }
    let visible_short_aliases: Vec<String> =
        cmd.get_visible_short_flag_aliases().map(|alias| format!("-{alias}")).collect();
    if !visible_short_aliases.is_empty() {
        write!(metadata, " [visible-short-flag-aliases: {}]", visible_short_aliases.join(", "))
            .unwrap();
    }

    if let Some(long) = cmd.get_long_flag() {
        write!(metadata, " [long-flag: --{long}]").unwrap();
    }
    let long_aliases: Vec<String> =
        cmd.get_all_long_flag_aliases().map(|alias| format!("--{alias}")).collect();
    if !long_aliases.is_empty() {
        write!(metadata, " [long-flag-aliases: {}]", long_aliases.join(", ")).unwrap();
    }
    let visible_long_aliases: Vec<String> =
        cmd.get_visible_long_flag_aliases().map(|alias| format!("--{alias}")).collect();
    if !visible_long_aliases.is_empty() {
        write!(metadata, " [visible-long-flag-aliases: {}]", visible_long_aliases.join(", "))
            .unwrap();
    }

    if cmd.is_hide_set() {
        metadata.push_str(" [hidden]");
    }
    if metadata != "command:" {
        writeln!(out, "{metadata}").unwrap();
    }
}

/// Renders a single argument as one line with a fixed field order.
fn render_arg(cmd: &clap::Command, arg: &clap::Arg) -> String {
    let mut s = format!("arg: {}", arg_name(arg));
    let arg_debug = format!("{arg:?}");

    let configured_num_args = arg.get_num_args();
    let takes_value = configured_num_args.map_or_else(
        || matches!(arg.get_action(), clap::ArgAction::Set | clap::ArgAction::Append),
        |range| range.takes_values(),
    );

    if takes_value {
        let value_names = match arg.get_value_names() {
            Some(names) if !names.is_empty() => {
                names.iter().map(|n| n.to_string()).collect::<Vec<_>>().join(" ")
            }
            _ => arg.get_id().as_str().to_uppercase(),
        };
        write!(s, " <{value_names}>").unwrap();
    }

    write!(s, " [action: {:?}]", arg.get_action()).unwrap();
    if let Some(range) = configured_num_args {
        write!(s, " [num-args: {range}]").unwrap();
    } else {
        write!(s, " [num-args: {}]", usize::from(takes_value)).unwrap();
    }
    if let Some(delimiter) = arg.get_value_delimiter() {
        write!(s, " [value-delimiter: {delimiter:?}]").unwrap();
    }
    if let Some(terminator) = arg.get_value_terminator() {
        write!(s, " [value-terminator: {}]", escape(terminator.as_str())).unwrap();
    }

    if let Some(aliases) = arg.get_all_aliases() &&
        !aliases.is_empty()
    {
        write!(s, " [aliases: {}]", aliases.join(", ")).unwrap();
    }
    if let Some(visible_aliases) = arg.get_visible_aliases() &&
        !visible_aliases.is_empty()
    {
        write!(s, " [visible-aliases: {}]", visible_aliases.join(", ")).unwrap();
    }
    let short_aliases = arg.get_all_short_aliases().unwrap_or_default();
    if !short_aliases.is_empty() {
        let rendered: Vec<String> = short_aliases.iter().map(|c| format!("-{c}")).collect();
        write!(s, " [short-aliases: {}]", rendered.join(", ")).unwrap();
    }
    let visible_short_aliases = arg.get_visible_short_aliases().unwrap_or_default();
    if !visible_short_aliases.is_empty() {
        let rendered: Vec<String> = visible_short_aliases.iter().map(|c| format!("-{c}")).collect();
        write!(s, " [visible-short-aliases: {}]", rendered.join(", ")).unwrap();
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
    render_arg_debug_field(
        &mut s,
        &arg_debug,
        "default_missing_vals",
        "ext",
        "default-missing-values",
    );

    if takes_value {
        let possible: Vec<String> =
            arg.get_possible_values().iter().map(render_possible_value).collect();
        if !possible.is_empty() {
            write!(s, " [possible: {}]", possible.join(", ")).unwrap();
        }
    }

    let mut conflicts: Vec<String> =
        cmd.get_arg_conflicts_with(arg).into_iter().map(arg_name).collect();
    conflicts.sort_unstable();
    conflicts.dedup();
    if !conflicts.is_empty() {
        write!(s, " [conflicts: {}]", conflicts.join(", ")).unwrap();
    }
    // clap does not expose reflection getters for requirement predicates or default-missing
    // values. Its Debug implementation does expose their value-only representations; the focused
    // test below ensures a clap update cannot silently remove or rename these fields.
    render_arg_debug_field(&mut s, &arg_debug, "requires", "r_ifs", "requires");
    render_arg_debug_field(&mut s, &arg_debug, "r_ifs", "r_unless", "requires-if");
    render_arg_debug_field(&mut s, &arg_debug, "r_unless", "short", "requires-unless");

    if arg.is_required_set() {
        s.push_str(" [required]");
    }
    if arg.is_global_set() {
        s.push_str(" [global]");
    }
    if arg.is_hide_set() {
        s.push_str(" [hidden]");
    }
    if arg.is_require_equals_set() {
        s.push_str(" [require-equals]");
    }
    if arg.is_allow_hyphen_values_set() {
        s.push_str(" [allow-hyphen-values]");
    }
    if arg.is_allow_negative_numbers_set() {
        s.push_str(" [allow-negative-numbers]");
    }
    if arg.is_exclusive_set() {
        s.push_str(" [exclusive]");
    }
    if arg.is_trailing_var_arg_set() {
        s.push_str(" [trailing-var-arg]");
    }
    if arg.is_last_set() {
        s.push_str(" [last]");
    }
    if arg.is_ignore_case_set() {
        s.push_str(" [ignore-case]");
    }

    if let Some(help) = arg.get_help() {
        write!(s, " help: {}", escape(&help.to_string())).unwrap();
    }

    s
}

/// Appends a non-empty field from clap's value-only [`Debug`] representation of an argument.
fn render_arg_debug_field(
    out: &mut String,
    debug: &str,
    field: &str,
    next_field: &str,
    label: &str,
) {
    let start = format!(", {field}: ");
    let end = format!(", {next_field}:");
    let value = debug
        .split_once(&start)
        .and_then(|(_, rest)| rest.split_once(&end))
        .map(|(value, _)| value);
    if let Some(value) = value &&
        !matches!(value, "[]" | "None")
    {
        write!(out, " [{label}: {value}]").unwrap();
    }
}

/// Renders a possible value, including accepted aliases and visibility.
fn render_possible_value(value: &clap::builder::PossibleValue) -> String {
    let mut names = value.get_name_and_aliases();
    let mut rendered = names.next().unwrap_or_default().to_string();
    let aliases: Vec<&str> = names.collect();
    if !aliases.is_empty() {
        write!(rendered, " (aliases: {})", aliases.join(", ")).unwrap();
    }
    if value.is_hide_set() {
        rendered.push_str(" (hidden)");
    }
    rendered
}

/// Returns an argument's operator-facing name.
fn arg_name(arg: &clap::Arg) -> String {
    match (arg.get_short(), arg.get_long()) {
        (Some(short), Some(long)) => format!("-{short}/--{long}"),
        (None, Some(long)) => format!("--{long}"),
        (Some(short), None) => format!("-{short}"),
        (None, None) => format!("[positional: {}]", arg.get_id()),
    }
}

#[test]
fn renders_parser_requirements_and_missing_defaults() {
    let command = clap::Command::new("test").arg(clap::Arg::new("mode").long("mode")).arg(
        clap::Arg::new("output")
            .long("output")
            .num_args(0..=1)
            .default_missing_value("stdout")
            .requires("mode"),
    );

    let rendered = render_snapshot(&command);
    assert!(rendered.contains(
        "arg: --output <OUTPUT> [action: Set] [num-args: 0..=1] \
         [default-missing-values: [\"stdout\"]] [requires: [(IsPresent, \"mode\")]]"
    ));
}

#[test]
fn renders_alias_and_visibility_metadata() {
    let command = clap::Command::new("test").subcommand(
        clap::Command::new("serve")
            .alias("s")
            .visible_alias("run")
            .short_flag('S')
            .short_flag_alias('x')
            .visible_short_flag_alias('r')
            .long_flag("serve-now")
            .long_flag_alias("start")
            .visible_long_flag_alias("run-now")
            .hide(true)
            .arg(
                clap::Arg::new("mode")
                    .long("mode")
                    .alias("kind")
                    .visible_alias("type")
                    .short_alias('k')
                    .visible_short_alias('t')
                    .value_parser(clap::builder::PossibleValuesParser::new([
                        clap::builder::PossibleValue::new("fast").alias("quick"),
                        clap::builder::PossibleValue::new("internal").hide(true),
                    ])),
            ),
    );

    let rendered = render_snapshot(&command);
    assert!(rendered.contains(
        "command: [aliases: s, run] [visible-aliases: run] [short-flag: -S] \
         [short-flag-aliases: -x, -r] [visible-short-flag-aliases: -r] \
         [long-flag: --serve-now] [long-flag-aliases: --start, --run-now] \
         [visible-long-flag-aliases: --run-now] [hidden]"
    ));
    assert!(rendered.contains(
        "arg: --mode <MODE> [action: Set] [num-args: 1] [aliases: kind, type] \
         [visible-aliases: type] [short-aliases: -k, -t] [visible-short-aliases: -t] \
         [possible: fast (aliases: quick), internal (hidden)]"
    ));
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
