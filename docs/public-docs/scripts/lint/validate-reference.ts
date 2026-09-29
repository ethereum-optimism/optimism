// Reference lint: enforces the "Component reference" section of the content
// guide (op-stack/contribute/content-guide.mdx).
//
// Checks:
//   R1. Every .mdx under the top-level reference/ directory is generated: the
//       first non-import content after frontmatter is a comment block carrying
//       the DO NOT EDIT marker. Hand-written pages do not live under
//       reference/; they live in the persona tabs.
//   R2. Every <Unreleased component="…" [version="…"] /> callout names a
//       covered component and, when it names a version, a release that does
//       not exist yet. Once the git tag `<component>/<version>` exists, the
//       callout is stale and must be removed (the prose it guards is now
//       current). Callouts without a version are listed by the weekly CI
//       sweep for a maintainer to remove. Callouts the lint cannot
//       parse (expression attributes, a missing component) are errors, so a
//       callout can never be silently unlinted.
//   R3. Every `tag` recorded in a generator manifest (scripts/*/manifest.json)
//       is an existing git tag, so provenance lines can never name a release
//       that was not published.
//
// Deterministic, Node builtins only. R2 and R3 need the repository's tags:
// local tags first, then `git ls-remote --tags` as a fallback for shallow
// checkouts. When neither is available the lint skips R2/R3 with a visible
// warning locally, and fails with a setup error under CI (CI=true), so a
// tagless runner can never report a false pass.
//
// Usage:
//   bun docs/public-docs/scripts/lint/validate-reference.ts   (from monorepo root, CI)
//   pnpm lint:reference                                        (from docs/public-docs)
//
// Exit codes: 0 = clean, 1 = violations, 2 = setup error.

import { execFileSync } from "child_process";
import * as fs from "fs";
import * as path from "path";
import { collectAllFiles, findDocsRoot, report, resolveInside } from "./common";

const docsRoot = findDocsRoot();
const errors: string[] = [];

// Components covered by the convention. Keep in sync with the "Components
// covered" list in the content guide and the Props comment in
// snippets/unreleased.mdx. A callout naming anything else can never become
// stale (no tag will ever match), so it is rejected outright.
const COMPONENTS = new Set([
  "op-node",
  "op-batcher",
  "op-proposer",
  "op-challenger",
  "op-conductor",
  "op-supernode",
  "op-deployer",
  "op-reth",
  "kona-node",
]);

// ─── Git tags ─────────────────────────────────────────────────────────────────

function run(cmd: string, args: string[]): string | null {
  try {
    return execFileSync(cmd, args, {
      cwd: docsRoot,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
      timeout: 60_000,
    });
  } catch {
    return null;
  }
}

function gitTags(): { tags: Set<string>; source: string } | null {
  const local = run("git", ["tag", "--list"]);
  if (local !== null) {
    const tags = new Set(local.split("\n").map((t) => t.trim()).filter(Boolean));
    if (tags.size > 0) return { tags, source: "local" };
  }
  // Shallow checkout: ask the remote for the tag list without fetching objects.
  const remote = run("git", ["ls-remote", "--tags", "--refs", "origin"]);
  if (remote !== null) {
    const tags = new Set(
      remote
        .split("\n")
        .map((line) => line.split("\t")[1] ?? "")
        .filter((ref) => ref.startsWith("refs/tags/"))
        .map((ref) => ref.slice("refs/tags/".length)),
    );
    if (tags.size > 0) return { tags, source: "origin (ls-remote)" };
  }
  return null;
}

const tagInfo = gitTags();
if (tagInfo === null) {
  const msg =
    "no git tags available from the local checkout or from origin — " +
    "the Unreleased and manifest tag checks (R2, R3) need them";
  if (process.env.CI) {
    console.error(`error: ${msg}. Fetch tags (git fetch --tags) or unshallow the checkout.`);
    process.exit(2);
  }
  console.warn(`warning: ${msg}; skipping R2 and R3 in this run`);
}
const tags = tagInfo?.tags ?? null;

// ─── Files ────────────────────────────────────────────────────────────────────

const toPosix = (p: string): string => p.split(path.sep).join("/");
const allFiles = [...collectAllFiles(docsRoot)].map(toPosix);
const mdxFiles = allFiles.filter((f) => f.endsWith(".mdx")).sort();
const isReferencePage = (rel: string): boolean => rel.startsWith("reference/");

function readDocsFile(rel: string): string | null {
  const abs = resolveInside(docsRoot, ...rel.split("/"));
  if (abs === null) {
    errors.push(`path escapes docs root: "${rel}"`);
    return null;
  }
  return fs.readFileSync(abs, "utf-8");
}

/** Strip a UTF-8 BOM, leading whitespace, and a leading YAML frontmatter block. */
function stripFrontmatter(src: string): string {
  let s = src.replace(/^﻿/, "").replace(/^\s+/, "");
  if (!s.startsWith("---")) return s;
  const end = s.search(/\r?\n---[ \t]*(\r?\n|$)/);
  if (end === -1) return s;
  s = s.slice(end);
  s = s.replace(/^\r?\n---[ \t]*(\r?\n|$)/, "");
  return s;
}

