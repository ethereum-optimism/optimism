// MDX emitter for gen-cli. Implements the page tree the content guide's
// "Component reference" section specifies:
//
//   reference/<component>/<version>/index.mdx           the binary: commands + root options
//   reference/<component>/<version>/<command>.mdx       one page per top-level command,
//                                                       nested subcommands as headings
//   reference/<component>/<version>/global-options.mdx  option sections every command repeats
//
// Every page opens with the DO NOT EDIT header and a provenance line. Flags
// render as tables; the verbatim --help follows in an expandable block.

import type { Component } from "./components";
import { type CommandTree, type Flag, type FlagSection, displayCommand, sectionKey, walkTree } from "./model";

export interface Provenance {
  component: Component;
  /** "v1.19" or "develop". */
  line: string;
  /** Release tag, or the commit for the develop line. */
  ref: string;
  refUrl: string;
  artifact: string;
  binaryVersion: string | null;
}

export interface Page {
  /** Path relative to the version directory, e.g. "db.mdx". */
  relPath: string;
  /** URL path relative to the site root, e.g. "reference/op-reth/v2.4/db". */
  url: string;
  content: string;
}

export interface PageMapEntry {
  /** Space-joined subcommand path; "" for the binary itself. */
  command: string;
  url: string;
  anchor: string | null;
}

// ─── Text helpers ─────────────────────────────────────────────────────────────

/**
 * Escape a string for an MDX table cell: one line, no unescaped |, {, }, <.
 * Help text sometimes arrives already backslash-escaped (reth writes
 * `extip:\<IP\>` in its doc strings); those escapes are dropped first so they
 * are not doubled into `\\<`, which MDX reads as a JSX tag.
 */
function cell(s: string): string {
  return s
    .replace(/\\([<>|{}])/g, "$1")
    .split(/\s+/)
    .join(" ")
    .trim()
    .replace(/\|/g, "\\|")
    .replace(/\{/g, "\\{")
    .replace(/\}/g, "\\}")
    .replace(/</g, "\\<");
}

/** Inline code; a value containing a backtick gets a double-backtick fence (CommonMark has no escape inside a span). */
function code(s: string): string {
  const escaped = s.replace(/\|/g, "\\|");
  return escaped.includes("`") ? "`` " + escaped + " ``" : "`" + escaped + "`";
}

function defaultCell(v: string | null): string {
  return v === null || v === "" ? "—" : code(v);
}

function envCell(env: string[]): string {
  return env.length === 0 ? "—" : env.map(code).join(", ");
}

/**
 * Mintlify's heading slug (github-slugger): lowercase, punctuation removed
 * except hyphens and underscores, whitespace to hyphens. A nested command
 * named `db.v2` therefore anchors as `dbv2`, not `db-v2`.
 */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s_-]/gu, "")
    .trim()
    .replace(/\s+/g, "-");
}

function yamlString(s: string): string {
  return JSON.stringify(s);
}

function flagName(f: Flag): string {
  const parts = [f.long];
  if (f.short) parts.push(f.short);
  parts.push(...f.aliases);
  let out = parts.map(code).join(", ");
  // urfave prints a generic "value" after every flag that takes one; only a
  // named placeholder (clap's <PATH>) tells the reader anything.
  if (f.value && f.value !== "value") out += ` ${code(f.optionalValue ? `[=${f.value}]` : f.value)}`;
  if (f.repeatable) out += " (repeatable)";
  return out;
}

function flagDescription(f: Flag): string {
  let d = f.summary || f.description;
  if (f.possibleValues.length > 0) {
    if (d && !/[.!?:]$/.test(d.trim())) d = d.trim() + ".";
    d += ` Possible values: ${f.possibleValues.map(code).join(", ")}.`;
  }
  return cell(d);
}

function renderSection(section: FlagSection, heading: string): string[] {
  const lines: string[] = [`${heading} ${section.title}`, ""];
  lines.push("| Flag | Description | Default | Environment variable |");
  lines.push("| --- | --- | --- | --- |");
  for (const f of section.flags) {
    lines.push(`| ${flagName(f)} | ${flagDescription(f)} | ${defaultCell(f.defaultValue)} | ${envCell(f.env)} |`);
  }
  lines.push("");
  return lines;
}

// ─── Hoisting ─────────────────────────────────────────────────────────────────

/**
 * A flag section is hoisted to the global-options page when its exact content
 * (title and flags) appears identically in at least half of all commands (and
 * at least two), and every command that has a section with that title has
 * this same content. Sections whose title appears with differing contents
 * stay inline, so the global page never has two "Datadir" tables; sections a
 * couple of commands happen to share (op-reth's Networking on node and p2p)
 * stay where they are used.
 */
