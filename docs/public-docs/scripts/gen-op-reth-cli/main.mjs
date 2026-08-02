#!/usr/bin/env node
// gen-op-reth-cli — regenerates the op-reth CLI reference for docs.optimism.io.
//
// Walks the `--help` tree of a pinned op-reth release binary and emits:
//   - one Mintlify MDX page per command under
//     docs/public-docs/node-operators/op-reth/cli/ (op-reth.mdx + op-reth/**),
//   - the nav fragment for the "op-reth CLI reference" group in docs.json
//     (the collapsed group under Node Operators > Reference),
//   - provenance in manifest.json (release tag, binary version, tree SHA-256).
//
// Adapted from upstream reth's docs/cli/update.sh + docs/cli/help.rs (the
// tooling whose output format the original snapshot matches), ported to a
// zero-dependency Node script so it runs with the docs toolchain.
//
// Usage (from anywhere; paths are resolved relative to this script):
//   node docs/public-docs/scripts/gen-op-reth-cli/main.mjs \
//     --bin /path/to/op-reth --tag op-reth/vX.Y.Z
//   node docs/public-docs/scripts/gen-op-reth-cli/main.mjs --bin ... --check
//
// Rules:
//   - Regenerate only from a finalized (non-rc) op-reth release tag's binary.
//   - The hand-written cli/overview.mdx page is never touched.
//   - Pages this run deletes are printed at the end: every deleted page URL
//     must gain a redirect in docs.json in the same PR (redirect rule R5).

import { execFileSync } from "node:child_process";
import {
  createHash,
} from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DOCS_ROOT = path.resolve(SCRIPT_DIR, "..", "..");
const CLI_DIR = path.join(DOCS_ROOT, "node-operators", "op-reth", "cli");
const CLI_SLUG_PREFIX = "node-operators/op-reth/cli";
const DOCS_JSON = path.join(DOCS_ROOT, "docs.json");
const MANIFEST = path.join(SCRIPT_DIR, "manifest.json");
const NAV_GROUP = "op-reth CLI reference";
const ROOT_NAME = "op-reth";

function usageExit(msg) {
  if (msg) console.error(`error: ${msg}`);
  console.error(
    "usage: main.mjs --bin <op-reth binary> [--tag op-reth/vX.Y.Z] [--check]",
  );
  process.exit(2);
}

function parseArgs(argv) {
  const args = { bin: null, tag: null, check: false };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--bin") args.bin = argv[++i];
    else if (a === "--tag") args.tag = argv[++i];
    else if (a === "--check") args.check = true;
    else usageExit(`unknown argument: ${a}`);
  }
  if (!args.bin) usageExit("--bin is required");
  if (!args.check && !args.tag) usageExit("--tag is required when writing");
  if (args.tag && !/^op-reth\/v\d+\.\d+\.\d+$/.test(args.tag)) {
    usageExit(
      `--tag must be a finalized op-reth release tag (op-reth/vX.Y.Z), got: ${args.tag}`,
    );
  }
  return args;
}

/** Run `<bin> <subcommands...> --help` the way upstream help.rs does. */
function runHelp(bin, subcommands) {
  return execFileSync(bin, [...subcommands, "--help"], {
    encoding: "utf8",
    env: { ...process.env, NO_COLOR: "1", COLUMNS: "100", LINES: "10000" },
    maxBuffer: 16 * 1024 * 1024,
  });
}

/** Parse subcommand names from a help output's "Commands:" section. */
function parseSubCommands(helpText) {
  const afterCommands = helpText.split("Commands:")[1];
  if (afterCommands === undefined) return [];
  const names = [];
  for (const line of afterCommands.split("\n")) {
    if (line.startsWith("Options:") || line.startsWith("Arguments:")) break;
    const m = /^ {2}(\S+)/.exec(line);
    if (m && m[1] !== "help") names.push(m[1]);
  }
  return names;
}

/**
 * The same environment-dependent-output scrubs as upstream help.rs
 * (preprocess_help), in the same order. These produced the committed
 * snapshot's `<CACHE_DIR>` / `<VERSION>` / `<OS>` placeholders.
 */
const REPLACEMENTS = [
  // Fork-specific: op-reth identifies as `op-reth/<version>-<short sha>/<arch>`
  // (rust/op-version) in e.g. the static-files client-version default; the
  // upstream patterns below only cover the `reth/...` spelling.
  [
    /default: op-reth\/.*-[0-9A-Fa-f]{6,10}\/[^\]\s]+/g,
    "default: op-reth/<VERSION>-<SHA>/<ARCH>",
  ],
  [/default: \/.*\/reth/g, "default: <CACHE_DIR>"],
  [
    /default: reth\/.*-[0-9A-Fa-f]{6,10}\/([_\w]+)-(\w+)-(\w+)(-\w+)?/g,
    "default: reth/<VERSION>-<SHA>/<ARCH>",
  ],
  [/default: reth\/.*\/\w+/g, "default: reth/<VERSION>/<OS>"],
  [
    /(rpc.max-tracing-requests <COUNT>\n.*\n.*\n.*\n.*\n.*)\[default: \d+\]/g,
    "$1[default: <NUM CPU CORES-2>]",
  ],
  [
    /(engine\.reserved-cpu-cores.*)\[default: \d+\]/g,
    "$1[default: <DYNAMIC: min(2, CPU cores)>]",
  ],
];

