# fix-rust-fmt

Fix Rust formatting issues in the optimism monorepo which would cause the `rust-fmt` CI job to fail.

## When to Use

Use this skill when the `rust-fmt` CI job fails on a PR that touches Rust code.

### Trigger Phrases

- "Fix rust formatting"
- "rust-fmt is failing"
- "Fix the rust-fmt CI job"

## Prerequisites

- `mise` must be trusted and installed for this repo (`mise trust && mise install`)

## Workflow

### Step 1: Ensure mise tools are installed

From the repo root (or worktree root):

```bash
cd <REPO_ROOT> && mise install
```

This installs `rust`, `just`, and all other tools pinned in `mise.toml`.

### Step 2: Install the nightly toolchain with rustfmt

The justfile pins a specific nightly (see `NIGHTLY` variable in `rust/justfile`).
Install it:

```bash
cd <REPO_ROOT>/rust && mise exec -- just install-nightly
```

### Step 3: Run the formatter

```bash
cd <REPO_ROOT>/rust && mise exec -- just fmt-fix
```

Any files that change are the reason the CI job failed. Stage and commit them.

## Notes

- `mise exec --` activates the mise environment so `cargo`, `just`, and
  `rustup` resolve to the versions pinned in `mise.toml`.
- The nightly toolchain is required because the workspace uses unstable
  `rustfmt` options (see `rust/rustfmt.toml`).
- There is no need to inspect `rust-fmt` CI errors — if the job failed, running
  `just fmt-fix` and committing the result is the complete fix.