export function hoistedSections(tree: CommandTree): Map<string, FlagSection> {
  const byTitle = new Map<string, Map<string, { section: FlagSection; count: number }>>();
  let total = 0;
  for (const node of walkTree(tree)) {
    total++;
    for (const s of node.doc.sections) {
      const key = sectionKey(s);
      const variants = byTitle.get(s.title) ?? new Map();
      const v = variants.get(key) ?? { section: s, count: 0 };
      v.count++;
      variants.set(key, v);
      byTitle.set(s.title, variants);
    }
  }
  const hoisted = new Map<string, FlagSection>();
  for (const [, variants] of byTitle) {
    if (variants.size !== 1) continue;
    const [key, v] = [...variants.entries()][0];
    if (v.count >= Math.max(2, Math.ceil(total / 2))) hoisted.set(key, v.section);
  }
  return hoisted;
}

/**
 * Remove hoisted sections from a verbatim --help text, matching each section
 * by the header line it opens with in the raw text (`rawHeader`): a clap or
 * urfave header at column 0 (`Logging:`, `GLOBAL OPTIONS:`), or a urfave
 * category line (`1. ROLLUP`). A block ends at the next header of either
 * kind. The page links to the global-options page in their place. Returns
 * the stripped text and the titles that were actually removed.
 */
export function stripHoistedFromHelp(help: string, hoisted: Iterable<FlagSection>): { text: string; removed: string[] } {
  const byRaw = new Map<string, string>();
  for (const s of hoisted) byRaw.set(s.rawHeader, s.title);
  if (byRaw.size === 0) return { text: help, removed: [] };
  const isHeader = (line: string) => /^[A-Z][A-Za-z0-9 /_-]*:\s*$/.test(line) || /^ {3}\d+\.\s+\S/.test(line);
  const out: string[] = [];
  const removed: string[] = [];
  let skipping = false;
  for (const line of help.split("\n")) {
    if (isHeader(line)) {
      const title = byRaw.get(line.trim());
      skipping = title !== undefined;
      if (title !== undefined && !removed.includes(title)) removed.push(title);
    }
    if (!skipping) out.push(line);
  }
  return { text: out.join("\n").replace(/\n{3,}/g, "\n\n").trim(), removed };
}

// ─── Pages ────────────────────────────────────────────────────────────────────

function header(p: Provenance, extra: string[] = []): string[] {
  const generatedFrom = p.line === "develop" ? `commit ${p.ref}` : `release tag ${p.ref}`;
  return [
    "{/*",
    "  GENERATED FILE — DO NOT EDIT.",
    "",
    `  Generated by docs/public-docs/scripts/gen-cli from the ${p.component.name} ${generatedFrom}`,
    `  (${p.artifact}). Hand edits fail the generator's --check; regenerate instead:`,
    "",
    p.line === "develop"
      ? `    bun docs/public-docs/scripts/gen-cli/main.ts --component ${p.component.name} --develop --commit <sha> --bin <binary>`
      : `    bun docs/public-docs/scripts/gen-cli/main.ts --component ${p.component.name} --tag ${p.ref} --image auto`,
    ...extra.map((l) => `  ${l}`),
    "*/}",
    "",
    p.line === "develop"
      ? `Generated from \`develop\` at [\`${p.ref.slice(0, 12)}\`](${p.refUrl}). **Unreleased:** this documents behavior no published release has yet.`
      : `Generated from [\`${p.ref}\`](${p.refUrl})${p.binaryVersion ? ` (\`${p.binaryVersion}\`)` : ""}.`,
    "",
  ];
}

function frontmatter(p: Provenance, title: string, sidebarTitle: string, description: string): string[] {
  const fm = [
    "---",
    `title: ${yamlString(title)}`,
    `sidebarTitle: ${yamlString(sidebarTitle)}`,
    `description: ${yamlString(description)}`,
    "diataxis: reference",
  ];
  if (p.line === "develop") fm.push("noindex: true");
  fm.push("---", "");
  return fm;
}

function accordion(binary: string, node: CommandTree, hoisted: Map<string, FlagSection>, globalOptionsUrl: string | null): string[] {
  const cmd = displayCommand(binary, node.doc.path);
  const { text, removed } = stripHoistedFromHelp(node.doc.rawHelp, hoisted.values());
  const note =
    globalOptionsUrl && removed.length > 0
      ? [`  The ${removed.join(", ")} ${removed.length === 1 ? "section is" : "sections are"} on the [global options](${globalOptionsUrl}) page.`, ""]
      : [];
  return [
    `<Accordion title="${cmd} --help">`,
    ...note,
    "  ```txt",
    ...text.split("\n").map((l) => (l ? `  ${l}` : "")),
    "  ```",
    "</Accordion>",
    "",
  ];
}

