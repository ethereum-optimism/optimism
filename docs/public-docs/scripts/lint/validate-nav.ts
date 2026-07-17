// Nav validator: keeps docs.json navigation and the on-disk page tree in sync.
//
// Checks (Option A slice 2 of the canonical-homes plan):
//   N1. No duplicate nav entries (same page listed twice anywhere in the nav).
//   N2. No nav entries without a file on disk (dead nav entries).
//   N3. Every routable .mdx on disk is reachable from the nav, or explicitly
//       allowlisted in nav-allowlist.json with a reason string (no orphans).
//   N4. The allowlist itself stays honest: entries must carry a reason, must
//       still exist on disk, and must still be orphans — otherwise they must
//       be removed.
//
// Usage:
//   bun docs/public-docs/scripts/lint/validate-nav.ts     (from monorepo root, CI)
//   pnpm lint:nav                                          (from docs/public-docs)
//
// Exit codes: 0 = clean, 1 = violations, 2 = setup error.

import * as fs from "fs";
import * as path from "path";
import {
  collectDiskPages,
  collectNavPages,
  findDocsRoot,
  loadDocsJson,
  report,
  resolvesToPage,
} from "./common";

interface AllowlistEntry {
  path: string;
  reason: string;
}

interface NavAllowlist {
  orphans: AllowlistEntry[];
  // Pre-existing duplicate nav entries grandfathered at guardrail introduction.
  // New duplicates are always errors; this list only shrinks.
  duplicateNavEntries: AllowlistEntry[];
}

const docsRoot = findDocsRoot();
const docsJson = loadDocsJson(docsRoot);
const navPages = collectNavPages(docsJson.navigation);
const diskPages = collectDiskPages(docsRoot);

const allowlistPath = path.join(docsRoot, "scripts", "lint", "nav-allowlist.json");
const allowlist = JSON.parse(fs.readFileSync(allowlistPath, "utf-8")) as NavAllowlist;

const errors: string[] = [];

// ─── N1: duplicate nav entries ────────────────────────────────────────────────

const allowedDuplicates = new Map<string, AllowlistEntry>();
for (const entry of allowlist.duplicateNavEntries) {
  allowedDuplicates.set(entry.path, entry);
}

const seen = new Map<string, number>();
for (const p of navPages) seen.set(p, (seen.get(p) ?? 0) + 1);
for (const [p, count] of seen) {
  if (count > 1 && !allowedDuplicates.has(p)) {
    errors.push(`duplicate nav entry: "${p}" appears ${count} times in docs.json navigation`);
  }
}
for (const entry of allowlist.duplicateNavEntries) {
  if ((seen.get(entry.path) ?? 0) <= 1) {
    errors.push(
      `nav-allowlist.json: stale duplicateNavEntries entry "${entry.path}" — no longer duplicated, remove it`,
    );
  }
  if (!entry.reason || entry.reason.trim() === "") {
    errors.push(`nav-allowlist.json: duplicateNavEntries entry "${entry.path}" has no reason string`);
  }
}

// ─── N2: nav entries without a file ───────────────────────────────────────────

for (const p of new Set(navPages)) {
  if (!resolvesToPage(p, diskPages)) {
    errors.push(`dead nav entry: "${p}" is in docs.json navigation but has no .mdx file`);
  }
}

// ─── N3: orphans (pages on disk missing from nav) ─────────────────────────────

// A page is reachable if the nav lists it directly, or lists its directory
// index form ("foo" covers foo/index.mdx and vice versa).
const navSet = new Set(navPages);
const reachable = (page: string): boolean => {
  if (navSet.has(page)) return true;
  if (page.endsWith("/index") && navSet.has(page.slice(0, -"/index".length))) return true;
  if (page === "index" && navSet.has("")) return true;
  return false;
};

const allowedOrphans = new Map<string, AllowlistEntry>();
for (const entry of allowlist.orphans) {
  allowedOrphans.set(entry.path, entry);
}

const orphans: string[] = [];
for (const page of [...diskPages].sort()) {
  if (reachable(page)) continue;
  if (allowedOrphans.has(page)) continue;
  orphans.push(page);
}
for (const o of orphans) {
  errors.push(
    `orphan page: "${o}.mdx" is not reachable from docs.json navigation — ` +
      `add it to the nav, or (deliberately) to scripts/lint/nav-allowlist.json with a reason`,
  );
}

// ─── N4: allowlist hygiene ────────────────────────────────────────────────────

for (const entry of allowlist.orphans) {
  if (!entry.reason || entry.reason.trim() === "") {
    errors.push(`nav-allowlist.json: entry "${entry.path}" has no reason string`);
  }
  if (!diskPages.has(entry.path)) {
    errors.push(
      `nav-allowlist.json: stale entry "${entry.path}" — file no longer exists, remove it from the allowlist`,
    );
  } else if (reachable(entry.path)) {
    errors.push(
      `nav-allowlist.json: stale entry "${entry.path}" — page is now in the nav, remove it from the allowlist`,
    );
  }
}
const allowPaths = allowlist.orphans.map((e) => e.path);
for (const p of new Set(allowPaths)) {
  if (allowPaths.filter((x) => x === p).length > 1) {
    errors.push(`nav-allowlist.json: duplicate entry "${p}"`);
  }
}

report(
  errors,
  `nav validator: OK — ${diskPages.size} pages on disk, ${navSet.size} distinct nav entries, ` +
    `${allowedOrphans.size} allowlisted orphan(s)`,
);
