#!/usr/bin/env bun
// gen-cli — generates the versioned component CLI reference for docs.optimism.io.
//
// One generator for every component, per the "Component reference" section
// of docs/public-docs/op-stack/contribute/content-guide.mdx: walk the
// release artifact's `--help` tree, parse it (clap or urfave/cli), and emit
// the pages, the docs.json version entry, and the manifest for one release
// line of one component.
//
//   bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --tag op-reth/v2.4.4 --image auto
//   bun docs/public-docs/scripts/gen-cli/main.ts --component op-node --tag op-node/v1.19.7 --bin ./op-node
//   bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --develop --commit <sha> --bin ./op-reth
//   bun docs/public-docs/scripts/gen-cli/main.ts --component op-reth --tag op-reth/v2.4.4 --image auto --check
//
// Sources: --image <ref|auto> (docker run; auto = the component's published
// image at the tag), --bin <path>, or --help-json <file> (a recording made
// with --dump-help-json). --check regenerates in memory and fails on any
// difference from the committed pages, nav entry, or manifest hash.
//
// Deleted pages are printed at the end; each needs a redirect in docs.json in
// the same PR (the redirect lint enforces it). --page-map <file> writes the
// command → URL(#anchor) map, which the migration redirects are built from.

import { createHash } from "crypto";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { COMPONENTS, findComponent, parseTag } from "./components";
import { emitPages, type Page, type Provenance } from "./emit";
import { spliceNav, versionGroups } from "./nav";
import { BinarySource, FixtureSource, type HelpSource, ImageSource, RecordingSource } from "./source";
import { walkCommands } from "./walk";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_DOCS_ROOT = path.resolve(SCRIPT_DIR, "..", "..");
const RELEASES_URL = "https://github.com/ethereum-optimism/optimism/releases/tag/";
const COMMIT_URL = "https://github.com/ethereum-optimism/optimism/commit/";

interface Args {
  component: string;
  tag: string | null;
  develop: boolean;
  commit: string | null;
  bin: string | null;
  image: string | null;
  helpJson: string | null;
  dumpHelpJson: string | null;
  pageMap: string | null;
  docsRoot: string;
  check: boolean;
}

function usage(msg?: string): never {
  if (msg) console.error(`error: ${msg}`);
  console.error(
    "usage: main.ts --component <name> (--tag <component>/vX.Y.Z | --develop --commit <sha>) (--image <ref|auto> | --bin <path> | --help-json <file>) [--check] [--dump-help-json <file>] [--page-map <file>] [--docs-dir <path>]",
  );
  console.error(`components: ${COMPONENTS.map((c) => c.name).join(", ")}`);
  process.exit(2);
}

function parseArgs(argv: string[]): Args {
  const a: Args = {
    component: "",
    tag: null,
    develop: false,
    commit: null,
    bin: null,
    image: null,
    helpJson: null,
    dumpHelpJson: null,
    pageMap: null,
    docsRoot: DEFAULT_DOCS_ROOT,
    check: false,
  };
  for (let i = 0; i < argv.length; i++) {
    const x = argv[i];
    const next = () => {
      const v = argv[++i];
      if (v === undefined) usage(`${x} needs a value`);
      return v;
    };
    switch (x) {
      case "--component": a.component = next(); break;
      case "--tag": a.tag = next(); break;
      case "--develop": a.develop = true; break;
      case "--commit": a.commit = next(); break;
      case "--bin": a.bin = next(); break;
      case "--image": a.image = next(); break;
      case "--help-json": a.helpJson = next(); break;
      case "--dump-help-json": a.dumpHelpJson = next(); break;
      case "--page-map": a.pageMap = next(); break;
      case "--docs-dir": a.docsRoot = fs.realpathSync(path.resolve(next())); break;
      case "--check": a.check = true; break;
      default: usage(`unknown argument: ${x}`);
    }
  }
  if (!a.component) usage("--component is required");
  if ([a.bin, a.image, a.helpJson].filter(Boolean).length !== 1) usage("exactly one of --bin, --image, --help-json is required");
  if (a.develop) {
    if (!a.commit || !/^[0-9a-f]{7,40}$/.test(a.commit)) usage("--develop needs --commit <sha>");
  } else if (!a.tag) {
    usage("--tag is required (or --develop --commit <sha>)");
  }
  return a;
}