function renderCommandBlock(
  binary: string,
  node: CommandTree,
  depth: number,
  hoisted: Map<string, FlagSection>,
  globalOptionsUrl: string | null,
  linkFor: (path: string[]) => string,
): string[] {
  const cmd = displayCommand(binary, node.doc.path);
  const h = "#".repeat(Math.min(depth, 6));
  const lines: string[] = [];
  if (node.doc.description) lines.push(node.doc.description, "");
  if (node.doc.usage) lines.push("```bash", ...node.doc.usage.split("\n"), "```", "");

  if (node.doc.args.length > 0) {
    lines.push(`${h} Arguments`, "", "| Argument | Description |", "| --- | --- |");
    for (const a of node.doc.args) lines.push(`| ${code(a.name)} | ${cell(a.description)} |`);
    lines.push("");
  }

  if (node.children.length > 0) {
    lines.push(`${h} Subcommands`, "", "| Subcommand | Description |", "| --- | --- |");
    for (const child of node.children) {
      const summary = child.doc.description || `${child.children.length} subcommands`;
      lines.push(`| [${code(displayCommand(binary, child.doc.path))}](${linkFor(child.doc.path)}) | ${cell(summary)} |`);
    }
    lines.push("");
  }

  let usesGlobal = false;
  for (const s of node.doc.sections) {
    if (hoisted.has(sectionKey(s))) {
      usesGlobal = true;
      continue;
    }
    lines.push(...renderSection(s, h));
  }
  if (usesGlobal && globalOptionsUrl) {
    lines.push(`${cmd} also accepts the [global options](${globalOptionsUrl}).`, "");
  }
  lines.push(...accordion(binary, node, hoisted, globalOptionsUrl));
  return lines;
}

export function emitPages(tree: CommandTree, p: Provenance): { pages: Page[]; pageMap: PageMapEntry[] } {
  const binary = p.component.binary ?? p.component.name;
  const base = `reference/${p.component.name}/${p.line}`;
  const hoisted = hoistedSections(tree);
  const globalOptionsUrl = hoisted.size > 0 ? `/${base}/global-options` : null;

  // URL and anchor for every command: top-level commands get a page, deeper
  // commands get a heading on their top-level ancestor's page.
  // Mintlify serves a folder's index.mdx at /folder/index, and this site's nav
  // and links spell index pages out that way.
  const rootUrl = `/${base}/index`;
  const pageMap: PageMapEntry[] = [{ command: "", url: rootUrl, anchor: null }];
  const linkFor = (path: string[]): string => {
    if (path.length === 0) return rootUrl;
    const top = path[0];
    if (path.length === 1) return `/${base}/${top}`;
    return `/${base}/${top}#${slugify(path.slice(1).join(" "))}`;
  };
  for (const node of walkTree(tree)) {
    if (node.doc.path.length === 0) continue;
    const url = linkFor(node.doc.path);
    const [u, anchor] = url.split("#");
    pageMap.push({ command: node.doc.path.join(" "), url: u, anchor: anchor ?? null });
  }

  const pages: Page[] = [];

  // Root page.
  {
    const lines: string[] = [
      ...frontmatter(p, binary, binary, tree.doc.description || `${binary} command-line reference`),
      ...header(p),
    ];
    lines.push(...renderCommandBlock(binary, tree, 2, hoisted, globalOptionsUrl, linkFor));
    pages.push({ relPath: "index.mdx", url: `${base}/index`, content: finish(lines) });
  }

  // One page per top-level command, nested commands as headings.
  const RESERVED = new Set(["index", "global-options"]);
  for (const top of tree.children) {
    if (RESERVED.has(top.doc.path[0])) {
      throw new Error(`top-level command "${top.doc.path[0]}" collides with a generated page name; the emitter needs a rule for it`);
    }
    const cmd = displayCommand(binary, top.doc.path);
    const lines: string[] = [
      ...frontmatter(p, cmd, top.doc.path[0], top.doc.description || `${cmd} command reference`),
      ...header(p),
    ];
    lines.push(...renderCommandBlock(binary, top, 2, hoisted, globalOptionsUrl, linkFor));
    const emitNested = (node: CommandTree, depth: number) => {
      for (const child of node.children) {
        const rel = child.doc.path.slice(1).join(" ");
        lines.push(`${"#".repeat(Math.min(depth, 6))} ${rel}`, "");
        lines.push(...renderCommandBlock(binary, child, Math.min(depth + 1, 6), hoisted, globalOptionsUrl, linkFor));
        emitNested(child, depth + 1);
      }
    };
    emitNested(top, 2);
    pages.push({ relPath: `${top.doc.path[0]}.mdx`, url: `${base}/${top.doc.path[0]}`, content: finish(lines) });
  }

  // Global options page.
  if (hoisted.size > 0) {
    const lines: string[] = [
      ...frontmatter(p, `${binary} global options`, "Global options", `Options every ${binary} command accepts.`),
      ...header(p),
      `These option sections are printed identically by every \`${binary}\` command that accepts them; the command pages link here instead of repeating them.`,
      "",
    ];
    for (const s of hoisted.values()) lines.push(...renderSection(s, "##"));
    pages.push({ relPath: "global-options.mdx", url: `${base}/global-options`, content: finish(lines) });
  }

  return { pages, pageMap };
}

function finish(lines: string[]): string {
  return lines
    .join("\n")
    .split("\n")
    .map((l) => l.replace(/\s+$/, ""))
    .join("\n")
    .replace(/\n{3,}/g, "\n\n")
    .trimEnd() + "\n";
}
