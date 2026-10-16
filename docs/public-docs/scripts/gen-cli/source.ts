// Help sources for gen-cli: where `--help` text comes from.
//
//  - BinarySource   a local executable (--bin), for development and for
//                   op-deployer's downloaded release binary.
//  - ImageSource    a Docker image (--image), the CI path for the components
//                   that publish only images.
//  - FixtureSource  a recorded help tree (--help-json), for deterministic
//                   tests and for regenerating from a recording; every source
//                   can be recorded with --dump-help-json.
//
// All sources run help with the same environment upstream reth's docs
// tooling uses (NO_COLOR, fixed COLUMNS) so wrapping is stable.

import { execFileSync } from "child_process";
import * as fs from "fs";

export interface HelpSource {
  /** `--help` output for the given subcommand path. */
  help(path: string[]): string;
  /** First line of `--version` output, or null when unavailable. */
  version(): string | null;
  /** Human description of the source, recorded in provenance. */
  describe(): string;
}

const HELP_ENV = { NO_COLOR: "1", COLUMNS: "100", LINES: "10000", TERM: "dumb" };

function run(cmd: string, args: string[]): string {
  return execFileSync(cmd, args, {
    encoding: "utf8",
    env: { ...process.env, ...HELP_ENV },
    maxBuffer: 32 * 1024 * 1024,
    stdio: ["ignore", "pipe", "pipe"],
  });
}

export class BinarySource implements HelpSource {
  constructor(private readonly bin: string) {}
  help(path: string[]): string {
    return run(this.bin, [...path, "--help"]);
  }
  version(): string | null {
    try {
      return run(this.bin, ["--version"]).trim().split("\n")[0] ?? null;
    } catch {
      return null;
    }
  }
  describe(): string {
    return `binary ${this.bin}`;
  }
}

export class ImageSource implements HelpSource {
  constructor(private readonly image: string) {}
  private docker(args: string[]): string {
    const envArgs = Object.entries(HELP_ENV).flatMap(([k, v]) => ["-e", `${k}=${v}`]);
    return run("docker", ["run", "--rm", ...envArgs, this.image, ...args]);
  }
  help(path: string[]): string {
    return this.docker([...path, "--help"]);
  }
  version(): string | null {
    try {
      return this.docker(["--version"]).trim().split("\n")[0] ?? null;
    } catch {
      return null;
    }
  }
  describe(): string {
    return `image ${this.image}`;
  }
}

export interface HelpFixture {
  binary: string;
  tag?: string;
  source?: string;
  version?: string | null;
  /** Keyed by the space-joined subcommand path; "" is the root. */
  help: Record<string, string>;
}

export class FixtureSource implements HelpSource {
  private readonly fixture: HelpFixture;
  constructor(private readonly file: string) {
    this.fixture = JSON.parse(fs.readFileSync(file, "utf8")) as HelpFixture;
  }
  help(path: string[]): string {
    const key = path.join(" ");
    const text = this.fixture.help[key];
    if (text === undefined) throw new Error(`fixture ${this.file} has no help for "${key}"`);
    return text;
  }
  version(): string | null {
    return this.fixture.version ?? null;
  }
  describe(): string {
    const name = this.file.split("/").pop() ?? this.file;
    return `fixture ${name}${this.fixture.source ? ` (${this.fixture.source})` : ""}`;
  }
}

/** Wraps a source and records every help text it serves, for --dump-help-json. */
export class RecordingSource implements HelpSource {
  readonly recorded: Record<string, string> = {};
  constructor(private readonly inner: HelpSource) {}
  help(path: string[]): string {
    const text = this.inner.help(path);
    this.recorded[path.join(" ")] = text;
    return text;
  }
  version(): string | null {
    return this.inner.version();
  }
  describe(): string {
    return this.inner.describe();
  }
}
