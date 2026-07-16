// Redirect + internal-link lint: keeps docs.json redirects honest and internal
// links resolvable.
//
// Checks (Option A slice 2 of the canonical-homes plan):
//   R1. No duplicate redirect sources.
//   R2. No redirects to non-existent targets: every internal destination must
//       resolve to a page on disk.
//   R3. No chained redirects: a destination must not itself be a redirect
//       source (point at the final page directly).
//   R4. No redirect source that shadows a live page (a source that still has
//       an .mdx on disk is almost always a half-finished move).
//   R5. A page deleted or moved in a PR must gain a redirect in the same PR:
//       every .mdx page present at the merge-base with the base branch but
//       absent from HEAD must appear as a redirect source in docs.json.
//   R6. No internal links to non-existent paths: every absolute internal link
//       in .mdx content must resolve to a page, a redirect source, or a static
//       file in the repo.
//
// Pre-existing R4/R6 violations on develop are grandfathered in
// redirect-lint-baseline.json so the check blocks NEW violations immediately;
// the baseline shrinks over time and stale baseline entries are errors.
//
// Environment:
//   DOCS_LINT_BASE_REF   base ref for the deleted-page check (default: origin/develop)
//   DOCS_LINT_SKIP_DIFF  set to 1 to skip R5 (local shallow clones only — CI must not set this)
//
// Usage:
//   bun docs/public-docs/scripts/lint/validate-redirects.ts   (from monorepo root, CI)
//   pnpm lint:redirects                                        (from docs/public-docs)
//
// Exit codes: 0 = clean, 1 = violations, 2 = setup error.

import * as fs from "fs";
import * as path from "path";
import { execFileSync } from "child_process";
import {
  NON_PAGE_DIRS,
  collectAllFiles,
  collectDiskPages,
  findDocsRoot,
  loadDocsJson,
  normalizePath,
  report,
  resolvesToPage,
} from "./common";

interface Baseline {
  shadowedSources: string[];
  deadInternalLinks: Record<string, string[]>;
}

const docsRoot = findDocsRoot();
const docsJson = loadDocsJson(docsRoot);
const diskPages = collectDiskPages(docsRoot);
const allFiles = collectAllFiles(docsRoot);

const baselinePath = path.join(docsRoot, "scripts", "lint", "redirect-lint-baseline.json");
const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf-8")) as Baseline;

const errors: string[] = [];

const redirects = docsJson.redirects ?? [];
const sources = redirects.map((r) => normalizePath(r.source));
const sourceSet = new Set(sources);

const isExternal = (dest: string): boolean =>
  /^[a-z][a-z0-9+.-]*:/i.test(dest) || dest.startsWith("//");

// ─── R1: duplicate redirect sources ───────────────────────────────────────────

// Anchor- and query-specific sources are deliberate (Mintlify redirects are
// client-side and may target different destinations per anchor), so the
// duplicate key keeps anchors/queries and only normalizes slashes.
const seen = new Map<string, number>();
for (const r of redirects) {
  const key = r.source.replace(/^\/+/, "").replace(/\/+$/, "");
  seen.set(key, (seen.get(key) ?? 0) + 1);
}
for (const [s, count] of seen) {
  if (count > 1) errors.push(`duplicate redirect source: "/${s}" appears ${count} times`);
}

// ─── R2 + R3: dead targets and chained redirects ──────────────────────────────

for (const r of redirects) {
  if (isExternal(r.destination)) continue;
  const dest = normalizePath(r.destination);
  if (resolvesToPage(dest, diskPages)) continue;
  if (sourceSet.has(dest)) {
    errors.push(
      `chained redirect: "${r.source}" -> "${r.destination}" points at another redirect's ` +
        `source — point it at the final page directly`,
    );
  } else {
    errors.push(
      `dead redirect target: "${r.source}" -> "${r.destination}" does not resolve to any page`,
    );
  }
}

// ─── R4: redirect sources shadowing live pages ────────────────────────────────

const baselineShadowed = new Set(baseline.shadowedSources.map(normalizePath));
for (const s of new Set(sources)) {
  if (!diskPages.has(s)) continue; // exact page match only; index form can't collide with a source
  if (baselineShadowed.has(s)) continue;
  errors.push(
    `redirect source shadows a live page: "/${s}" is a redirect source but "${s}.mdx" still exists`,
  );
}
for (const s of baselineShadowed) {
  if (!diskPages.has(s) || !sourceSet.has(s)) {
    errors.push(
      `redirect-lint-baseline.json: stale shadowedSources entry "/${s}" — no longer a violation, remove it`,
    );
  }
}

// ─── R5: deleted/moved pages must gain a redirect in the same PR ──────────────

