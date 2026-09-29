#!/usr/bin/env bun
// Tests for gen-cli, run with `bun scripts/gen-cli/test.ts` (or `pnpm test:gen-cli`).
// Deterministic: they use the recorded help trees under fixtures/ and a
// temporary docs root, and touch nothing in the repository.

import { execFileSync } from "child_process";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { findComponent } from "./components";
import { emitPages, hoistedSections, slugify, stripHoistedFromHelp } from "./emit";
import { compareLines, spliceNav, versionGroups } from "./nav";
import { parseClapHelp } from "./parse-clap";
import { parseUrfaveHelp } from "./parse-urfave";
import { FixtureSource } from "./source";
import { walkCommands } from "./walk";

import { fileURLToPath } from "url";
const HERE = path.dirname(fileURLToPath(import.meta.url));
// Under bun, process.execPath runs .ts directly; under node (pnpm test:gen-cli
// via tsx) use the tsx binary so extensionless imports resolve.
const RUNNER: string[] = process.versions.bun
  ? [process.execPath]
  : [path.join(HERE, "..", "..", "node_modules", ".bin", "tsx")];
const FIXTURES = path.join(HERE, "fixtures");
let failures = 0;

function check(name: string, cond: boolean, detail = ""): void {
  if (cond) console.log(`ok   ${name}`);
  else {
    failures++;
    console.error(`FAIL ${name}${detail ? `: ${detail}` : ""}`);
  }
}

// ─── clap parser ──────────────────────────────────────────────────────────────
{
  const fx = JSON.parse(fs.readFileSync(path.join(FIXTURES, "op-reth.json"), "utf8"));
  const mdbx = parseClapHelp(["db", "checksum", "mdbx"], fx.help["db checksum mdbx"]);
  check("clap: description", mdbx.description === "Calculates the checksum of a database table", mdbx.description);
  check("clap: usage", mdbx.usage === "op-reth db checksum mdbx [OPTIONS] <TABLE>", mdbx.usage);
  check("clap: positional argument with description", mdbx.args.length === 1 && mdbx.args[0].name === "<TABLE>" && mdbx.args[0].description === "The table name", JSON.stringify(mdbx.args));
  const options = mdbx.sections.find((s) => s.title === "Options");
  check("clap: Options section flags", options?.flags.map((f) => f.long).join(",") === "--start-key,--end-key,--limit", JSON.stringify(options?.flags.map((f) => f.long)));
  check("clap: --help is dropped", !mdbx.sections.some((s) => s.flags.some((f) => f.long === "--help")));
  const logging = mdbx.sections.find((s) => s.title === "Logging");
  const fmt = logging?.flags.find((f) => f.long === "--log.stdout.format");
  check("clap: default value", fmt?.defaultValue === "terminal", String(fmt?.defaultValue));
  check("clap: possible values", (fmt?.possibleValues ?? []).join(",") === "json,log-fmt,terminal", JSON.stringify(fmt?.possibleValues));
  check("clap: summary is first paragraph", fmt?.summary === "The format to use for logs written to stdout", String(fmt?.summary));
  const root = parseClapHelp([], fx.help[""]);
  check("clap: root subcommands", root.subcommands.map((s) => s.name).join(",") === "node,db", JSON.stringify(root.subcommands.map((s) => s.name)));
  const node = parseClapHelp(["node"], fx.help["node"]);
  const chain = node.sections.flatMap((s) => s.flags).find((f) => f.long === "--chain");
  check("clap: env var", node.sections.flatMap((s) => s.flags).some((f) => f.env.length > 0), "expected at least one [env: …]");
  check("clap: short flag", node.sections.flatMap((s) => s.flags).some((f) => f.short === "-d" && f.long === "--disable-discovery"));
  check("clap: --chain default", chain?.defaultValue === "optimism", String(chain?.defaultValue));
  const nodeFlags = node.sections.flatMap((s) => s.flags);
  const otlp = nodeFlags.find((f) => f.long === "--logs-otlp");
  check("clap: optional-value flag parsed", otlp?.optionalValue === true && otlp?.value === "<URL>", JSON.stringify(otlp && { v: otlp.value, o: otlp.optionalValue }));
  check("clap: optional-value flag keeps its own env", (otlp?.env ?? []).includes("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"), JSON.stringify(otlp?.env));
  const color = nodeFlags.find((f) => f.long === "--color");
  check("clap: previous flag does not absorb the next flag's body", !(color?.description ?? "").includes("OTLP") && (color?.env ?? []).length === 0, JSON.stringify(color?.env));
  const verbosity = nodeFlags.find((f) => f.long === "--verbosity");
  check("clap: repeatable flag name excludes the ellipsis", verbosity?.repeatable === true && verbosity?.short === "-v", JSON.stringify(verbosity && { r: verbosity.repeatable, s: verbosity.short }));
  const synthetic = parseClapHelp(["x"], [
    "Usage: t x [OPTIONS]",
    "",
    "Options:",
    "      --pair <A> <B>",
    "          Two values",
    "",
    "      --many <M>...",
    "          Repeats",
    "",
    "          [aliases: old-many, --older]",
    "",
    "  -h, --help",
    "          Print help",
    "",
  ].join("\n"));
  const sx = synthetic.sections[0].flags;
  check("clap: multi-value placeholder", sx[0].value === "<A> <B>", String(sx[0].value));
  check("clap: repeatable value + aliases", sx[1].repeatable && sx[1].aliases.join(",") === "--old-many,--older", JSON.stringify(sx[1]));
  let threw = false;
  try { parseClapHelp(["y"], "Usage: t y\n\nOptions:\n  --weird{shape}\n          ?\n"); } catch { threw = true; }
  check("clap: unknown flag shape is an error, not swallowed", threw);
}