function preprocessHelp(s) {
  let out = s;
  for (const [re, replacement] of REPLACEMENTS) {
    out = out.replace(re, replacement);
  }
  return out;
}

/** Split help output into (description, rest-from-Usage). */
function parseDescription(s) {
  const idx = s.indexOf("Usage:");
  if (idx === -1) return ["", s];
  const description = s.slice(0, idx).trim().split("\n")[0] ?? "";
  return [description, s.slice(idx)];
}

/** Trailing-whitespace-trimmed page content, upstream write_file semantics. */
function trimLineEnds(content) {
  return content
    .split("\n")
    .map((line) => line.replace(/\s+$/, ""))
    .join("\n");
}

/** Render one command's MDX page (frontmatter port of upstream cmd_markdown). */
function renderPage(displayCmd, helpText) {
  const [description, rest] = parseDescription(helpText);
  const body = `---\ntitle: "${displayCmd}"\ndiataxis: reference\n---\n\n${description}\n\n\`\`\`bash\n$ ${displayCmd} --help\n\`\`\`\n\`\`\`txt\n${preprocessHelp(rest.trim())}\n\`\`\``;
  return trimLineEnds(body);
}

/** Walk the binary's help tree depth-first, preserving help order. */
function walk(bin, subcommands) {
  const helpText = runHelp(bin, subcommands);
  const displayCmd = [ROOT_NAME, ...subcommands].join(" ");
  const node = {
    subcommands,
    displayCmd,
    slug:
      subcommands.length === 0
        ? `${CLI_SLUG_PREFIX}/${ROOT_NAME}`
        : `${CLI_SLUG_PREFIX}/${ROOT_NAME}/${subcommands.join("/")}`,
    page: renderPage(displayCmd, helpText),
    children: [],
  };
  for (const name of parseSubCommands(helpText)) {
    node.children.push(walk(bin, [...subcommands, name]));
  }
  return node;
}

/** Flatten the tree to [{relPath, content}] with upstream's path layout. */
function flattenPages(node, out = []) {
  const rel =
    node.subcommands.length === 0
      ? `${ROOT_NAME}.mdx`
      : `${ROOT_NAME}/${node.subcommands.join("/")}.mdx`;
  out.push({ relPath: rel, content: node.page });
  for (const child of node.children) flattenPages(child, out);
  return out;
}

/** Build the docs.json nav entry for a node: a string for leaves, a group otherwise. */
function navEntry(node) {
  if (node.children.length === 0) return node.slug;
  return {
    group: node.displayCmd,
    pages: [node.slug, ...node.children.map(navEntry)],
  };
}

/** Recursively find the CLI reference group object inside docs.json navigation. */
function findNavGroup(value, results = []) {
  if (Array.isArray(value)) {
    for (const item of value) findNavGroup(item, results);
  } else if (value && typeof value === "object") {
    if (value.group === NAV_GROUP) results.push(value);
    for (const v of Object.values(value)) findNavGroup(v, results);
  }
  return results;
}

/** Collect committed generated pages: op-reth.mdx plus everything under op-reth/. */
function collectExistingPages() {
  const out = new Map();
  const rootPage = path.join(CLI_DIR, `${ROOT_NAME}.mdx`);
  if (fs.existsSync(rootPage)) {
    out.set(`${ROOT_NAME}.mdx`, fs.readFileSync(rootPage, "utf8"));
  }
  const subdir = path.join(CLI_DIR, ROOT_NAME);
  const visit = (dir) => {
    if (!fs.existsSync(dir)) return;
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const p = path.join(dir, entry.name);
      if (entry.isDirectory()) visit(p);
      else if (entry.name.endsWith(".mdx")) {
        out.set(path.relative(CLI_DIR, p), fs.readFileSync(p, "utf8"));
      }
    }
  };
  visit(subdir);
  return out;
}

/** Deterministic SHA-256 over the generated tree (sorted relPath + NUL + bytes). */
function treeSha256(pages) {
  const h = createHash("sha256");
  for (const { relPath, content } of [...pages].sort((a, b) =>
    a.relPath < b.relPath ? -1 : 1,
  )) {
    h.update(relPath);
    h.update("\0");
    h.update(content);
    h.update("\0");
  }
  return h.digest("hex");
}

function removeEmptyDirs(dir) {
  if (!fs.existsSync(dir) || !fs.statSync(dir).isDirectory()) return;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) removeEmptyDirs(path.join(dir, entry.name));
  }
  if (fs.readdirSync(dir).length === 0) fs.rmdirSync(dir);
}

