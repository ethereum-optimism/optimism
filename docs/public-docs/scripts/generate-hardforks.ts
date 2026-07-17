// Hardfork registry renderer + schema check (docs-improver slice B4; consumed by
// Option A slice 6).
//
// Every hardfork has exactly one permanent registry page under
// docs/public-docs/op-stack/protocol/hardforks/<fork>.mdx. The machine-readable
// facts (lifecycle, activation timestamps, governing spec link, minimum component
// versions) live in that page's frontmatter — see scripts/hardfork-registry.md for
// the schema. This script:
//
//   1. Validates every registry page's frontmatter against the schema.
//   2. Cross-validates activation timestamps against the superchain-registry
//      (the monorepo submodule at superchain-registry/), so a wrong activation
//      row cannot render: superchain/configs/{mainnet,sepolia}/superchain.toml
//      is the source of truth for superchain-wide activation defaults.
//   3. Renders the generated snippets under snippets/generated/hardforks/:
//      one summary table (summary.mdx) and one per-fork facts table
//      (<fork>.mdx), which the registry pages import. Nobody hand-edits the
//      snippets; a frontmatter edit is the only way to change an activation row.
//
// Usage (from docs/public-docs/):
//   pnpm gen:hardforks                # regenerate snippets
//   pnpm gen:hardforks:check          # drift check: fail if snippets are stale
//   tsx scripts/generate-hardforks.ts --registry /path/to/superchain-registry
//
// The superchain-registry defaults to the monorepo submodule two levels up
// (../../superchain-registry). If the submodule is not initialized, run
// `git submodule update --init superchain-registry` from the monorepo root or
// pass --registry pointing at a checkout of
// https://github.com/ethereum-optimism/superchain-registry.
//
// Exit codes: 0 = clean, 1 = validation failure or drift, 2 = setup error.

import * as fs from "fs";
import * as path from "path";

// ---------------------------------------------------------------------------
// Locations
// ---------------------------------------------------------------------------

function findDocsRoot(): string {
  let dir = process.cwd();
  for (let i = 0; i < 6; i++) {
    if (fs.existsSync(path.join(dir, "docs.json"))) return dir;
    const candidate = path.join(dir, "docs", "public-docs");
    if (fs.existsSync(path.join(candidate, "docs.json"))) return candidate;
    dir = path.dirname(dir);
  }
  console.error("setup error: could not locate docs/public-docs (no docs.json found)");
  process.exit(2);
}

const docsRoot = findDocsRoot();
const hardforksDir = path.join(docsRoot, "op-stack", "protocol", "hardforks");
const outDir = path.join(docsRoot, "snippets", "generated", "hardforks");

const args = process.argv.slice(2);
const checkMode = args.includes("--check");
const registryArgIdx = args.indexOf("--registry");
const registryDir =
  registryArgIdx >= 0
    ? args[registryArgIdx + 1]
    : path.join(docsRoot, "..", "..", "superchain-registry");

// ---------------------------------------------------------------------------
// Frontmatter schema (see scripts/hardfork-registry.md)
// ---------------------------------------------------------------------------

const LIFECYCLES = ["active", "scheduled", "development"] as const;
type Lifecycle = (typeof LIFECYCLES)[number];

interface HardforkEntry {
  file: string;
  name: string;
  lifecycle: Lifecycle;
  spec: string;
  activationMainnet?: number;
  activationSepolia?: number;
  governance?: string;
  notice?: string;
  upgradeNumber?: number;
  minVersions: { component: string; version: string }[];
  minVersionsSource?: string;
}

