// urfave/cli (Go) --help parser for gen-cli, for the help template the OP
// Stack Go services print (op-service's app template, not urfave's default):
//
//   NAME:
//      op-node - Optimism Rollup Node
//
//   USAGE:
//      op-node [global options] command [command options]
//
//   COMMANDS:
//      p2p
//      help, h   Shows a list of commands or help for one command
//
//   GLOBAL OPTIONS:
//
//       --signer.header value                          ($OP_NODE_SIGNER_HEADER)
//             Headers to pass to the remote signer. Format `key=value`. …
//             using flags one key value pair per flag.
//
//       --signer.tls.enabled       (default: true)     ($OP_NODE_SIGNER_TLS_ENABLED)
//             Enable or disable TLS client authentication for the signer
//
//      1. ROLLUP
//
//       --finality.delay value     (default: 0)        ($OP_NODE_FINALITY_DELAY)
//             Number of L1 blocks to traverse …
//
// Section headers are UPPERCASE, end with ":" at column 0. Inside an options
// section a flag line is indented and starts with "-"; it carries the names
// (with " value" after names that take one), an optional "(default: …)" and
// an optional "($ENV[, $ENV])". The usage follows on lines indented deeper
// than the flag. A numbered line ("1. ROLLUP") starts a category, which
// becomes its own section. Subcommand help uses "OPTIONS:" for the header.

import type { CommandDoc, Flag, FlagSection, SubcommandRef } from "./model";
import { SUBCOMMAND_NAME } from "./parse-clap";

const HEADER_RE = /^([A-Z][A-Z ]*):\s*$/;
const FLAG_LINE_RE = /^( +)(-{1,2}[A-Za-z0-9][^\s,]*(?:\s+value)?(?:,\s*-{1,2}[A-Za-z0-9][^\s,]*(?:\s+value)?)*)(.*)$/;
const CATEGORY_RE = /^ {3}(\d+\.\s+\S.*?)\s*$/;
const COMMAND_LINE_RE = /^ {3}([a-z0-9][a-z0-9._-]*)(?:,\s*[a-z0-9._-]+)*(?:\s{2,}(.*\S))?\s*$/;

