// Shared helpers for the docs lint scripts (validate-nav.ts, validate-redirects.ts,
// validate-reference.ts).
//
// Both scripts are deterministic, dependency-free (Node builtins only), and
// runnable with either `tsx` (pnpm devDependency, like the generator scripts)
// or `bun` (pinned in the monorepo's mise.toml, used by CI).

import * as fs from "fs";
import * as path from "path";

// ─── Docs root resolution ─────────────────────────────────────────────────────

// Directories under the docs root that never contain routable pages.
// - snippets/    reusable MDX fragments, included into pages, never routed
// - public/      static assets
// - styles/      CSS
// - scripts/     tooling (this directory)
// - create-l2-rollup-example/  companion code for a tutorial, not docs pages
export const NON_PAGE_DIRS = new Set([
  "node_modules",
  ".git",
  "snippets",
  "public",
  "styles",
  "scripts",
  "create-l2-rollup-example",
]);

/**
 * Locate the docs root (the directory containing docs.json). Works whether the
 * script is invoked from the monorepo root (CI) or from docs/public-docs.
 */
export function findDocsRoot(): string {
  const candidates = [
    path.join(process.cwd(), "docs", "public-docs"),
    process.cwd(),
  ];
  for (const dir of candidates) {
    if (fs.existsSync(path.join(dir, "docs.json"))) return dir;
  }
  console.error(
    "error: could not find docs.json — run from the monorepo root or from docs/public-docs",
  );
  process.exit(2);
}

// ─── Path containment ─────────────────────────────────────────────────────────

/**
 * Resolve path segments against `root` and require the result to stay inside
 * `root` (or be `root` itself). Returns the resolved absolute path, or null
 * when it escapes. Cheap CWE-22 containment hardening: every path these
 * scripts read is derived from docs.json, committed baseline files, or
 * directory walks — all repo-controlled — but routing every fs access through
 * this check keeps that provable. Callers treat null as a lint/setup error
 * and skip the fs call.
 */
export function resolveInside(root: string, ...segments: string[]): string | null {
  const resolved = path.resolve(root, ...segments);
  if (resolved === root || resolved.startsWith(root + path.sep)) return resolved;
  return null;
}

// ─── docs.json ────────────────────────────────────────────────────────────────

export interface Redirect {
  source: string;
  destination: string;
}

export interface DocsJson {
  navigation: unknown;
  redirects: Redirect[];
}

export function loadDocsJson(docsRoot: string): DocsJson {
  const docsJsonPath = resolveInside(docsRoot, "docs.json");
  if (docsJsonPath === null) {
    console.error("error: docs.json path escapes docs root");
    process.exit(2);
  }
  const raw = fs.readFileSync(docsJsonPath, "utf-8");
  return JSON.parse(raw) as DocsJson;
}

// Every Mintlify navigation container key that can hold pages or further
// containers. `versions` is handled separately (see collectNavPages).
const NAV_CONTAINER_KEYS = ["tabs", "anchors", "dropdowns", "products", "menu", "groups", "pages"];

/**
 * Collect every page entry from the Mintlify navigation tree, in order.
 * Page entries are strings; containers are objects carrying one of the
 * NAV_CONTAINER_KEYS arrays or a `versions` array. Returned paths are
 * normalized (no leading slash).
 *
 * Versions: sibling versions of one container (the versioned component
 * references under the Reference tab) may legitimately list the same page —
 * a page shared across versions renders once at one URL, and the switcher
 * keeps the reader on it. Such a page is therefore collected once per
 * `versions` array, not once per version, so the duplicate-entry check does
 * not fire on it. The same page listed twice *within* one version, or in two
 * unrelated places, is still reported as a duplicate.
 */
export function collectNavPages(navigation: unknown): string[] {
  const pages: string[] = [];
  const walk = (node: unknown, sink: string[]): void => {
    if (typeof node === "string") {
      sink.push(normalizePath(node));
      return;
    }
    if (node === null || typeof node !== "object") return;
    const obj = node as Record<string, unknown>;
    if (Array.isArray(obj.versions)) {
      const seenAcrossVersions = new Set<string>();
      for (const version of obj.versions) {
        const inVersion: string[] = [];
        walk(version, inVersion);
        const seenInVersion = new Set<string>();
        for (const p of inVersion) {
          if (seenInVersion.has(p)) {
            sink.push(p); // duplicate within one version: keep it, so N1 reports it
            continue;
          }
          seenInVersion.add(p);
          if (seenAcrossVersions.has(p)) continue; // shared across sibling versions: count once
          seenAcrossVersions.add(p);
          sink.push(p);
        }
      }
    }
    for (const key of NAV_CONTAINER_KEYS) {
      const children = obj[key];
      if (Array.isArray(children)) children.forEach((child) => walk(child, sink));
    }
  };
  walk(navigation, pages);
  return pages;
}

// ─── Filesystem walk ──────────────────────────────────────────────────────────

/**
 * All routable page paths on disk: every .mdx under the docs root, excluding
 * NON_PAGE_DIRS, as extension-less paths relative to the docs root
 * (e.g. "chain-operators/guides/configuration/batcher").
 */
export function collectDiskPages(docsRoot: string): Set<string> {
  const pages = new Set<string>();
  const walk = (dir: string): void => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = resolveInside(docsRoot, dir, entry.name);
      if (abs === null) continue; // never touch a path outside the docs root
      if (entry.isDirectory()) {
        if (dir === docsRoot && NON_PAGE_DIRS.has(entry.name)) continue;
        if (entry.name === "node_modules" || entry.name === ".git") continue;
        walk(abs);
      } else if (entry.isFile() && entry.name.endsWith(".mdx")) {
        pages.add(path.relative(docsRoot, abs).slice(0, -".mdx".length));
      }
    }
  };
  walk(docsRoot);
  return pages;
}

/**
 * Every file under the docs root (relative paths), excluding node_modules and
 * .git. Used to resolve internal links to static assets (e.g. /public/img/...).
 */
export function collectAllFiles(docsRoot: string): Set<string> {
  const files = new Set<string>();
  const walk = (dir: string): void => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = resolveInside(docsRoot, dir, entry.name);
      if (abs === null) continue; // never touch a path outside the docs root
      if (entry.isDirectory()) {
        if (entry.name === "node_modules" || entry.name === ".git") continue;
        walk(abs);
      } else if (entry.isFile()) {
        files.add(path.relative(docsRoot, abs));
      }
    }
  };
  walk(docsRoot);
  return files;
}

// ─── Path helpers ─────────────────────────────────────────────────────────────

/** Normalize a docs URL path: strip leading/trailing slashes, anchors, queries. */
export function normalizePath(p: string): string {
  return p.split("#")[0].split("?")[0].replace(/^\/+/, "").replace(/\/+$/, "");
}

/**
 * True if a normalized URL path resolves to a page on disk: either
 * `<path>.mdx` or the directory index `<path>/index.mdx` (Mintlify serves
 * both; nav entries and redirect targets use both forms today).
 */
export function resolvesToPage(p: string, diskPages: Set<string>): boolean {
  if (p === "") return diskPages.has("index");
  return diskPages.has(p) || diskPages.has(`${p}/index`);
}

// ─── Reporting ────────────────────────────────────────────────────────────────

export function report(errors: string[], okMessage: string): never {
  if (errors.length > 0) {
    for (const e of errors) console.error(`error: ${e}`);
    console.error(`\n${errors.length} error(s).`);
    process.exit(1);
  }
  console.log(okMessage);
  process.exit(0);
}
