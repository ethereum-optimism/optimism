// Component registry for gen-cli: the nine OP Stack components whose CLI
// reference is generated, with everything the generator needs to know about
// each one that cannot be read from --help.
//
// Keep the list in sync with the "Components covered" list in the content
// guide (op-stack/contribute/content-guide.mdx) and the COMPONENTS set in
// scripts/lint/validate-reference.ts.

import type { Framework } from "./model";

export interface Component {
  /** Release-tag prefix and docs slug, e.g. "op-node" (tags are "op-node/vX.Y.Z"). */
  name: string;
  /** Binary name inside the artifact, when it differs from `name`. */
  binary?: string;
  framework: Framework;
  /**
   * Where the release artifact lives.
   *  - image: the Docker image published for the tag; `--help` is run with
   *    `docker run --rm <image>:<version> …`.
   *  - github-asset: a tarball attached to the GitHub release; the pattern
   *    names the linux/amd64 asset with `{version}` for the bare semver.
   */
  artifact:
    | { kind: "image"; repository: string }
    | { kind: "github-asset"; assetPattern: string; binaryInArchive: string };
  /** Human-facing sidebar title for the component dropdown. */
  title: string;
  /** Nav icon (Font Awesome name) for the component dropdown. */
  icon: string;
  /**
   * Flag-description rewrites that keep the output byte-stable across runs
   * and readable in a table. Each entry is applied to the matching flag's
   * description and summary; `usage` receives the text and returns the
   * replacement. Mirrors gen-flags' canonicalizeUsage.
   */
  canonicalize?: Array<{ flag: string; rewrite: (text: string) => string }>;
  /**
   * Rewrites applied to the verbatim --help text shown on every page, for
   * blocks that are long, environment-dependent, or repeated on every
   * command. Applied after the environment scrubs.
   */
  canonicalizeHelp?: Array<[RegExp, string]>;
}

const IMAGES = "us-docker.pkg.dev/oplabs-tools-artifacts/images";

/**
 * op-reth's `--chain` and the Go services' `--network` enumerate every chain
 * bundled from the superchain-registry, which changes with each registry
 * bump independently of the component. Replace the list with a pointer so
 * the tables stay stable and the row stays readable.
 */
function pointAtRegistry(listMarker: string, component: string) {
  return (text: string): string => {
    const i = text.indexOf(listMarker);
    if (i === -1) return text;
    return (
      text.slice(0, i + listMarker.length) +
      " every chain bundled from the [superchain-registry](https://github.com/ethereum-optimism/superchain-registry) at the release; run `" +
      component +
      " --help` for the exact list."
    );
  };
}

export const COMPONENTS: Component[] = [
  {
    name: "op-node",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-node` },
    title: "op-node",
    icon: "cube",
    canonicalize: [{ flag: "--network", rewrite: pointAtRegistry("Available networks:", "op-node") }],
    canonicalizeHelp: [
      // The list wraps onto continuation lines indented under the usage.
      [/(Available networks: )[^\n]+(?:\n\s+[a-z0-9_.-]+(?:,\s*[a-z0-9_.-]+)*,?)*/g, "$1<every chain bundled from the superchain-registry at this release>"],
    ],
  },
  {
    name: "op-batcher",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-batcher` },
    title: "op-batcher",
    icon: "layer-group",
    canonicalize: [
      {
        flag: "--compressor",
        // The option list is built from map iteration in op-batcher and
        // changes order on every start; sort it so the table is stable.
        rewrite: (text) => {
          const marker = "Valid options: ";
          const i = text.indexOf(marker);
          if (i === -1) return text;
          const head = text.slice(0, i + marker.length);
          const list = text.slice(i + marker.length).replace(/\.$/, "");
          return head + list.split(", ").sort().join(", ");
        },
      },
    ],
  },
  {
    name: "op-proposer",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-proposer` },
    title: "op-proposer",
    icon: "paper-plane",
  },
  {
    name: "op-challenger",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-challenger` },
    title: "op-challenger",
    icon: "shield-halved",
    canonicalize: [{ flag: "--network", rewrite: pointAtRegistry("Available networks:", "op-challenger") }],
  },
  {
    name: "op-conductor",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-conductor` },
    title: "op-conductor",
    icon: "arrows-split-up-and-left",
    canonicalize: [{ flag: "--network", rewrite: pointAtRegistry("Available networks:", "op-conductor") }],
  },
  {
    name: "op-supernode",
    framework: "urfave",
    artifact: { kind: "image", repository: `${IMAGES}/op-supernode` },
    title: "op-supernode",
    icon: "circle-nodes",
  },
  {
    name: "op-deployer",
    framework: "urfave",
    artifact: {
      kind: "github-asset",
      assetPattern: "op-deployer-{version}-linux-amd64.tar.gz",
      binaryInArchive: "op-deployer",
    },
    title: "op-deployer",
    icon: "rocket",
  },
  {
    name: "op-reth",
    framework: "clap",
    artifact: { kind: "image", repository: `${IMAGES}/op-reth` },
    title: "op-reth",
    icon: "gear",
    canonicalize: [{ flag: "--chain", rewrite: pointAtRegistry("Built-in chains:", "op-reth") }],
    canonicalizeHelp: [
      // The bundled-chain list changes with every superchain-registry bump
      // and is repeated on every command; keep the pointer, drop the list.
      [/(Built-in chains:\n\s+)[^\n]+/g, "$1<every chain bundled from the superchain-registry at this release>"],
    ],
  },
  {
    name: "kona-node",
    framework: "clap",
    artifact: { kind: "image", repository: `${IMAGES}/kona-node` },
    title: "kona-node",
    icon: "circle-nodes",
  },
];

export function findComponent(name: string): Component {
  const c = COMPONENTS.find((x) => x.name === name);
  if (!c) {
    throw new Error(`unknown component "${name}"; known: ${COMPONENTS.map((x) => x.name).join(", ")}`);
  }
  return c;
}

/** "op-node/v1.19.7" → { version: "v1.19.7", line: "v1.19" }. */
export function parseTag(component: string, tag: string): { version: string; line: string } {
  const m = new RegExp(`^${component.replace(/[-/]/g, "\\$&")}/(v(\\d+)\\.(\\d+)\\.(\\d+))$`).exec(tag);
  if (!m) {
    throw new Error(`--tag must be a finalized ${component} release tag (${component}/vX.Y.Z), got: ${tag}`);
  }
  return { version: m[1], line: `v${m[2]}.${m[3]}` };
}
