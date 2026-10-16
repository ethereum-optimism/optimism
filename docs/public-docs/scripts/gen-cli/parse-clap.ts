// clap (Rust) --help parser for gen-cli. Handles the long-help layout clap
// prints for op-reth and kona-node:
//
//   <description>
//
//   Usage: op-reth db checksum mdbx [OPTIONS] <TABLE>
//
//   Commands:
//     mdbx         Calculates the checksum of a database table
//     help         Print this message or the help of the given subcommand(s)
//
//   Arguments:
//     <TABLE>
//             The table name
//
//   Options:
//         --start-key <START_KEY>
//             The start of the range to checksum
//
//     -h, --help
//             Print help (see a summary with '-h')
//
//   Logging:
//         --log.stdout.format <FORMAT>
//             The format to use for logs written to stdout
//
//             Possible values:
//             - json:     …
//
//             [default: terminal]
//             [env: RUST_LOG=]
//
// Every header ends with ":" at column 0; flag lines are indented 2 or 6
// spaces and start with "-"; description lines are indented 10 spaces.

import type { CommandDoc, Flag, FlagSection, PositionalArg, SubcommandRef } from "./model";

const HEADER_RE = /^([A-Z][A-Za-z0-9 /_-]*):\s*$/;
// `  -s, --long-name <VALUE> <VALUE2>...`, `  --flag[=<VALUE>]`, `  -v, --verbosity...`
const FLAG_LINE_RE =
  /^ {2,6}(?:(-[A-Za-z0-9]),\s+)?(--[A-Za-z0-9](?:[A-Za-z0-9_-]|\.(?!\.))*)(?:\[=(<[^>]+>)\]|\s+((?:<[^>]+>|\[[^\]]+\])(?:\s+(?:<[^>]+>|\[[^\]]+\]))*))?(\.\.\.)?\s*$/;
// Any other indented dash line inside an options section is a flag shape the
// parser does not know; fail loudly rather than fold it into the previous flag.
const UNKNOWN_FLAG_LINE_RE = /^ {2,6}-/;
const ARG_LINE_RE = /^ {2}(<[^>]+>|\[[^\]]+\])(\.\.\.)?\s*$/;
const COMMAND_LINE_RE = /^ {2}([a-z0-9][a-z0-9._-]*)\s{2,}(.*\S)\s*$/;

/** Subcommand names reach process arguments and file paths; accept only clap's shape. */
export const SUBCOMMAND_NAME = /^[a-z0-9][a-z0-9._-]*$/;

function firstParagraph(text: string): string {
  const para = text.split(/\n\s*\n/)[0] ?? "";
  return para.split("\n").map((l) => l.trim()).join(" ").trim();
}

/** Parse one flag's description block into its parts. */
function parseFlagBody(lines: string[]): Pick<Flag, "summary" | "description" | "defaultValue" | "env" | "possibleValues" | "aliases"> {
  const body = lines.map((l) => l.replace(/^ {10}/, "").replace(/\s+$/, ""));
  let defaultValue: string | null = null;
  const env: string[] = [];
  const possibleValues: string[] = [];
  const aliases: string[] = [];
  const kept: string[] = [];
  let inPossible = false;
  for (const line of body) {
    const bracket = /^\[(default|env|possible values|aliases): ?(.*)\]$/i.exec(line.trim());
    if (bracket) {
      const key = bracket[1].toLowerCase();
      if (key === "default") defaultValue = bracket[2];
      else if (key === "env") env.push(...bracket[2].split(",").map((e) => e.trim().replace(/=$/, "")).filter(Boolean));
      else if (key === "possible values") possibleValues.push(...bracket[2].split(",").map((v) => v.trim()).filter(Boolean));
      else if (key === "aliases") aliases.push(...bracket[2].split(",").map((a) => a.trim()).filter(Boolean).map((a) => (a.startsWith("-") ? a : `--${a}`)));
      continue;
    }
    if (/^Possible values:$/i.test(line.trim())) {
      inPossible = true;
      continue;
    }
    if (inPossible) {
      const item = /^-\s+([^:\s]+):?/.exec(line.trim());
      if (item) {
        possibleValues.push(item[1]);
        continue;
      }
      if (line.trim() === "") continue;
      inPossible = false;
    }
    kept.push(line);
  }
  const description = kept.join("\n").replace(/\n{3,}/g, "\n\n").trim();
  return { summary: firstParagraph(description), description, defaultValue, env, possibleValues, aliases };
}