// ─── urfave parser ────────────────────────────────────────────────────────────
{
  const fx = JSON.parse(fs.readFileSync(path.join(FIXTURES, "op-node.json"), "utf8"));
  const root = parseUrfaveHelp([], fx.help[""]);
  check("urfave: description from NAME", root.description === "Optimism Rollup Node", root.description);
  check("urfave: subcommands", root.subcommands.map((s) => s.name).join(",") === "p2p,genesis,doc,networks", JSON.stringify(root.subcommands.map((s) => s.name)));
  const all = root.sections.flatMap((s) => s.flags);
  const l1 = all.find((f) => f.long === "--l1");
  check("urfave: flag with default and env", l1?.defaultValue === '"http://127.0.0.1:8545"' && l1?.env[0] === "OP_NODE_L1_ETH_RPC", JSON.stringify(l1));
  const header = all.find((f) => f.long === "--signer.header");
  check("urfave: multi-line usage joined", (header?.summary ?? "").startsWith("Headers to pass to the remote signer") && (header?.summary ?? "").endsWith("one key value pair per flag."), String(header?.summary));
  check("urfave: categories become sections", root.sections.some((s) => s.title === "Rollup") && root.sections.some((s) => s.title === "L1 RPC"), JSON.stringify(root.sections.map((s) => s.title)));
  check("urfave: uncategorized flags keep the options title", root.sections[0].title === "Global options" && root.sections[0].flags.some((f) => f.long === "--signer.tls.enabled"), JSON.stringify(root.sections[0]));
  check("urfave: --help dropped", !all.some((f) => f.long === "--help"));
  const p2p = parseUrfaveHelp(["p2p"], fx.help["p2p"]);
  check("urfave: subcommand with only --help has no sections", p2p.sections.length === 0 && p2p.subcommands.length === 3, JSON.stringify({ s: p2p.sections.length, c: p2p.subcommands.length }));
  const l2 = parseUrfaveHelp(["genesis", "l2"], fx.help["genesis l2"]);
  check("urfave: bare OPTIONS header keeps its capital", l2.sections[0]?.title === "Options", String(l2.sections[0]?.title));
  const synthetic = parseUrfaveHelp(["z"], [
    "NAME:",
    "   t z - Zed",
    "",
    "OPTIONS:",
    "   ",
    "    --a value, --a-alias value          (default: f(x))                    ($T_A, $T_A2)",
    "          Alpha",
    "   ",
    "    --b                                 (default: (1))",
    "          Beta",
    "",
  ].join("\n"));
  const [a, b] = synthetic.sections[0].flags;
  check("urfave: default with parentheses", a.defaultValue === "f(x)" && b.defaultValue === "(1)", JSON.stringify([a.defaultValue, b.defaultValue]));
  check("urfave: second long name kept as alias, both env vars kept", a.aliases.join(",") === "--a-alias" && a.env.join(",") === "T_A,T_A2", JSON.stringify(a));
  const rollup = parseUrfaveHelp([], fx.help[""]).sections.find((s) => s.title === "Rollup");
  check("urfave: category keeps its raw header for stripping", rollup?.rawHeader === "1. ROLLUP", String(rollup?.rawHeader));
}

