#!/usr/bin/env node
/**
 * lint-link-policy.mjs — deterministic cross-repo link-policy linter for docs.optimism.io.
 *
 * Enforces the policy defined at docs/public-docs/op-stack/contribute/link-policy.mdx:
 *
 *   retired-spec-path      specs.optimism.io link rides a retired path that only resolves
 *                          through the specs repo's hand-maintained book.toml redirect table
 *   anchor-utm-order       query parameters (e.g. UTM) appended AFTER the #fragment — the
 *                          anchor never resolves in the browser
 *   spec-blob-link         GitHub blob/raw link into the specs repo's specs/**.md sources —
 *                          a rendered specs.optimism.io page exists for every specs/**.md file
 *   unbadged-pinned-link   commit- or tag-pinned ethereum-optimism source link without an
 *                          "as of `<tag>`" badge on (or adjacent to) the same line
 *   dead-internal-link     root-relative internal link whose target page/asset does not exist
 *                          (docs.json redirects are honored)
 *   malformed-link         internal link target that is neither root-relative, a fragment,
 *                          nor an absolute URL
 *   unresolved-spec-path   specs.optimism.io link whose source file does not exist in the
 *                          specs repo (requires --specs-src)
 *   unresolved-spec-anchor deep anchor into specs.optimism.io that does not match any heading
 *                          slug in the corresponding specs source file (requires --specs-src)
 *
 * Usage:
 *   node scripts/lint-link-policy.mjs [options]
 *     --docs <dir>          docs root (default: the directory containing scripts/)
 *     --specs-src <dir>     checkout of ethereum-optimism/specs for path/anchor resolution;
 *                           when omitted, spec path/anchor resolution checks are SKIPPED
 *     --baseline <file>     baseline JSON; only findings NOT in the baseline fail the run
 *     --update-baseline     rewrite the baseline file from current findings
 *     --report <file>       write a markdown report of all findings
 *     --self-test           run against the synthetic fixtures and verify each violation
 *                           class fires (and that the passing fixtures are clean)
 *
 * Exit codes: 0 clean (or all findings baselined), 1 new findings, 2 usage/internal error.
 *
 * This script is intentionally dependency-free and offline-deterministic: specs resolution
 * runs against a local checkout, never against the network.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));

// ---------------------------------------------------------------------------
// Vendored ground truth: retired rendered-site paths.
// Source: [output.html.redirect] table in ethereum-optimism/specs book.toml
// (the hand-maintained redirect table that keeps these paths barely alive).
// Re-vendor when the specs repo adds redirect rules.
// ---------------------------------------------------------------------------
const RETIRED_SPEC_PATHS = {
  "/experimental/fault-proof/index.html": "/fault-proof/index.html",
  "/experimental/fault-proof/cannon-fault-proof-vm.html": "/fault-proof/cannon-fault-proof-vm.html",
  "/experimental/fault-proof/stage-one/index.html": "/fault-proof/stage-one/index.html",
  "/experimental/fault-proof/stage-one/bond-incentives.html": "/fault-proof/stage-one/bond-incentives.html",
  "/experimental/fault-proof/stage-one/bridge-integration.html": "/fault-proof/stage-one/bridge-integration.html",
  "/experimental/fault-proof/stage-one/dispute-game-interface.html": "/fault-proof/stage-one/dispute-game-interface.html",
  "/experimental/fault-proof/stage-one/fault-dispute-game.html": "/fault-proof/stage-one/fault-dispute-game.html",
  "/experimental/fault-proof/stage-one/honest-challenger-fdg.html": "/fault-proof/stage-one/honest-challenger-fdg.html",
  "/experimental/plasma.html": "/experimental/alt-da.html",
  "/fjord/overview.html": "/protocol/fjord/overview.html",
  "/fjord/derivation.html": "/protocol/fjord/derivation.html",
  "/fjord/exec-engine.html": "/protocol/fjord/exec-engine.html",
  "/fjord/predeploys.html": "/protocol/fjord/predeploys.html",
};

const SPECS_HOST = "specs.optimism.io";
const GH_ORG = "ethereum-optimism";
// Branch refs that are "floating" (allowed without a badge). Everything else that
// looks like a commit sha or a release tag is a pin and needs the badge.
const FLOATING_REFS = new Set(["develop", "main", "master", "HEAD"]);

const URL_RE = /https?:\/\/[^\s)\]"'<>`]+/g;
const MD_LINK_RE = /\]\(([^()\s]*(?:\([^()]*\)[^()\s]*)*)\)/g;
const HREF_RE = /(?:href|src)\s*=\s*"([^"]+)"/g;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function listMarkdownFiles(root, { mdxOnly = false } = {}) {
  const out = [];
  const skip = new Set(["node_modules", ".git", "scripts"]);
  (function walk(dir) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (entry.isDirectory()) {
        if (!skip.has(entry.name)) walk(path.join(dir, entry.name));
      } else if (entry.name.endsWith(".mdx") || (!mdxOnly && entry.name.endsWith(".md"))) {
        out.push(path.join(dir, entry.name));
      }
    }
  })(root);
  return out;
}

/**
 * Returns per-line "prose" text with fenced code blocks and inline code spans
 * blanked out, so illustrative URLs inside code are not linted.
 */