export function parseClapHelp(path: string[], help: string): CommandDoc {
  const lines = help.replace(/\r\n?/g, "\n").split("\n");

  // Description: everything before "Usage:".
  const usageIdx = lines.findIndex((l) => /^Usage:/.test(l));
  const description = usageIdx > 0 ? lines.slice(0, usageIdx).join("\n").trim().split("\n")[0] ?? "" : "";
  let usage = "";
  let i = usageIdx === -1 ? 0 : usageIdx;
  if (usageIdx !== -1) {
    usage = lines[i].replace(/^Usage:\s*/, "").trim();
    i++;
    // Continuation usage lines (indented, before the first blank line).
    while (i < lines.length && lines[i].trim() !== "" && !HEADER_RE.test(lines[i])) {
      usage += "\n" + lines[i].trim();
      i++;
    }
  }

  const subcommands: SubcommandRef[] = [];
  const args: PositionalArg[] = [];
  const sections: FlagSection[] = [];

  let section: string | null = null;
  let current: FlagSection | null = null;
  let pendingFlag: Flag | null = null;
  let pendingBody: string[] = [];
  let pendingArg: PositionalArg | null = null;
  let pendingArgBody: string[] = [];

  const flushFlag = () => {
    if (pendingFlag && current) {
      Object.assign(pendingFlag, parseFlagBody(pendingBody));
      current.flags.push(pendingFlag);
    }
    pendingFlag = null;
    pendingBody = [];
  };
  const flushArg = () => {
    if (pendingArg) {
      pendingArg.description = firstParagraph(pendingArgBody.map((l) => l.trim()).join("\n"));
      args.push(pendingArg);
    }
    pendingArg = null;
    pendingArgBody = [];
  };

  for (; i < lines.length; i++) {
    const line = lines[i];
    const header = HEADER_RE.exec(line);
    if (header) {
      flushFlag();
      flushArg();
      section = header[1];
      if (section === "Commands" || section === "Arguments") {
        current = null;
      } else {
        current = { title: section, rawHeader: line.trim(), flags: [] };
        sections.push(current);
      }
      continue;
    }
    if (section === "Commands") {
      if (line.trim() === "") continue;
      const m = COMMAND_LINE_RE.exec(line);
      if (!m) {
        // A wrapped description continuation is indented deeper than a name.
        if (/^ {4,}\S/.test(line) && subcommands.length > 0) continue;
        throw new Error(`cannot parse Commands line in --help: ${JSON.stringify(line)}`);
      }
      if (m[1] === "help") continue;
      if (!SUBCOMMAND_NAME.test(m[1])) throw new Error(`unexpected subcommand name in --help: ${JSON.stringify(m[1])}`);
      subcommands.push({ name: m[1], summary: m[2] });
      continue;
    }
    if (section === "Arguments") {
      const m = ARG_LINE_RE.exec(line);
      if (m) {
        flushArg();
        pendingArg = { name: m[1], description: "" };
      } else if (pendingArg) {
        pendingArgBody.push(line);
      }
      continue;
    }
    if (current) {
      const m = FLAG_LINE_RE.exec(line);
      if (m) {
        flushFlag();
        pendingFlag = {
          long: m[2],
          short: m[1] ?? null,
          aliases: [],
          value: m[3] ?? m[4] ?? null,
          optionalValue: m[3] !== undefined,
          repeatable: m[5] !== undefined,
          summary: "",
          description: "",
          defaultValue: null,
          env: [],
          possibleValues: [],
          required: false,
        };
        continue;
      }
      if (UNKNOWN_FLAG_LINE_RE.test(line)) {
        throw new Error(`cannot parse flag line in --help for "${path.join(" ")}": ${JSON.stringify(line)}`);
      }
      if (pendingFlag) pendingBody.push(line);
    }
  }
  flushFlag();
  flushArg();

  // clap always adds -h/--help; it is not a configuration surface.
  for (const s of sections) s.flags = s.flags.filter((f) => f.long !== "--help" && f.long !== "--version");
  const nonEmpty = sections.filter((s) => s.flags.length > 0);

  return { path, description, usage, args, sections: nonEmpty, subcommands, rawHelp: help };
}
