// Shared data model for gen-cli: the framework-neutral shape both help
// parsers (clap, urfave/cli) produce and the emitter consumes.

export interface Flag {
  /** Long name including dashes, e.g. "--datadir". */
  long: string;
  /** Short alias including dash, e.g. "-d", or null. */
  short: string | null;
  /** Further long names the flag answers to, including dashes. */
  aliases: string[];
  /** Value placeholder as printed, e.g. "<PATH>" or "value", or null for booleans. */
  value: string | null;
  /** clap: the value is optional (`--flag[=<VALUE>]`). */
  optionalValue: boolean;
  /** clap: the flag may repeat (`--flag...`). */
  repeatable: boolean;
  /** First paragraph of the description, single line. */
  summary: string;
  /** Full description text as printed (paragraphs joined by blank lines). */
  description: string;
  /** Default value as printed, or null. */
  defaultValue: string | null;
  /** Environment variable(s) that set the flag, as printed. */
  env: string[];
  /** Enumerated allowed values, when the help lists them. */
  possibleValues: string[];
  /** Whether the help marks the flag as required (urfave prints this in usage). */
  required: boolean;
}

export interface FlagSection {
  /** Section title as rendered ("Options", "Logging", "Global options", "Rollup"). */
  title: string;
  /** The line that opens the section in the raw help, exactly as printed and trimmed ("Logging:", "GLOBAL OPTIONS:", "1. ROLLUP"). */
  rawHeader: string;
  flags: Flag[];
}

export interface PositionalArg {
  name: string;
  description: string;
}

export interface SubcommandRef {
  name: string;
  summary: string;
}

export interface CommandDoc {
  /** Subcommand path below the binary, e.g. ["db", "checksum", "mdbx"]; [] for the root. */
  path: string[];
  /** One-line description printed above Usage, or "". */
  description: string;
  /** The Usage line(s) as printed, without the "Usage:" label. */
  usage: string;
  args: PositionalArg[];
  sections: FlagSection[];
  subcommands: SubcommandRef[];
  /** The full --help output after environment scrubbing. */
  rawHelp: string;
}

export interface CommandTree {
  doc: CommandDoc;
  children: CommandTree[];
}

export type Framework = "clap" | "urfave";

/** Stable identity of a flag section's content, used to detect repeats. */
export function sectionKey(section: FlagSection): string {
  return JSON.stringify([
    section.title,
    section.flags.map((f) => [f.long, f.short, f.aliases, f.value, f.optionalValue, f.repeatable, f.description, f.defaultValue, f.env, f.possibleValues]),
  ]);
}

export function displayCommand(binary: string, path: string[]): string {
  return [binary, ...path].join(" ");
}

/** Walk a tree depth-first, root first. */
export function* walkTree(tree: CommandTree): Generator<CommandTree> {
  yield tree;
  for (const child of tree.children) yield* walkTree(child);
}
