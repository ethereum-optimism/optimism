# gen-cli — generated, versioned component CLI reference for docs.optimism.io

One generator for every OP Stack component with a command-line interface. It
walks the `--help` tree of a component's **published release artifact**,
parses it (clap for the Rust components, urfave/cli for the Go services), and
emits the pages, the `docs.json` version entry, and the manifest for one
release line of one component, exactly as the
[Component reference](../../op-stack/contribute/content-guide.mdx#component-reference)
section of the content guide (ethereum-optimism/optimism#23091) specifies. It replaces `gen-flags` and
`gen-op-reth-cli`, which stay until every component has migrated.

## What it emits

For component `C` and release line `vX.Y` (the newest finalized patch in the
line; or `develop`):

| Path | Content |
| --- | --- |
| `reference/C/vX.Y/index.mdx` | The binary: description, usage, table of commands, its own option tables |
| `reference/C/vX.Y/<command>.mdx` | One page per top-level command. Nested subcommands are headings on it (`## checksum`, `### checksum mdbx`), so `db#checksum-mdbx` deep-links a subcommand without a page of its own |
| `reference/C/vX.Y/global-options.mdx` | Option sections that at least half of the commands print identically (op-reth's Logging, Display, Tracing). Command pages link here instead of repeating them, and the verbatim help on each page omits them |
| `docs.json` | Reference tab → dropdown `C` → version `vX.Y` → groups Commands and Global options. The newest line is `default` and tagged Latest; `develop` is tagged Unreleased and sorts last |
| `manifest.json` | Per component per line: the tag (or commit for `develop`), the artifact, the binary's `--version`, and a SHA-256 over the emitted pages |

Every flag renders as a table row (flag, description, default, environment
variable), and each command's verbatim `--help` follows in an `<Accordion>`.
Every page opens with a `DO NOT EDIT` comment and a provenance line linking
the GitHub release. Pages on the `develop` line carry `noindex: true`.

## Usage

Run with bun (mise pins it) from the monorepo root or `docs/public-docs/`:

```bash
# Regenerate a release line from the published image (CI path)
bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --tag op-reth/v2.4.4 --image auto

# From a local binary (op-deployer's release tarball, or a development build)
bun docs/public-docs/scripts/gen-cli/main.ts --component op-node --tag op-node/v1.19.7 --bin ./op-node

# The develop line, from a binary built at a commit
bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --develop --commit <sha> --bin ./op-reth

# Verify the committed pages, nav entry and manifest against the artifact
bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --tag op-reth/v2.4.4 --image auto --check

# Record a help tree (for a fixture) / build the migration redirect map
bun docs/public-docs/scripts/gen-cli/main.ts … --dump-help-json op-reth.json --page-map op-reth-pages.json
```

`--image auto` resolves to the component's image at the tag
(`us-docker.pkg.dev/oplabs-tools-artifacts/images/<component>:<version>`)
and runs `docker run --rm … --help`. `--help-json` replays a recording, which
is how the tests run without a binary. Exactly one source is required.

The version directory is owned by the generator: any `.mdx` in it that the
run did not emit (a hand-written page, a command that disappeared) is
deleted, and deleted pages are printed at the end; each needs a redirect in
`docs.json` in the same PR (the redirect lint enforces it). `--dump-help-json`
and `--page-map` also write in `--check` mode. The CI job that runs this on
every finalized tag lives in `.circleci/continue/docs-ci.yml`
(ethereum-optimism/solutions#1518, Phase 2); until it exists, regenerate by
hand when a release is published.

## Tests

```bash
bun docs/public-docs/scripts/gen-cli/test.ts     # or: pnpm test:gen-cli
```

The tests replay the recordings under `fixtures/`: a pruned op-reth v2.4.1
tree (root, `node`, `db` and three `db` subcommands) and a development
op-node tree. They cover both parsers, hoisting, the page tree and anchors,
determinism, nav ordering, and a write-then-`--check` round trip in a
temporary docs root. Re-record a fixture with `--dump-help-json` when a
parser change needs new input; prune it to the commands the tests use.

## How the parsers work

- **clap** (op-reth, kona-node): headers end with `:` at column 0; a flag
  line is `  -s, --long <VALUE>` and its description follows indented ten
  spaces, with `[default: …]`, `[env: …]` and `Possible values:` blocks
  parsed out. The environment-dependent defaults upstream reth scrubs
  (`<CACHE_DIR>`, `<VERSION>`, CPU-count defaults) are scrubbed the same way.
- **urfave/cli** (the Go services, via op-service's help template): `NAME:`,
  `USAGE:`, `COMMANDS:` and `GLOBAL OPTIONS:`/`OPTIONS:`; a flag line carries
  the names, `(default: …)` and `($ENV)`, with the usage on the following
  lines; a numbered line (`1. ROLLUP`) starts a category, which becomes a
  section. Subcommands are found from `COMMANDS:` and walked recursively.

Both parsers reject a subcommand name that does not match `[a-z0-9][a-z0-9._-]*`
before it reaches a process argument or a file path.

Per-component rewrites live in `components.ts`: op-batcher's `--compressor`
option list is sorted (the binary prints it in map order), and the
superchain-registry chain lists behind op-reth's `--chain` and the Go
services' `--network` are replaced by a pointer, since they change with every
registry bump independently of the component.

## Adding a component

1. Add it to `COMPONENTS` in `components.ts` with its framework, artifact,
   title and icon, and to the covered list in the content guide and
   `scripts/lint/validate-reference.ts`.
2. Run the generator at the component's newest finalized tag.
3. Move any hand-written prose about the component's flags into a guide in
   the persona tabs, and add redirects from the old reference URLs.

## Ownership

Generator code is owned by solutions via the CODEOWNERS rule
(`/docs/public-docs/scripts/  @ethereum-optimism/solutions`). Generated pages
are owned by the pipeline: nobody hand-edits them, and `--check` fails on a
hand edit by construction. CLI facts belong to the component teams; the docs
follow at the next finalized release.