function main() {
  const args = parseArgs(process.argv.slice(2));

  const versionOut = execFileSync(args.bin, ["--version"], {
    encoding: "utf8",
    env: { ...process.env, NO_COLOR: "1" },
  });
  const binaryVersion = versionOut.trim().split("\n")[0];
  console.log(`binary: ${args.bin}`);
  console.log(`binary version: ${binaryVersion}`);

  console.log("walking --help tree...");
  const tree = walk(args.bin, []);
  const pages = flattenPages(tree);
  console.log(`generated ${pages.length} pages from --help`);

  const generated = new Map(pages.map((p) => [p.relPath, p.content]));
  const existing = collectExistingPages();

  const added = [...generated.keys()].filter((k) => !existing.has(k)).sort();
  const removed = [...existing.keys()].filter((k) => !generated.has(k)).sort();
  const changed = [...generated.keys()]
    .filter((k) => existing.has(k) && existing.get(k) !== generated.get(k))
    .sort();

  // Nav fragment: [hand-written overview, regenerated command tree].
  const docsJsonRaw = fs.readFileSync(DOCS_JSON, "utf8");
  const docsJson = JSON.parse(docsJsonRaw);
  const groups = findNavGroup(docsJson.navigation);
  if (groups.length !== 1) {
    console.error(
      `error: expected exactly one "${NAV_GROUP}" group in docs.json, found ${groups.length}`,
    );
    process.exit(1);
  }
  const group = groups[0];
  const overviewEntry = group.pages[0];
  if (
    typeof overviewEntry !== "string" ||
    !overviewEntry.endsWith("/cli/overview")
  ) {
    console.error(
      `error: first page of "${NAV_GROUP}" must be the hand-written overview, found: ${JSON.stringify(overviewEntry)}`,
    );
    process.exit(1);
  }
  const newNavPages = [overviewEntry, navEntry(tree)];
  const navChanged =
    JSON.stringify(group.pages) !== JSON.stringify(newNavPages);

  const sha256 = treeSha256(pages);
  const manifest = fs.existsSync(MANIFEST)
    ? JSON.parse(fs.readFileSync(MANIFEST, "utf8"))
    : {};

  if (args.check) {
    let failed = false;
    const recorded = manifest[ROOT_NAME];
    if (!recorded) {
      console.error("check: manifest.json has no op-reth entry");
      failed = true;
    }
    if (added.length || removed.length || changed.length) {
      console.error(
        `check: committed pages differ from regeneration (added ${added.length}, removed ${removed.length}, changed ${changed.length})`,
      );
      for (const k of added) console.error(`  new:     ${k}`);
      for (const k of removed) console.error(`  stale:   ${k}`);
      for (const k of changed) console.error(`  changed: ${k}`);
      failed = true;
    }
    if (navChanged) {
      console.error("check: docs.json nav fragment differs from regeneration");
      failed = true;
    }
    if (recorded && recorded.sha256 !== sha256) {
      console.error(
        `check: manifest sha256 ${recorded.sha256} != regenerated ${sha256}`,
      );
      failed = true;
    }
    if (failed) {
      console.error(
        "check failed. If op-reth's source moved past the manifest tag, this is expected drift: regenerate at the next finalized op-reth release tag.",
      );
      process.exit(1);
    }
    console.log(
      `check OK: ${pages.length} pages, nav fragment, and manifest (tag ${recorded.tag}) all match the binary's --help tree`,
    );
    return;
  }

  // Write pages.
  for (const { relPath, content } of pages) {
    const abs = path.join(CLI_DIR, relPath);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content);
  }
  // Delete stale pages and prune empty directories.
  for (const relPath of removed) {
    fs.rmSync(path.join(CLI_DIR, relPath));
  }
  removeEmptyDirs(path.join(CLI_DIR, ROOT_NAME));

  // Splice the nav fragment and write docs.json back (byte-stable round-trip).
  group.pages = newNavPages;
  fs.writeFileSync(DOCS_JSON, `${JSON.stringify(docsJson, null, 2)}\n`);

  // Record provenance.
  manifest[ROOT_NAME] = { binaryVersion, sha256, tag: args.tag };
  fs.writeFileSync(MANIFEST, `${JSON.stringify(manifest, null, 2)}\n`);

  console.log(
    `wrote ${pages.length} pages (${added.length} new, ${changed.length} changed), removed ${removed.length}, nav fragment ${navChanged ? "updated" : "unchanged"}`,
  );
  console.log(`manifest: tag ${args.tag}, sha256 ${sha256}`);
  if (removed.length) {
    console.log(
      "\nDeleted pages — each URL below must gain a redirect in docs.json in this same PR (redirect rule R5):",
    );
    for (const relPath of removed) {
      console.log(`  /${CLI_SLUG_PREFIX}/${relPath.replace(/\.mdx$/, "")}`);
    }
  }
}

main();
