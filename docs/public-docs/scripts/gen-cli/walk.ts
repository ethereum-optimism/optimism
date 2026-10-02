// Builds the CommandTree for a component by walking `--help` recursively
// through a HelpSource, parsing each output with the framework's parser, and
// scrubbing environment-dependent text so the output is byte-stable.

import type { Component } from "./components";
import type { CommandDoc, CommandTree } from "./model";
import { parseClapHelp } from "./parse-clap";
import { parseUrfaveHelp } from "./parse-urfave";
import type { HelpSource } from "./source";

/**
 * The same environment-dependent-output scrubs upstream reth's docs tooling
 * (help.rs preprocess_help) applies, plus the op-reth spelling. Applied to
 * every raw help text and every default value, whatever the framework.
 */
const SCRUBS: Array<[RegExp, string]> = [
  [/default: op-reth\/.*-[0-9A-Fa-f]{6,10}\/[^\]\s]+/g, "default: op-reth/<VERSION>-<SHA>/<ARCH>"],
  [/default: \/.*\/reth/g, "default: <CACHE_DIR>"],
  [/default: reth\/.*-[0-9A-Fa-f]{6,10}\/([_\w]+)-(\w+)-(\w+)(-\w+)?/g, "default: reth/<VERSION>-<SHA>/<ARCH>"],
  [/default: reth\/.*\/\w+/g, "default: reth/<VERSION>/<OS>"],
  [/(rpc.max-tracing-requests <COUNT>\n.*\n.*\n.*\n.*\n.*)\[default: \d+\]/g, "$1[default: <NUM CPU CORES-2>]"],
  [/(engine\.reserved-cpu-cores.*)\[default: \d+\]/g, "$1[default: <DYNAMIC: min(2, CPU cores)>]"],
];

const DEFAULT_SCRUBS: Array<[RegExp, string]> = [
  [/^op-reth\/.*-[0-9A-Fa-f]{6,10}\/.+$/, "op-reth/<VERSION>-<SHA>/<ARCH>"],
  [/^\/.*\/reth$/, "<CACHE_DIR>"],
  [/^reth\/.*-[0-9A-Fa-f]{6,10}\/.+$/, "reth/<VERSION>-<SHA>/<ARCH>"],
  [/^reth\/.*\/\w+$/, "reth/<VERSION>/<OS>"],
];

export function scrubHelp(text: string): string {
  let out = text;
  for (const [re, rep] of SCRUBS) out = out.replace(re, rep);
  return out;
}

function scrubDefault(value: string | null, flag: string): string | null {
  if (value === null) return null;
  for (const [re, rep] of DEFAULT_SCRUBS) if (re.test(value)) return rep;
  if (flag === "--rpc.max-tracing-requests" && /^\d+$/.test(value)) return "<NUM CPU CORES-2>";
  if (flag === "--engine.reserved-cpu-cores" && /^\d+$/.test(value)) return "<DYNAMIC: min(2, CPU cores)>";
  return value;
}

function parse(component: Component, path: string[], help: string): CommandDoc {
  const doc = component.framework === "clap" ? parseClapHelp(path, help) : parseUrfaveHelp(path, help);
  let raw = scrubHelp(doc.rawHelp);
  for (const [re, rep] of component.canonicalizeHelp ?? []) raw = raw.replace(re, rep);
  doc.rawHelp = raw.split("\n").map((l) => l.replace(/\s+$/, "")).join("\n").trim();
  for (const section of doc.sections) {
    for (const flag of section.flags) {
      flag.defaultValue = scrubDefault(flag.defaultValue, flag.long);
      const rule = component.canonicalize?.find((c) => c.flag === flag.long);
      if (rule) {
        flag.description = rule.rewrite(flag.description);
        flag.summary = rule.rewrite(flag.summary);
      }
    }
  }
  return doc;
}

export function walkCommands(component: Component, source: HelpSource, path: string[] = []): CommandTree {
  const doc = parse(component, path, source.help(path));
  const children = doc.subcommands.map((sub) => walkCommands(component, source, [...path, sub.name]));
  return { doc, children };
}