function parseTrailer(trailer: string): { defaultValue: string | null; env: string[] } {
  let defaultValue: string | null = null;
  const env: string[] = [];
  // The default may itself contain parentheses; it ends where the env list
  // starts or the line ends.
  const def = /\(default:\s*(.*?)\)(?=\s*(?:\(\$|$))/.exec(trailer);
  if (def) defaultValue = def[1].trim();
  const envM = /\((\$[A-Z0-9_]+(?:\s*,\s*\$[A-Z0-9_]+)*)\)/.exec(trailer);
  if (envM) env.push(...envM[1].split(",").map((e) => e.trim().replace(/^\$/, "")));
  return { defaultValue, env };
}

function parseNames(names: string): Pick<Flag, "long" | "short" | "aliases" | "value"> {
  let long: string | null = null;
  let short: string | null = null;
  const aliases: string[] = [];
  let value: string | null = null;
  for (const part of names.split(",").map((p) => p.trim())) {
    const [name, ...rest] = part.split(/\s+/);
    if (name.startsWith("--")) {
      if (long === null) long = name;
      else aliases.push(name);
    } else if (name.startsWith("-")) {
      if (short === null) short = name;
      else aliases.push(name);
    }
    if (rest.length > 0 && value === null) value = rest.join(" ");
  }
  return { long: long ?? short ?? names, short, aliases, value };
}

export function parseUrfaveHelp(path: string[], help: string): CommandDoc {
  const lines = help.replace(/\r\n?/g, "\n").split("\n");
  let description = "";
  let usage = "";
  const subcommands: SubcommandRef[] = [];
  const sections: FlagSection[] = [];

  let section: string | null = null;
  let inOptions = false;
  let current: FlagSection | null = null;
  let pending: Flag | null = null;
  let pendingIndent = 0;
  let pendingBody: string[] = [];

  const flush = () => {
    if (pending && current) {
      const text = pendingBody.map((l) => l.trim()).join(" ").replace(/\s+/g, " ").trim();
      pending.summary = text;
      pending.description = text;
      current.flags.push(pending);
    }
    pending = null;
    pendingBody = [];
  };

  for (const line of lines) {
    const header = HEADER_RE.exec(line);
    if (header) {
      flush();
      section = header[1];
      inOptions = /OPTIONS$/.test(section);
      current = inOptions ? { title: section, rawHeader: line.trim(), flags: [] } : null;
      if (current) sections.push(current);
      continue;
    }
    switch (section) {
      case "NAME": {
        const m = /^\s+\S.*? - (.*)$/.exec(line);
        if (m && !description) description = m[1].trim();
        break;
      }
      case "USAGE":
        if (line.trim()) usage += (usage ? "\n" : "") + line.trim();
        break;
      case "COMMANDS": {
        if (line.trim() === "") break;
        const m = COMMAND_LINE_RE.exec(line);
        if (!m) {
          if (/^ {5,}\S/.test(line) && subcommands.length > 0) break; // wrapped description
          throw new Error(`cannot parse COMMANDS line in --help: ${JSON.stringify(line)}`);
        }
        if (m[1] === "help") break;
        if (!SUBCOMMAND_NAME.test(m[1])) throw new Error(`unexpected subcommand name in --help: ${JSON.stringify(m[1])}`);
        subcommands.push({ name: m[1], summary: m[2] ?? "" });
        break;
      }
      default: {
        if (!inOptions) break;
        if (line.trim() === "") {
          // Blank lines separate flags; a flag's usage never contains one.
          flush();
          break;
        }
        const cat = CATEGORY_RE.exec(line);
        if (cat && !/^\s*-/.test(cat[1])) {
          flush();
          current = { title: cat[1].replace(/^\d+\.\s+/, ""), rawHeader: line.trim(), flags: [] };
          sections.push(current);
          break;
        }
        const fl = FLAG_LINE_RE.exec(line);
        if (fl && (pending === null || fl[1].length <= pendingIndent)) {
          flush();
          const { defaultValue, env } = parseTrailer(fl[3]);
          pending = {
            ...parseNames(fl[2]),
            optionalValue: false,
            repeatable: false,
            summary: "",
            description: "",
            defaultValue,
            env,
            possibleValues: [],
            required: false,
          };
          pendingIndent = fl[1].length;
          break;
        }
        if (pending) pendingBody.push(line);
      }
    }
  }
  flush();

  for (const s of sections) s.flags = s.flags.filter((f) => f.long !== "--help" && f.long !== "--version");
  const nonEmpty = sections.filter((s) => s.flags.length > 0);
  // Headers and category titles are printed in caps; render them as titles.
  for (const s of nonEmpty) {
    if (s.title === s.title.toUpperCase()) {
      s.title = s.title
        .toLowerCase()
        .replace(/\b([a-z])/g, (c) => c.toUpperCase())
        .replace(/\bRpc\b/g, "RPC")
        .replace(/\bApi\b/g, "API")
        .replace(/\bL1\b/gi, "L1")
        .replace(/\bAlt-Da\b/g, "Alt-DA")
        .replace(/\bP2p\b/gi, "P2P")
        .replace(/\bPeer-To-Peer\b/g, "Peer-to-Peer")
        .replace(/\b(And|Or|The|Of|For|To)\b/g, (w) => w.toLowerCase())
        // "GLOBAL OPTIONS" → "Global options", but a bare "OPTIONS" stays "Options".
        .replace(/^(\S.* )Options\b/, "$1options");
    }
  }
  return { path, description, usage, args: [], sections: nonEmpty, subcommands, rawHelp: help };
}