// Minimal frontmatter reader for the flat schema: scalar `key: value` lines and
// block string lists (`key:` followed by `  - item` lines). Deliberately not a
// general YAML parser — the schema is designed to stay within this shape.
function parseFrontmatter(raw: string, file: string): Map<string, string | string[]> {
  const m = raw.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!m) fail(`${file}: no frontmatter block`);
  const out = new Map<string, string | string[]>();
  const lines = m![1].split(/\r?\n/);
  let listKey: string | null = null;
  for (const line of lines) {
    if (!line.trim() || line.trim().startsWith("#")) continue;
    const listItem = line.match(/^\s+-\s+(.*)$/);
    if (listItem && listKey) {
      (out.get(listKey) as string[]).push(stripQuotes(listItem[1].trim()));
      continue;
    }
    const kv = line.match(/^([A-Za-z0-9_]+):\s*(.*)$/);
    if (kv) {
      const [, key, value] = kv;
      if (value === "") {
        listKey = key;
        out.set(key, []);
      } else {
        listKey = null;
        out.set(key, stripQuotes(value.trim()));
      }
      continue;
    }
    listKey = null; // multi-line scalars etc. — not part of the schema
  }
  return out;
}

function stripQuotes(s: string): string {
  return s.replace(/^["']|["']$/g, "");
}

const errors: string[] = [];
function err(msg: string) {
  errors.push(msg);
}
function fail(msg: string): never {
  console.error(msg);
  process.exit(2);
}

function scalar(fm: Map<string, string | string[]>, key: string): string | undefined {
  const v = fm.get(key);
  return typeof v === "string" ? v : undefined;
}

function loadEntries(): HardforkEntry[] {
  if (!fs.existsSync(hardforksDir)) fail(`setup error: ${hardforksDir} does not exist`);
  const entries: HardforkEntry[] = [];
  for (const f of fs.readdirSync(hardforksDir).sort()) {
    if (!f.endsWith(".mdx")) continue;
    const file = path.join("op-stack", "protocol", "hardforks", f);
    const fm = parseFrontmatter(fs.readFileSync(path.join(hardforksDir, f), "utf-8"), file);

    const name = scalar(fm, "hardfork_name");
    if (!name) {
      err(`${file}: missing hardfork_name`);
      continue;
    }
    if (name !== f.replace(/\.mdx$/, "")) {
      err(`${file}: hardfork_name "${name}" must match the filename`);
    }
    const lifecycle = scalar(fm, "hardfork_lifecycle") as Lifecycle | undefined;
    if (!lifecycle || !LIFECYCLES.includes(lifecycle)) {
      err(`${file}: hardfork_lifecycle must be one of ${LIFECYCLES.join(" | ")}`);
      continue;
    }
    const spec = scalar(fm, "hardfork_spec");
    if (!spec || !/^https:\/\/specs\.optimism\.io\//.test(spec)) {
      err(`${file}: hardfork_spec must be a rendered specs.optimism.io URL (link policy)`);
      continue;
    }

    const entry: HardforkEntry = {
      file,
      name,
      lifecycle,
      spec,
      minVersions: [],
    };

    for (const [key, prop] of [
      ["hardfork_activation_mainnet", "activationMainnet"],
      ["hardfork_activation_sepolia", "activationSepolia"],
    ] as const) {
      const v = scalar(fm, key);
      if (v !== undefined) {
        if (!/^\d+$/.test(v)) {
          err(`${file}: ${key} must be a unix timestamp integer, got "${v}"`);
          continue;
        }
        entry[prop] = Number(v);
      }
    }
    if (lifecycle !== "development") {
      if (entry.activationMainnet === undefined)
        err(`${file}: hardfork_activation_mainnet is required when lifecycle is "${lifecycle}"`);
      if (entry.activationSepolia === undefined)
        err(`${file}: hardfork_activation_sepolia is required when lifecycle is "${lifecycle}"`);
    }

    entry.governance = scalar(fm, "hardfork_governance");
    entry.notice = scalar(fm, "hardfork_notice");
    if (entry.notice && !entry.notice.startsWith("/")) {
      err(`${file}: hardfork_notice must be a root-relative internal link`);
    }
    const upgradeNumber = scalar(fm, "hardfork_upgrade_number");
    if (upgradeNumber !== undefined) {
      if (!/^\d+[a-z]?$/.test(upgradeNumber)) err(`${file}: hardfork_upgrade_number must be a number`);
      else entry.upgradeNumber = Number(upgradeNumber);
    }
    entry.minVersionsSource = scalar(fm, "hardfork_min_versions_source");
    const mv = fm.get("hardfork_min_versions");
    if (Array.isArray(mv)) {
      for (const item of mv) {
        const parts = item.split(/\s+/);
        if (parts.length !== 2) {
          err(`${file}: hardfork_min_versions entries must be "<component> <version>", got "${item}"`);
          continue;
        }
        entry.minVersions.push({ component: parts[0], version: parts[1] });
      }
      if (entry.minVersions.length > 0 && !entry.minVersionsSource) {
        err(`${file}: hardfork_min_versions requires hardfork_min_versions_source (cite the notice or release)`);
      }
    }

    entries.push(entry);
  }
  if (entries.length === 0) fail(`setup error: no registry pages found in ${hardforksDir}`);
  return entries;
}

// ---------------------------------------------------------------------------
// superchain-registry cross-validation
// ---------------------------------------------------------------------------

// [hardforks] keys in superchain.toml that are not L2 hardforks in the fork
// series (L1-schedule shims, per-fork feature toggles).
const NON_FORK_KEYS = new Set(["pectra_blob_schedule_time", "keep_karst_upgrade_gas"]);

interface RegistryData {
  // fork name -> activation timestamp, per superchain target
  mainnet: Map<string, number>;
  sepolia: Map<string, number>;
  // genesis anchors of the flagship chain (OP Mainnet / OP Sepolia), used for
  // "around block" estimates (2-second block time since Bedrock).
  anchors: { mainnet: { time: number; block: number }; sepolia: { time: number; block: number } };
}

function parseHardforksToml(file: string): Map<string, number> {
  const out = new Map<string, number>();
  const raw = fs.readFileSync(file, "utf-8");
  const section = raw.match(/\[hardforks\]([\s\S]*?)(\n\[|$)/);
  if (!section) fail(`setup error: no [hardforks] section in ${file}`);
  for (const line of section![1].split(/\r?\n/)) {
    const m = line.match(/^\s*([a-z0-9_]+)\s*=\s*(\S+)/);
    if (!m) continue;
    const [, key, value] = m;
    if (NON_FORK_KEYS.has(key)) continue;
    const fork = key.replace(/_time$/, "");
    if (!/^\d+$/.test(value)) continue;
    out.set(fork, Number(value));
  }
  return out;
}

function parseGenesisAnchor(file: string): { time: number; block: number } {
  const raw = fs.readFileSync(file, "utf-8");
  const time = raw.match(/^\s*l2_time\s*=\s*(\d+)/m);
  const l2 = raw.match(/\[genesis\.l2\]([\s\S]*?)(\n\s*\[|$)/);
  const block = l2 && l2[1].match(/^\s*number\s*=\s*(\d+)/m);
  if (!time || !block) fail(`setup error: could not read genesis anchor from ${file}`);
  return { time: Number(time![1]), block: Number(block![1]) };
}

function loadRegistry(): RegistryData {
  const cfg = path.join(registryDir, "superchain", "configs");
  if (!fs.existsSync(cfg)) {
    fail(
      `setup error: superchain-registry not found at ${registryDir}.\n` +
        `Initialize the monorepo submodule (git submodule update --init superchain-registry) ` +
        `or pass --registry <path>.`,
    );
  }
  return {
    mainnet: parseHardforksToml(path.join(cfg, "mainnet", "superchain.toml")),
    sepolia: parseHardforksToml(path.join(cfg, "sepolia", "superchain.toml")),
    anchors: {
      mainnet: parseGenesisAnchor(path.join(cfg, "mainnet", "op.toml")),
      sepolia: parseGenesisAnchor(path.join(cfg, "sepolia", "op.toml")),
    },
  };
}

function crossValidate(entries: HardforkEntry[], reg: RegistryData) {
  const byName = new Map(entries.map((e) => [e.name, e]));

  // Every fork the registry schedules must have a page, and the page's row must
  // match the registry exactly.
  for (const [target, forks] of [
    ["mainnet", reg.mainnet],
    ["sepolia", reg.sepolia],
  ] as const) {
    for (const [fork, ts] of forks) {
      const entry = byName.get(fork);
      if (!entry) {
        err(`superchain-registry schedules "${fork}" on ${target} but there is no registry page for it`);
        continue;
      }
      const val = target === "mainnet" ? entry.activationMainnet : entry.activationSepolia;
      if (val !== ts) {
        err(
          `${entry.file}: hardfork_activation_${target} = ${val} does not match ` +
            `superchain-registry (${ts}). The registry is the source of truth.`,
        );
      }
    }
  }

  // Any nonzero activation stated on a page must exist in the registry — a page
  // may not invent an activation. (Zero means genesis-active, predating the
  // superchain-wide hardfork defaults; e.g. Regolith.)
  for (const e of entries) {
    for (const [target, forks] of [
      ["mainnet", reg.mainnet],
      ["sepolia", reg.sepolia],
    ] as const) {
      const val = target === "mainnet" ? e.activationMainnet : e.activationSepolia;
      if (val !== undefined && val !== 0 && forks.get(e.name) !== val) {
        err(
          `${e.file}: hardfork_activation_${target} = ${val} is not in the ` +
            `superchain-registry ${target} config`,
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

const GENERATED_HEADER = (source: string) => `{/*
  AUTO-GENERATED — DO NOT EDIT.
  Generated by scripts/generate-hardforks.ts from the frontmatter of ${source}.
  Activation data is validated against the superchain-registry
  (superchain/configs/{mainnet,sepolia}/superchain.toml).
  Regenerate with: pnpm gen:hardforks
*/}
`;

function fmtDate(ts: number): string {
  return new Date(ts * 1000).toUTCString().replace(/GMT$/, "UTC");
}

function fmtActivation(
  ts: number | undefined,
  anchor: { time: number; block: number },
): string {
  if (ts === undefined) return "Not scheduled";
  if (ts === 0) return "Genesis / at Bedrock";
  const approxBlock = anchor.block + Math.floor((ts - anchor.time) / 2);
  return `${fmtDate(ts)} (\`${ts}\`, ≈ block ${approxBlock})`;
}

function lifecycleLabel(l: Lifecycle): string {
  return { active: "Active", scheduled: "Scheduled", development: "In development" }[l];
}

function renderSummary(entries: HardforkEntry[], reg: RegistryData): string {
  // Newest first: in-development forks on top, then by mainnet activation desc.
  const sorted = [...entries].sort((a, b) => {
    const av = a.activationMainnet ?? Number.MAX_SAFE_INTEGER;
    const bv = b.activationMainnet ?? Number.MAX_SAFE_INTEGER;
    return bv - av;
  });
  const rows = sorted.map((e) => {
    const title = e.name.charAt(0).toUpperCase() + e.name.slice(1);
    return (
      `| [${title}](/op-stack/protocol/hardforks/${e.name}) ` +
      `| ${lifecycleLabel(e.lifecycle)} ` +
      `| ${fmtActivation(e.activationMainnet, reg.anchors.mainnet)} ` +
      `| ${fmtActivation(e.activationSepolia, reg.anchors.sepolia)} |`
    );
  });
  return (
    GENERATED_HEADER("op-stack/protocol/hardforks/*.mdx") +
    `\n| Hardfork | Status | [Mainnet activation](https://github.com/ethereum-optimism/superchain-registry/blob/main/superchain/configs/mainnet/superchain.toml) | [Sepolia activation](https://github.com/ethereum-optimism/superchain-registry/blob/main/superchain/configs/sepolia/superchain.toml) |\n` +
    `| --- | --- | --- | --- |\n` +
    rows.join("\n") +
    `\n\nActivation timestamps are the superchain-wide defaults from the ` +
    `[superchain-registry](https://github.com/ethereum-optimism/superchain-registry); ` +
    `block numbers are estimates for OP Mainnet and OP Sepolia (2-second blocks since Bedrock). ` +
    `Individual chains outside those defaults set their own activation times in their chain config.\n`
  );
}

function renderFork(e: HardforkEntry, reg: RegistryData): string {
  const rows: string[] = [];
  rows.push(`| Status | ${lifecycleLabel(e.lifecycle)} |`);
  rows.push(`| Governing spec | [specs.optimism.io](${e.spec}) |`);
  if (e.governance) rows.push(`| Governance approval | [proposal](${e.governance}) |`);
  if (e.notice) {
    const label = e.upgradeNumber !== undefined ? `Upgrade ${e.upgradeNumber} notice` : "View notice";
    rows.push(`| Operator notice | [${label}](${e.notice}) |`);
  }
  rows.push(
    `| Mainnet activation (superchain default) | ${fmtActivation(e.activationMainnet, reg.anchors.mainnet)} |`,
  );
  rows.push(
    `| Sepolia activation (superchain default) | ${fmtActivation(e.activationSepolia, reg.anchors.sepolia)} |`,
  );

  let out =
    GENERATED_HEADER(e.file) +
    `\n| | |\n| --- | --- |\n` +
    rows.join("\n") +
    "\n";

  if (e.minVersions.length > 0) {
    out += `\nMinimum component versions (from ${e.minVersionsSource}):\n\n`;
    out += `| Component | Minimum version |\n| --- | --- |\n`;
    for (const { component, version } of e.minVersions) {
      out += `| \`${component}\` | \`${version}\` |\n`;
    }
  }
  return out;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

const entries = loadEntries();
const registry = loadRegistry();
crossValidate(entries, registry);

if (errors.length > 0) {
  console.error(`hardfork registry validation failed (${errors.length} error(s)):\n`);
  for (const e of errors) console.error(`  - ${e}`);
  process.exit(1);
}

const outputs = new Map<string, string>();
outputs.set("summary.mdx", renderSummary(entries, registry));
for (const e of entries) outputs.set(`${e.name}.mdx`, renderFork(e, registry));

if (checkMode) {
  const stale: string[] = [];
  for (const [file, content] of outputs) {
    const p = path.join(outDir, file);
    if (!fs.existsSync(p) || fs.readFileSync(p, "utf-8") !== content) stale.push(file);
  }
  const onDisk = fs.existsSync(outDir) ? fs.readdirSync(outDir).filter((f) => f.endsWith(".mdx")) : [];
  for (const f of onDisk) if (!outputs.has(f)) stale.push(`${f} (orphaned)`);
  if (stale.length > 0) {
    console.error(
      `hardfork snippets are stale: ${stale.join(", ")}\n` +
        `Regenerate with: pnpm gen:hardforks (from docs/public-docs/)`,
    );
    process.exit(1);
  }
  console.log(`hardfork registry: ${entries.length} pages valid, snippets up to date`);
} else {
  fs.mkdirSync(outDir, { recursive: true });
  for (const [file, content] of outputs) {
    fs.writeFileSync(path.join(outDir, file), content);
  }
  const onDisk = fs.readdirSync(outDir).filter((f) => f.endsWith(".mdx"));
  for (const f of onDisk) if (!outputs.has(f)) fs.unlinkSync(path.join(outDir, f));
  console.log(`hardfork registry: ${entries.length} pages valid, wrote ${outputs.size} snippets to snippets/generated/hardforks/`);
}