// ─── walk + hoist + emit ─────────────────────────────────────────────────────
{
  const component = findComponent("op-reth");
  const tree = walkCommands(component, new FixtureSource(path.join(FIXTURES, "op-reth.json")));
  check("walk: full pruned tree", [...(function* w(t: typeof tree): Generator<string> { yield t.doc.path.join(" "); for (const c of t.children) yield* w(c); })(tree)].length === 7);
  const hoisted = hoistedSections(tree);
  const titles = [...hoisted.values()].map((s) => s.title).sort();
  check("hoist: sections repeated on every command", titles.join(",") === "Display,Logging,Tracing", titles.join(","));
  check("hoist: Datadir variants stay inline", !titles.includes("Datadir"));
  const stripped = stripHoistedFromHelp(tree.children[1].doc.rawHelp, hoisted.values());
  check("hoist: verbatim help loses hoisted sections", !/^Logging:$/m.test(stripped.text) && /^Options:$/m.test(stripped.text) && stripped.removed.includes("Logging"));
  // urfave stripping matches the raw uppercase header and category lines.
  const urfaveStrip = stripHoistedFromHelp("NAME:\n   t\n\nGLOBAL OPTIONS:\n   \n    --x\n          X\n\n   1. ROLLUP\n\n    --y\n          Y\n", [
    { title: "Rollup", rawHeader: "1. ROLLUP", flags: [] },
  ]);
  check("hoist: urfave category block is stripped by its raw header", !urfaveStrip.text.includes("--y") && urfaveStrip.text.includes("--x") && urfaveStrip.removed.join() === "Rollup", urfaveStrip.text);
  check("canonicalize: chain list replaced in verbatim help", tree.children[0].doc.rawHelp.includes("<every chain bundled from the superchain-registry at this release>") && !tree.children[0].doc.rawHelp.includes("optimism, op-mainnet"));

  const { pages, pageMap } = emitPages(tree, {
    component,
    line: "v2.4",
    ref: "op-reth/v2.4.1",
    refUrl: "https://example.invalid/op-reth/v2.4.1",
    artifact: "fixture op-reth.json",
    binaryVersion: null,
  });
  const names = pages.map((p) => p.relPath).sort();
  check("emit: one page per top-level command plus index and global options", names.join(",") === "db.mdx,global-options.mdx,index.mdx,node.mdx", names.join(","));
  const db = pages.find((p) => p.relPath === "db.mdx")!.content;
  check("emit: nested commands are headings", /^## checksum$/m.test(db) && /^### checksum mdbx$/m.test(db) && /^### checksum static-file$/m.test(db));
  check("emit: argument table filled", db.includes("| `<TABLE>` | The table name |"));
  check("emit: generated header first after frontmatter", /^---\n[\s\S]*?\n---\n\n\{\/\*\n  GENERATED FILE — DO NOT EDIT\./.test(db));
  check("emit: no develop noindex on a release line", !db.includes("noindex"));
  check("emit: table cells escape pipes and braces", !/\| [^|\n]*[^\\]\{[^|\n]*\|/.test(db));
  const mdbx = pageMap.find((e) => e.command === "db checksum mdbx");
  check("page map: nested command anchors onto the top-level page", mdbx?.url === "/reference/op-reth/v2.4/db" && mdbx?.anchor === slugify("checksum mdbx"), JSON.stringify(mdbx));
  check("page map: root and top-level entries", pageMap.some((e) => e.command === "" && e.url === "/reference/op-reth/v2.4/index") && pageMap.some((e) => e.command === "node" && e.anchor === null));
  check("emit: pre-escaped help text is not double-escaped", !pages.some((p) => p.content.includes("\\\\<")));
  check("emit: optional-value flag rendered", pages.some((p) => p.content.includes("`--logs-otlp` `[=<URL>]`")));
  check("slugify: matches github-slugger (drops dots, keeps underscores)", slugify("db.v2 set_key") === "dbv2-set_key", slugify("db.v2 set_key"));

  // Determinism: a second emit is byte-identical.
  const again = emitPages(walkCommands(component, new FixtureSource(path.join(FIXTURES, "op-reth.json"))), {
    component, line: "v2.4", ref: "op-reth/v2.4.1", refUrl: "https://example.invalid/op-reth/v2.4.1", artifact: "fixture op-reth.json", binaryVersion: null,
  });
  check("emit: deterministic", JSON.stringify(again.pages) === JSON.stringify(pages));

  // Develop line.
  const nodeTree = walkCommands(findComponent("op-node"), new FixtureSource(path.join(FIXTURES, "op-node.json")));
  check("canonicalize: wrapped Available networks list fully replaced", !nodeTree.doc.rawHelp.includes("op-sepolia") && nodeTree.doc.rawHelp.includes("Available networks: <every chain"), nodeTree.doc.rawHelp.split("Available networks")[1]?.slice(0, 120));
  const dev = emitPages(tree, { component, line: "develop", ref: "0123456789abcdef", refUrl: "https://example.invalid/c", artifact: "binary op-reth", binaryVersion: "op-reth 2.5.0-dev" });
  check("emit: develop pages are noindex and say Unreleased", dev.pages.every((p) => p.content.includes("noindex: true")) && dev.pages[0].content.includes("**Unreleased:**"));

  // Nav.
  const docsJson = { navigation: { tabs: [{ tab: "Protocol", groups: [] }, { tab: "OP Mainnet", groups: [] }] } } as any;
  spliceNav(docsJson, component, "v2.4", versionGroups(pages));
  spliceNav(docsJson, component, "v2.3", versionGroups(pages));
  spliceNav(docsJson, component, "develop", versionGroups(dev.pages));
  const tab = docsJson.navigation.tabs[1];
  check("nav: Reference tab inserted after Protocol", tab.tab === "Reference" && docsJson.navigation.tabs.length === 3);
  const versions = tab.dropdowns[0].versions.map((v: any) => `${v.version}${v.default ? "*" : ""}${v.tag ? `(${v.tag})` : ""}`);
  check("nav: newest line default and Latest, develop last and Unreleased", versions.join(" ") === "v2.4*(Latest) v2.3 develop(Unreleased)", versions.join(" "));
  check("nav: groups", tab.dropdowns[0].versions[0].groups.map((g: any) => g.group).join(",") === "Commands,Global options");
  check("nav: line ordering", ["v1.9", "develop", "v1.10", "v2.0"].sort(compareLines).join(",") === "v2.0,v1.10,v1.9,develop");
}

// ─── main.ts end to end: write then --check ──────────────────────────────────
{
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "gen-cli-test-"));
  try {
    fs.mkdirSync(path.join(tmp, "scripts", "gen-cli"), { recursive: true });
    fs.writeFileSync(path.join(tmp, "docs.json"), JSON.stringify({ navigation: { tabs: [{ tab: "Protocol", groups: [] }] }, redirects: [] }));
    const run = (args: string[]) =>
      execFileSync(RUNNER[0], [...RUNNER.slice(1), path.join(HERE, "main.ts"), ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    const common = ["--component", "op-reth", "--tag", "op-reth/v2.4.1", "--help-json", path.join(FIXTURES, "op-reth.json"), "--docs-dir", tmp];
    const out1 = run(common);
    check("main: writes pages, nav, manifest", out1.includes("wrote 4 pages") && fs.existsSync(path.join(tmp, "reference", "op-reth", "v2.4", "db.mdx")) && fs.existsSync(path.join(tmp, "scripts", "gen-cli", "manifest.json")), out1);
    const out2 = run([...common, "--check"]);
    check("main: --check passes on its own output", out2.includes("check OK"), out2);
    fs.appendFileSync(path.join(tmp, "reference", "op-reth", "v2.4", "db.mdx"), "\nhand edit\n");
    let failed = false;
    try {
      run([...common, "--check"]);
    } catch (e) {
      failed = String((e as { stderr?: string }).stderr ?? "").includes("changed: db.mdx");
    }
    check("main: --check fails on a hand edit", failed);
    // Restore the page, then drift the nav's version tag: --check must notice.
    run(common);
    const dj = JSON.parse(fs.readFileSync(path.join(tmp, "docs.json"), "utf8"));
    dj.navigation.tabs.find((t: any) => t.tab === "Reference").dropdowns[0].versions[0].tag = "Old";
    fs.writeFileSync(path.join(tmp, "docs.json"), JSON.stringify(dj));
    let navFailed = false;
    try { run([...common, "--check"]); } catch (e) { navFailed = String((e as { stderr?: string }).stderr ?? "").includes("version entry differs"); }
    check("main: --check fails on a nav default/tag drift", navFailed);
    const manifest = JSON.parse(fs.readFileSync(path.join(tmp, "scripts", "gen-cli", "manifest.json"), "utf8"));
    check("main: manifest records the tag", manifest["op-reth"]["v2.4"].tag === "op-reth/v2.4.1" && !("commit" in manifest["op-reth"]["v2.4"]), JSON.stringify(manifest));
    const dev = run(["--component", "op-node", "--develop", "--commit", "0123456789abcdef0123456789abcdef01234567", "--help-json", path.join(FIXTURES, "op-node.json"), "--docs-dir", tmp]);
    const manifest2 = JSON.parse(fs.readFileSync(path.join(tmp, "scripts", "gen-cli", "manifest.json"), "utf8"));
    check("main: develop line records a commit, not a tag", dev.includes("wrote") && manifest2["op-node"]["develop"].commit === "0123456789abcdef0123456789abcdef01234567" && !("tag" in manifest2["op-node"]["develop"]), JSON.stringify(manifest2["op-node"]));
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

console.log(failures === 0 ? "\nall gen-cli tests passed" : `\n${failures} gen-cli test(s) failed`);
process.exit(failures === 0 ? 0 : 1);