function proseLines(content) {
  const lines = content.split("\n");
  const out = [];
  let inFence = false;
  for (const line of lines) {
    const fenceMatch = line.match(/^\s*(```|~~~)/);
    if (fenceMatch) {
      inFence = !inFence;
      out.push("");
      continue;
    }
    if (inFence) {
      out.push("");
      continue;
    }
    // Blank inline code spans.
    out.push(line.replace(/`[^`]*`/g, (m) => " ".repeat(m.length)));
  }
  return out;
}

/** mdBook (pulldown-cmark) heading id: lowercase, drop non [a-z0-9 _-], spaces -> '-'. */
function mdbookSlug(headingText) {
  let text = headingText.trim();
  // Custom id attribute: "## Heading {#custom-id}"
  const custom = text.match(/\{#([^}]+)\}\s*$/);
  if (custom) return custom[1];
  text = text
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "") // images
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1") // links -> text
    .replace(/[`*_~]/g, ""); // emphasis / code markers
  let slug = "";
  for (const ch of text.toLowerCase()) {
    if (/[a-z0-9_-]/.test(ch)) slug += ch;
    else if (/\s/.test(ch)) slug += "-";
    // everything else is dropped
  }
  return slug;
}

/** Heading anchor set for one specs markdown file, with mdBook duplicate suffixes. */
function specAnchors(mdFile) {
  const anchors = new Set();
  const counts = new Map();
  let inFence = false;
  for (const line of fs.readFileSync(mdFile, "utf8").split("\n")) {
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const m = line.match(/^\s{0,3}(#{1,6})\s+(.*)$/);
    if (!m) continue;
    const base = mdbookSlug(m[2]);
    const n = counts.get(base) ?? 0;
    counts.set(base, n + 1);
    anchors.add(n === 0 ? base : `${base}-${n}`);
  }
  return anchors;
}

/** Map a rendered specs.optimism.io pathname to its markdown source inside --specs-src. */
function specSourcePath(specsSrc, pathname) {
  let p = pathname.replace(/^\/+/, "");
  if (p === "" || p === "index.html") p = "root.md";
  else if (p.endsWith(".html")) p = p.slice(0, -5) + ".md";
  else if (p.endsWith("/")) p += "index.md";
  else if (!p.endsWith(".md")) p += ".md";
  return path.join(specsSrc, "specs", p);
}

function loadRedirects(docsRoot) {
  const docsJson = path.join(docsRoot, "docs.json");
  if (!fs.existsSync(docsJson)) return [];
  try {
    const parsed = JSON.parse(fs.readFileSync(docsJson, "utf8"));
    return (parsed.redirects ?? []).map((r) => r.source).filter(Boolean);
  } catch {
    return [];
  }
}

function redirectCovers(sources, target) {
  for (const src of sources) {
    if (src === target) return true;
    // Wildcard/param patterns: "/foo/:slug*" or "/foo/*"
    const cut = src.search(/[:*]/);
    if (cut !== -1 && target.startsWith(src.slice(0, cut))) return true;
  }
  return false;
}

function parseRef(segments) {
  // Given path segments after /blob|tree/, return {ref, kind} where kind is
  // "floating" | "commit" | "tag". Handles slash-containing tags (op-node/v1.2.3).
  const [a, b] = segments;
  if (!a) return null;
  if (FLOATING_REFS.has(a)) return { ref: a, kind: "floating" };
  if (/^[0-9a-f]{7,40}$/.test(a)) return { ref: a, kind: "commit" };
  if (/^v\d/.test(a)) return { ref: a, kind: "tag" };
  if (b && /^v\d/.test(b) && /^[a-z0-9][a-z0-9._-]*$/i.test(a)) return { ref: `${a}/${b}`, kind: "tag" };
  return { ref: a, kind: "floating" }; // unknown branch name — treat as floating
}

// ---------------------------------------------------------------------------
// The lint engine
// ---------------------------------------------------------------------------

function lintTree(docsRoot, { specsSrc = null } = {}) {
  const findings = [];
  const redirects = loadRedirects(docsRoot);
  const notes = [];
  if (!specsSrc) {
    notes.push("specs path/anchor resolution SKIPPED (no --specs-src given)");
  }

  const add = (rule, file, line, url, detail) =>
    findings.push({ rule, file: path.relative(docsRoot, file).split(path.sep).join("/"), line, url, detail });

  for (const file of listMarkdownFiles(docsRoot)) {
    const raw = fs.readFileSync(file, "utf8");
    const rawLines = raw.split("\n");
    const prose = proseLines(raw);
    const isMdx = file.endsWith(".mdx");

    prose.forEach((lineText, i) => {
      const lineNo = i + 1;

      // ---- external URL checks --------------------------------------------
      for (const rawUrl of lineText.match(URL_RE) ?? []) {
        const urlStr = rawUrl.replace(/[.,;:!]+$/, "");
        let u;
        try {
          u = new URL(urlStr);
        } catch {
          continue;
        }

        // anchor-utm-order: query glued after the fragment — anchor never resolves.
        const brokenAnchorOrder = u.hash.includes("?");
        if (brokenAnchorOrder) {
          add("anchor-utm-order", file, lineNo, urlStr, "query parameters must come before the #fragment");
        }

        if (u.hostname === SPECS_HOST) {
          const retired = RETIRED_SPEC_PATHS[u.pathname];
          if (retired) {
            add("retired-spec-path", file, lineNo, urlStr, `retired path; current path is ${retired}`);
          } else if (specsSrc) {
            const src = specSourcePath(specsSrc, u.pathname);
            if (!fs.existsSync(src)) {
              add("unresolved-spec-path", file, lineNo, urlStr, `no source file ${path.relative(specsSrc, src)} in the specs repo`);
            } else if (u.hash && u.hash !== "#" && !brokenAnchorOrder) {
              const anchor = decodeURIComponent(u.hash.slice(1)).toLowerCase();
              if (!specAnchors(src).has(anchor)) {
                add("unresolved-spec-anchor", file, lineNo, urlStr, `no heading with slug "${anchor}" in ${path.relative(specsSrc, src)}`);
              }
            }
          }
        }

        if (u.hostname === "github.com") {
          const seg = u.pathname.replace(/^\/+/, "").split("/");
          const [org, repo, view, ...rest] = seg;
          if (org === GH_ORG && (view === "blob" || view === "raw" || view === "tree")) {
            // spec-blob-link: a rendered specs.optimism.io page exists for every specs/**.md.
            if (repo === "specs" && (view === "blob" || view === "raw")) {
              const refInfo = parseRef(rest);
              const tail = rest.slice(refInfo && refInfo.ref.includes("/") ? 2 : 1).join("/");
              if (tail.startsWith("specs/") && tail.endsWith(".md")) {
                add("spec-blob-link", file, lineNo, urlStr, "link the rendered page on specs.optimism.io instead of the GitHub blob");
              }
            }
            // unbadged-pinned-link: sha/tag pins need an "as of `<tag>`" badge nearby.
            const refInfo = parseRef(rest);
            if (refInfo && refInfo.kind !== "floating") {
              const window = [rawLines[i - 1] ?? "", rawLines[i] ?? "", rawLines[i + 1] ?? ""].join("\n");
              if (!/as of/i.test(window)) {
                add(
                  "unbadged-pinned-link",
                  file,
                  lineNo,
                  urlStr,
                  `pinned to ${refInfo.kind} ${refInfo.ref} without an "as of \`<tag>\`" badge`
                );
              }
            }
          }
        }
      }

      // ---- internal link checks (site pages only) ---------------------------
      if (!isMdx) return;
      const targets = [];
      for (const m of lineText.matchAll(MD_LINK_RE)) targets.push(m[1]);
      for (const m of lineText.matchAll(HREF_RE)) targets.push(m[1]);
      for (const target of targets) {
        if (target === "" || target.startsWith("#") || target.startsWith("{")) continue;
        if (/^[a-z][a-z0-9+.-]*:/i.test(target)) continue; // absolute URL / mailto / etc.
        if (!target.startsWith("/")) {
          // Relative target: fine if it resolves against the containing file's
          // directory; otherwise it is malformed (e.g. a bare address pasted
          // where a URL belongs).
          const relClean = decodeURIComponent(target.split("#")[0].split("?")[0]);
          const resolved = path.resolve(path.dirname(file), relClean);
          const relCandidates = [resolved, `${resolved}.mdx`, `${resolved}.md`];
          if (relClean === "" || !relCandidates.some((c) => fs.existsSync(c))) {
            add("malformed-link", file, lineNo, target, "not an absolute URL, and resolves to no page relative to this file");
          }
          continue;
        }
        const clean = decodeURIComponent(target.split("#")[0].split("?")[0]).replace(/\/+$/, "");
        if (clean === "") continue;
        const rel = clean.replace(/^\/+/, "");
        const candidates = [
          path.join(docsRoot, `${rel}.mdx`),
          path.join(docsRoot, `${rel}.md`),
          // Static assets are referenced with their on-disk path from the docs
          // root, including the public/ prefix (e.g. /public/img/...).
          path.join(docsRoot, rel),
        ];
        if (!candidates.some((c) => fs.existsSync(c)) && !redirectCovers(redirects, clean)) {
          add("dead-internal-link", file, lineNo, target, "no page or asset at this path (and no docs.json redirect)");
        }
      }
    });
  }

  findings.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line || a.rule.localeCompare(b.rule));
  return { findings, notes };
}

// ---------------------------------------------------------------------------
// Baseline
// ---------------------------------------------------------------------------

const baselineKey = (f) => `${f.rule} ${f.file} ${f.url}`;

function splitByBaseline(findings, baselineFile) {
  if (!baselineFile || !fs.existsSync(baselineFile)) {
    return { fresh: findings, baselined: [], stale: [] };
  }
  const baseline = JSON.parse(fs.readFileSync(baselineFile, "utf8"));
  const known = new Set((baseline.findings ?? []).map(baselineKey));
  const fresh = findings.filter((f) => !known.has(baselineKey(f)));
  const baselined = findings.filter((f) => known.has(baselineKey(f)));
  const current = new Set(findings.map(baselineKey));
  const stale = (baseline.findings ?? []).filter((f) => !current.has(baselineKey(f)));
  return { fresh, baselined, stale };
}

// ---------------------------------------------------------------------------
// Self-test on the synthetic fixtures
// ---------------------------------------------------------------------------

function selfTest() {
  // The fixtures live OUTSIDE the Mintlify content root (docs/public-docs/) so
  // their deliberate violations never show up in `mint broken-links` output.
  const fixtures = path.resolve(SCRIPT_DIR, "..", "..", "lint-link-policy-fixtures");
  const failingDir = path.join(fixtures, "failing");
  const passingDir = path.join(fixtures, "passing");
  const specsFixture = path.join(fixtures, "specs-src");
  let ok = true;

  // Every failing fixture must fire the rule named by its filename.
  const { findings: failFindings } = lintTree(failingDir, { specsSrc: specsFixture });
  for (const fixture of fs.readdirSync(failingDir).filter((f) => f.endsWith(".mdx")).sort()) {
    const rule = path.basename(fixture, ".mdx");
    const hits = failFindings.filter((f) => f.file === fixture && f.rule === rule);
    if (hits.length === 0) {
      console.error(`SELF-TEST FAIL: fixture failing/${fixture} did not trigger rule "${rule}"`);
      ok = false;
    } else {
      console.log(`self-test ok: failing/${fixture} triggers ${rule} (${hits.length}x)`);
    }
  }

  // The passing fixtures must be completely clean.
  const { findings: passFindings } = lintTree(passingDir, { specsSrc: specsFixture });
  if (passFindings.length > 0) {
    console.error(`SELF-TEST FAIL: passing fixtures produced ${passFindings.length} finding(s):`);
    for (const f of passFindings) console.error(`  ${f.file}:${f.line} [${f.rule}] ${f.url}`);
    ok = false;
  } else {
    console.log("self-test ok: passing fixtures are clean");
  }

  return ok;
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

function main(argv) {
  const args = { docs: path.resolve(SCRIPT_DIR, ".."), specsSrc: null, baseline: null, update: false, report: null, selfTest: false };
  for (let i = 0; i < argv.length; i++) {
    switch (argv[i]) {
      case "--docs": args.docs = path.resolve(argv[++i]); break;
      case "--specs-src": args.specsSrc = path.resolve(argv[++i]); break;
      case "--baseline": args.baseline = path.resolve(argv[++i]); break;
      case "--update-baseline": args.update = true; break;
      case "--report": args.report = path.resolve(argv[++i]); break;
      case "--self-test": args.selfTest = true; break;
      default:
        console.error(`unknown argument: ${argv[i]}`);
        return 2;
    }
  }

  if (args.selfTest) return selfTest() ? 0 : 1;

  if (!fs.existsSync(args.docs)) {
    console.error(`--docs directory not found: ${args.docs}`);
    return 2;
  }
  if (args.specsSrc && !fs.existsSync(path.join(args.specsSrc, "specs"))) {
    console.error(`--specs-src is not a specs checkout (no specs/ subdirectory): ${args.specsSrc}`);
    return 2;
  }

  const { findings, notes } = lintTree(args.docs, { specsSrc: args.specsSrc });
  for (const n of notes) console.log(`note: ${n}`);

  if (args.update) {
    const baselineFile = args.baseline ?? path.join(SCRIPT_DIR, "lint-link-policy.baseline.json");
    fs.writeFileSync(
      baselineFile,
      JSON.stringify(
        {
          comment:
            "Pre-existing link-policy violations, baselined so CI blocks only NEW violations. " +
            "This file is the worklist for the B2 remediation sweep — shrink it, never grow it. " +
            "Regenerate: node scripts/lint-link-policy.mjs --update-baseline [--specs-src <dir>]",
          findings: findings.map(({ rule, file, url }) => ({ rule, file, url })),
        },
        null,
        2
      ) + "\n"
    );
    console.log(`baseline updated: ${baselineFile} (${findings.length} findings)`);
    return 0;
  }

  const { fresh, baselined, stale } = splitByBaseline(findings, args.baseline);

  if (args.report) {
    const lines = [
      `# link-policy findings`,
      "",
      `Total: ${findings.length} (${fresh.length} new, ${baselined.length} baselined)`,
      "",
      ...findings.map((f) => `- \`${f.file}:${f.line}\` **${f.rule}** — ${f.url} (${f.detail})`),
    ];
    fs.writeFileSync(args.report, lines.join("\n") + "\n");
  }

  const byRule = {};
  for (const f of findings) byRule[f.rule] = (byRule[f.rule] ?? 0) + 1;
  console.log(`scanned ${args.docs}: ${findings.length} finding(s) ${JSON.stringify(byRule)}`);
  if (stale.length > 0) {
    console.log(`note: ${stale.length} baseline entr${stale.length === 1 ? "y is" : "ies are"} stale (fixed or moved) — prune with --update-baseline`);
  }

  if (fresh.length > 0) {
    console.error(`\n${fresh.length} NEW link-policy violation(s) (not in baseline):\n`);
    for (const f of fresh) console.error(`  ${f.file}:${f.line} [${f.rule}] ${f.url}\n      ${f.detail}`);
    console.error(
      "\nFix the links per docs/public-docs/op-stack/contribute/link-policy.mdx." +
        "\nIf a finding is a deliberate exception, discuss it in the PR — do not silently rebaseline."
    );
    return 1;
  }
  console.log(baselined.length > 0 ? `clean: no new violations (${baselined.length} pre-existing baselined)` : "clean: no violations");
  return 0;
}

process.exit(main(process.argv.slice(2)));