// ─── R1: everything under reference/ is generated ─────────────────────────────

const GENERATED_MARKER = /DO NOT EDIT/;

for (const rel of mdxFiles) {
  if (!isReferencePage(rel)) continue;
  const src = readDocsFile(rel);
  if (src === null) continue;
  // The first non-import content after frontmatter must be the generator's
  // provenance comment. Blank lines and import lines may precede it.
  const lines = stripFrontmatter(src).split(/\r?\n/);
  let i = 0;
  while (i < lines.length && (lines[i].trim() === "" || /^import\s/.test(lines[i]))) i++;
  const rest = lines.slice(i).join("\n").trimStart();
  const firstBlock = rest.startsWith("{/*") ? rest.slice(0, rest.indexOf("*/}") + 3) : "";
  if (!GENERATED_MARKER.test(firstBlock)) {
    errors.push(
      `hand-written page under reference/: "${rel}" does not open with a generated DO NOT EDIT header — ` +
        `pages under reference/ are emitted by a generator; move hand-written content to a guide in the persona tabs`,
    );
  }
}

// ─── R2: no stale or unparseable <Unreleased> callouts ────────────────────────

// Pages exempt from the staleness check: the snippet's own usage comment, and
// the content guide, whose rendered example names a version so that readers
// can see the callout. Neither is a claim about a real unreleased change.
const R2_EXEMPT = new Set(["snippets/unreleased.mdx", "op-stack/contribute/content-guide.mdx"]);

const UNRELEASED_TAG_RE = /<Unreleased\b[^>]*>/g;
const ATTR_RE = /\b(component|version)\s*=\s*["']([^"']+)["']/g;
const SEMVER_RE = /^v\d+\.\d+\.\d+$/;

let unreleasedCount = 0;
for (const rel of mdxFiles) {
  if (R2_EXEMPT.has(rel)) continue;
  const src = readDocsFile(rel);
  if (src === null) continue;
  for (const m of src.matchAll(UNRELEASED_TAG_RE)) {
    unreleasedCount++;
    const tag = m[0];
    const attrs: Record<string, string> = {};
    for (const a of tag.matchAll(ATTR_RE)) attrs[a[1]] = a[2];
    const component = attrs.component;
    const version = attrs.version;
    const hasVersionAttr = /\bversion\s*=/.test(tag);
    if (!component || (hasVersionAttr && !version)) {
      errors.push(
        `${rel}: cannot parse ${tag.split(/\s+/).join(" ")} — <Unreleased> needs a literal, quoted component="…" attribute (and a literal, quoted version="…" if given)`,
      );
      continue;
    }
    if (isReferencePage(rel)) {
      errors.push(`${rel}: <Unreleased> is not allowed under reference/ (generated pages are never hand-edited)`);
      continue;
    }
    if (!COMPONENTS.has(component)) {
      errors.push(
        `${rel}: <Unreleased component="${component}"> — not a covered component; use one of: ${[...COMPONENTS].join(", ")}`,
      );
      continue;
    }
    if (!version) continue; // no version: the weekly CI sweep lists it for review
    if (!SEMVER_RE.test(version)) {
      errors.push(
        `${rel}: <Unreleased component="${component}" version="${version}"> — version must be a full semver with a leading v (e.g. v2.5.0)`,
      );
      continue;
    }
    if (tags !== null && tags.has(`${component}/${version}`)) {
      errors.push(
        `${rel}: stale <Unreleased component="${component}" version="${version}"> — ` +
          `${component}/${version} is published; remove the callout and let the prose read as current`,
      );
    }
  }
}

// ─── R3: manifest tags exist ─────────────────────────────────────────────────

function collectTags(value: unknown, sink: string[]): void {
  if (value === null || typeof value !== "object") return;
  for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
    if (k === "tag" && typeof v === "string") sink.push(v);
    else collectTags(v, sink);
  }
}

const manifests = allFiles.filter((f) => /^scripts\/[^/]+\/manifest\.json$/.test(f)).sort();
let manifestTagCount = 0;
for (const rel of manifests) {
  const src = readDocsFile(rel);
  if (src === null) continue;
  let parsed: unknown;
  try {
    parsed = JSON.parse(src);
  } catch (e) {
    errors.push(`${rel}: invalid JSON (${(e as Error).message})`);
    continue;
  }
  const found: string[] = [];
  collectTags(parsed, found);
  for (const tag of found) {
    manifestTagCount++;
    if (tags !== null && !tags.has(tag)) {
      errors.push(`${rel}: manifest tag "${tag}" is not a git tag — generators run only at published, finalized release tags`);
    }
  }
}

const tagNote = tags === null ? " (R2/R3 SKIPPED: no tags)" : ` (tags from ${tagInfo!.source})`;
report(
  errors,
  `reference lint: OK — ${mdxFiles.filter(isReferencePage).length} generated page(s) under reference/, ` +
    `${unreleasedCount} <Unreleased> callout(s), ${manifestTagCount} manifest tag(s) across ${manifests.length} manifest(s)${tagNote}`,
);