function sha256Tree(pages: Page[]): string {
  const h = createHash("sha256");
  for (const p of [...pages].sort((x, y) => (x.relPath < y.relPath ? -1 : 1))) {
    h.update(p.relPath);
    h.update("\0");
    h.update(p.content);
    h.update("\0");
  }
  return h.digest("hex");
}

/**
 * Resolve `segments` under `root` and require the result to stay inside it.
 * Every path the generator reads or writes goes through here: the docs root
 * comes from --docs-dir, page names from a component's --help output (already
 * validated against SUBCOMMAND_NAME), and the version line from a validated
 * tag, so nothing can escape the docs tree even if one of those inputs is
 * hostile. Same containment rule as scripts/lint/common.ts resolveInside.
 */
function inside(root: string, ...segments: string[]): string {
  const resolved = path.resolve(root, ...segments);
  if (resolved !== root && !resolved.startsWith(root + path.sep)) {
    throw new Error(`refusing path outside ${root}: ${path.join(...segments)}`);
  }
  return resolved;
}

const PAGE_FILE_RE = /^[a-z0-9][a-z0-9._-]*\.mdx$/;

/** The generated pages committed in a version directory, by file name. Only well-formed page names are read. */
function readTree(docsRoot: string, dir: string): Map<string, string> {
  const out = new Map<string, string>();
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isFile() || !PAGE_FILE_RE.test(entry.name)) continue;
    out.set(entry.name, fs.readFileSync(inside(docsRoot, dir, entry.name), "utf8"));
  }
  return out;
}