if (process.env.DOCS_LINT_SKIP_DIFF === "1") {
  console.warn("warning: DOCS_LINT_SKIP_DIFF=1 — skipping the deleted-page redirect check (R5)");
} else {
  const baseRef = process.env.DOCS_LINT_BASE_REF ?? "origin/develop";
  let deleted: string[];
  try {
    const mergeBase = execFileSync("git", ["merge-base", baseRef, "HEAD"], {
      cwd: docsRoot,
      encoding: "utf-8",
    }).trim();
    // --no-renames: a rename shows up as D+A, so moved pages are covered by the
    // same rule as deletions. Tree-level diff only — safe on blobless clones.
    deleted = execFileSync(
      "git",
      ["diff", "--name-status", "--no-renames", `${mergeBase}..HEAD`, "--", "."],
      { cwd: docsRoot, encoding: "utf-8" },
    )
      .split("\n")
      .filter((line) => line.startsWith("D\t"))
      .map((line) => line.slice(2).trim())
      .filter((p) => p.endsWith(".mdx"));
  } catch (err) {
    console.error(
      `error: could not diff against "${baseRef}" for the deleted-page check: ${String(err)}\n` +
        "fetch the base ref, set DOCS_LINT_BASE_REF, or (locally only) set DOCS_LINT_SKIP_DIFF=1",
    );
    process.exit(2);
  }
  const docsPrefix = path.relative(
    execFileSync("git", ["rev-parse", "--show-toplevel"], { cwd: docsRoot, encoding: "utf-8" }).trim(),
    docsRoot,
  );
  for (const repoRel of deleted) {
    if (!repoRel.startsWith(`${docsPrefix}/`)) continue;
    const rel = repoRel.slice(docsPrefix.length + 1);
    const topDir = rel.split("/")[0];
    if (NON_PAGE_DIRS.has(topDir)) continue;
    const page = rel.slice(0, -".mdx".length);
    if (!sourceSet.has(page)) {
      errors.push(
        `deleted/moved page without redirect: "${rel}" was removed relative to ${baseRef} ` +
          `but "/${page}" is not a redirect source in docs.json — add the redirect in this PR`,
      );
    }
  }
}

// ─── R6: internal links must resolve ──────────────────────────────────────────

// Match absolute internal link targets in markdown links and JSX attributes.
const LINK_RE = /(?:\]\(|href=["']|to=["'])(\/[^)\s"'#?]*)/g;

/** Strip fenced code blocks and inline code so example links aren't linted. */
function stripCode(text: string): string {
  return text.replace(/```[\s\S]*?```/g, "").replace(/`[^`\n]*`/g, "");
}

const baselineLinks = new Map<string, Set<string>>();
for (const [file, targets] of Object.entries(baseline.deadInternalLinks)) {
  baselineLinks.set(file, new Set(targets));
}

const linkOk = (target: string): boolean => {
  if (target.startsWith("//")) return true; // protocol-relative external URL
  const t = normalizePath(target);
  if (resolvesToPage(t, diskPages)) return true;
  if (sourceSet.has(t)) return true; // redirect covers it
  if (allFiles.has(t)) return true; // static asset (e.g. public/img/...)
  return false;
};

const mdxFiles = [...diskPages].sort().map((p) => `${p}.mdx`);
const snippetFiles: string[] = [];
const snippetsDir = path.join(docsRoot, "snippets");
if (fs.existsSync(snippetsDir)) {
  const walk = (dir: string): void => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(abs);
      else if (entry.name.endsWith(".mdx")) snippetFiles.push(path.relative(docsRoot, abs));
    }
  };
  walk(snippetsDir);
}

const usedBaseline = new Map<string, Set<string>>();
for (const file of [...mdxFiles, ...snippetFiles]) {
  const text = stripCode(fs.readFileSync(path.join(docsRoot, file), "utf-8"));
  for (const match of text.matchAll(LINK_RE)) {
    const target = match[1];
    if (linkOk(target)) continue;
    const grandfathered = baselineLinks.get(file);
    if (grandfathered?.has(target)) {
      if (!usedBaseline.has(file)) usedBaseline.set(file, new Set());
      usedBaseline.get(file)!.add(target);
      continue;
    }
    errors.push(
      `dead internal link: ${file} links to "${target}" which resolves to no page, redirect, or file`,
    );
  }
}

// Baseline hygiene: entries that no longer occur must be removed.
for (const [file, targets] of baselineLinks) {
  for (const target of targets) {
    if (!usedBaseline.get(file)?.has(target)) {
      errors.push(
        `redirect-lint-baseline.json: stale deadInternalLinks entry ${file} -> "${target}" — ` +
          "no longer a violation, remove it",
      );
    }
  }
}

const baselineCount = [...baselineLinks.values()].reduce((n, s) => n + s.size, 0);
report(
  errors,
  `redirect lint: OK — ${redirects.length} redirects, ${mdxFiles.length + snippetFiles.length} ` +
    `files scanned, ${baselineCount + baselineShadowed.size} grandfathered violation(s) remaining`,
);