function main(): void {
  const args = parseArgs(process.argv.slice(2));
  const component = findComponent(args.component);

  let line: string;
  let ref: string;
  let refUrl: string;
  if (args.develop) {
    line = "develop";
    ref = args.commit!;
    refUrl = `${COMMIT_URL}${ref}`;
  } else {
    const parsed = parseTag(component.name, args.tag!);
    line = parsed.line;
    ref = args.tag!;
    refUrl = `${RELEASES_URL}${encodeURIComponent(ref)}`;
  }

  let source: HelpSource;
  let artifact: string;
  if (args.image) {
    let image = args.image;
    if (image === "auto") {
      if (component.artifact.kind !== "image") usage(`${component.name} publishes no image; use --bin with the downloaded release binary`);
      if (args.develop) usage("--image auto needs a release tag; pass an explicit image ref or --bin for develop");
      image = `${component.artifact.repository}:${parseTag(component.name, args.tag!).version}`;
    }
    source = new ImageSource(image);
    artifact = `image ${image}`;
  } else if (args.bin) {
    source = new BinarySource(args.bin);
    artifact = `binary ${path.basename(args.bin)}`;
  } else {
    source = new FixtureSource(args.helpJson!);
    artifact = source.describe();
  }
  const recorder = new RecordingSource(source);

  console.log(`component: ${component.name} (${component.framework}); line ${line}; ${artifact}`);
  const binaryVersion = recorder.version();
  const tree = walkCommands(component, recorder);
  const commandCount = countNodes(tree);
  console.log(`walked ${commandCount} commands from --help`);

  if (args.dumpHelpJson) {
    fs.writeFileSync(
      args.dumpHelpJson,
      JSON.stringify({ binary: component.binary ?? component.name, tag: ref, source: artifact, version: binaryVersion, help: recorder.recorded }, null, 1) + "\n",
    );
    console.log(`recorded help tree to ${args.dumpHelpJson}`);
  }

  const provenance: Provenance = { component, line, ref, refUrl, artifact, binaryVersion };
  const { pages, pageMap } = emitPages(tree, provenance);
  const sha256 = sha256Tree(pages);

  const docsRoot = args.docsRoot;
  if (!fs.existsSync(path.join(docsRoot, "docs.json"))) usage(`${docsRoot} is not a docs root (no docs.json)`);
  const versionDir = inside(docsRoot, "reference", component.name, line);
  const existing = readTree(docsRoot, versionDir);
  const generated = new Map(pages.map((p) => [p.relPath, p.content]));
  const added = [...generated.keys()].filter((k) => !existing.has(k)).sort();
  const removed = [...existing.keys()].filter((k) => !generated.has(k)).sort();
  const changed = [...generated.keys()].filter((k) => existing.has(k) && existing.get(k) !== generated.get(k)).sort();

  const docsJsonPath = inside(docsRoot, "docs.json");
  const docsJson = JSON.parse(fs.readFileSync(docsJsonPath, "utf8"));
  const groups = versionGroups(pages);
  // The nav check compares the whole navigation after a splice into a clone,
  // so a hand edit to a version's default/tag, the dropdown's icon or the
  // tab's position is caught, not only the page list.
  const navChanged = spliceNav(JSON.parse(JSON.stringify(docsJson)), component, line, groups);

  const manifestPath = inside(docsRoot, "scripts", "gen-cli", "manifest.json");
  // A release line records its tag; the develop line records the commit it
  // was generated from (never under a `tag` key: the reference lint checks
  // every manifest `tag` against git tags).
  const manifest: Record<string, Record<string, { tag?: string; commit?: string; artifact: string; version: string | null; sha256: string }>> = fs.existsSync(manifestPath)
    ? JSON.parse(fs.readFileSync(manifestPath, "utf8"))
    : {};
  const recorded = manifest[component.name]?.[line];

  if (args.pageMap) {
    fs.writeFileSync(args.pageMap, JSON.stringify(pageMap, null, 1) + "\n");
    console.log(`wrote page map (${pageMap.length} commands) to ${args.pageMap}`);
  }

  if (args.check) {
    const problems: string[] = [];
    if (!recorded) problems.push(`manifest.json has no ${component.name}/${line} entry`);
    if (added.length || removed.length || changed.length) {
      problems.push(`committed pages differ from regeneration (added ${added.length}, removed ${removed.length}, changed ${changed.length})`);
      for (const k of added) problems.push(`  new:     ${k}`);
      for (const k of removed) problems.push(`  stale:   ${k}`);
      for (const k of changed) problems.push(`  changed: ${k}`);
    }
    if (navChanged) problems.push("docs.json version entry differs from regeneration");
    if (recorded && recorded.sha256 !== sha256) problems.push(`manifest sha256 ${recorded.sha256} != regenerated ${sha256}`);
    if (problems.length) {
      for (const p of problems) console.error(`check: ${p}`);
      console.error("check failed. If the component moved past the manifest tag this is expected drift: regenerate at the next finalized release.");
      process.exit(1);
    }
    console.log(`check OK: ${pages.length} pages, nav entry, and manifest (${recorded!.tag ?? recorded!.commit}) match the --help tree`);
    return;
  }

  // Write pages, contained to the version directory.
  fs.mkdirSync(versionDir, { recursive: true });
  for (const p of pages) fs.writeFileSync(inside(versionDir, p.relPath), p.content);
  for (const k of removed) fs.rmSync(inside(versionDir, k));
  fs.mkdirSync(path.dirname(manifestPath), { recursive: true });

  const navReallyChanged = spliceNav(docsJson, component, line, groups);
  if (navReallyChanged) fs.writeFileSync(docsJsonPath, `${JSON.stringify(docsJson, null, 2)}\n`);

  manifest[component.name] = manifest[component.name] ?? {};
  manifest[component.name][line] = args.develop
    ? { commit: ref, artifact, version: binaryVersion, sha256 }
    : { tag: ref, artifact, version: binaryVersion, sha256 };
  fs.writeFileSync(manifestPath, `${JSON.stringify(sortKeys(manifest), null, 2)}\n`);

  console.log(
    `wrote ${pages.length} pages to reference/${component.name}/${line} (${added.length} new, ${changed.length} changed, ${removed.length} removed); nav ${navReallyChanged ? "updated" : "unchanged"}; manifest sha256 ${sha256.slice(0, 12)}…`,
  );
  if (removed.length) {
    console.log("\nDeleted pages — each URL below needs a redirect in docs.json in this same PR:");
    for (const k of removed) console.log(`  /reference/${component.name}/${line}/${k.replace(/\.mdx$/, "")}`);
  }
}

function countNodes(tree: { children: unknown[] }): number {
  let n = 1;
  for (const c of tree.children as Array<{ children: unknown[] }>) n += countNodes(c);
  return n;
}

function sortKeys<T>(value: T): T {
  if (Array.isArray(value)) return value.map(sortKeys) as unknown as T;
  if (value && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const k of Object.keys(value as Record<string, unknown>).sort()) out[k] = sortKeys((value as Record<string, unknown>)[k]);
    return out as T;
  }
  return value;
}

main();
