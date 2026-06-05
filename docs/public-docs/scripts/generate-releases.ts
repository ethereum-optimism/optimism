import * as fs from "fs";
import * as path from "path";

const GITHUB_ORG = "ethereum-optimism";
const GITHUB_API = "https://api.github.com";
const MAX_RELEASES_PER_COMPONENT = 20;
const MAX_BODY_LINES = 40;
const OUTPUT_DIR = path.join(process.cwd(), "releases");

// ─── Types ────────────────────────────────────────────────────────────────────

interface Component {
  slug: string;
  prefix: string;
  label: string;
  description: string; // plain text — used in frontmatter meta
  intro: string;       // MDX — used as the page intro paragraph (may contain links)
  group: string;
  icon: string;
  repo: string; // e.g. "optimism" or "infra"
}

interface GitHubRelease {
  id: number;
  tag_name: string;
  name: string | null;
  body: string | null;
  published_at: string;
  html_url: string;
  draft: boolean;
  prerelease: boolean;
}

// ─── Configuration ────────────────────────────────────────────────────────────

// Tag prefix formats confirmed from .goreleaser.yaml, release.toml, and
// ops/scripts/find_release_tag.sh in the monorepo.
// Order controls the Latest Releases card list on the index page.
const COMPONENTS: Component[] = [
  // ── optimism repo ──────────────────────────────────────────────────────────
  {
    slug: "op-node",
    prefix: "op-node/v",
    label: "op-node",
    description: "OP Stack consensus-layer client",
    intro: "**op-node** implements the [rollup-node spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/rollup-node.md), functioning as the consensus layer (CL) client of an OP Stack chain. It builds, relays, and verifies the canonical L2 chain, working alongside an execution layer client such as op-reth.",
    group: "Protocol",
    icon: "cube",
    repo: "optimism",
  },
  {
    slug: "kona-node",
    prefix: "kona-node/v",
    label: "kona-node",
    description: "Rust implementation of the OP Stack rollup node",
    intro: "**kona-node** is a Rust implementation of the [rollup-node spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/rollup-node.md), backed by kona-derive. It is a Rust-native alternative to op-node.",
    group: "Protocol",
    icon: "circle-nodes",
    repo: "optimism",
  },
  // Execution
  {
    slug: "op-reth",
    prefix: "op-reth/v",
    label: "op-reth",
    description: "OP Stack execution layer built on Reth",
    intro: "**op-reth** is the OP Stack execution layer (EL) built on [Reth](https://reth.rs), a modular, high-performance Rust Ethereum node. It is the recommended execution client for OP Stack chains, replacing op-geth.",
    group: "Protocol",
    icon: "bolt",
    repo: "optimism",
  },
  // Go — consensus / sequencing
  {
    slug: "op-batcher",
    prefix: "op-batcher/v",
    label: "op-batcher",
    description: "L2 batch submitter",
    intro: "**op-batcher** is responsible for data availability — it reads unsafe blocks from the sequencer and posts transaction batches to the DA layer (L1 or Alt DA). See the [batcher spec](https://specs.optimism.io/protocol/batcher.html).",
    group: "Protocol",
    icon: "layer-group",
    repo: "optimism",
  },
  {
    slug: "op-proposer",
    prefix: "op-proposer/v",
    label: "op-proposer",
    description: "L2 output root proposer",
    intro: "**op-proposer** automates output-root proposal transactions on L1 at a regular interval, submitting claims of L2 state that enable withdrawals. See the [proposals spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/proposals.md).",
    group: "Protocol",
    icon: "stamp",
    repo: "optimism",
  },
  {
    slug: "op-challenger",
    prefix: "op-challenger/v",
    label: "op-challenger",
    description: "Dispute game challenger",
    intro: "**op-challenger** is a modular dispute game agent that monitors and challenges invalid fault proof games on-chain. See the [fault proof specs](https://specs.optimism.io/experimental/fault-proof/index.html).",
    group: "Protocol",
    icon: "shield",
    repo: "optimism",
  },
  // Contracts
  {
    slug: "op-contracts",
    prefix: "op-contracts/v",
    label: "op-contracts",
    description: "OP Stack L1 and L2 smart contracts",
    intro: "**op-contracts** contains the L1 and L2 smart contracts for the OP Stack. See the [contract reference](https://devdocs.optimism.io/contracts-bedrock) for interface documentation and architecture details.",
    group: "Protocol",
    icon: "file-code",
    repo: "optimism",
  },
  {
    slug: "op-conductor",
    prefix: "op-conductor/v",
    label: "op-conductor",
    description: "High-availability sequencer coordination service",
    intro: "**op-conductor** manages sequencer high-availability using Raft consensus, coordinating a multi-node sequencer cluster to ensure no unsafe reorgs and continuous uptime with up to one node failure.",
    group: "Tooling",
    icon: "sliders",
    repo: "optimism",
  },
  // Rust / Fault Proofs — kona workspace uses per-crate tags
  {
    slug: "kona-client",
    prefix: "kona-client/v",
    label: "kona-client",
    description: "Rust fault proof client binary",
    intro: "**kona-client** is the bare-metal fault proof program that executes the OP Stack state transition on a MIPS prover. It is the primary fault proof program for OP Stack chains. See the [fault proof specs](https://specs.optimism.io/experimental/fault-proof/index.html).",
    group: "Fault Proofs",
    icon: "microchip",
    repo: "optimism",
  },
  {
    slug: "kona-host",
    prefix: "kona-host/v",
    label: "kona-host",
    description: "Rust fault proof host binary",
    intro: "**kona-host** runs natively alongside the prover, serving as the [Preimage Oracle](https://specs.optimism.io/experimental/fault-proof/index.html) server that supplies chain data to kona-client during proof execution.",
    group: "Fault Proofs",
    icon: "server",
    repo: "optimism",
  },
  // Tooling
  {
    slug: "op-deployer",
    prefix: "op-deployer/v",
    label: "op-deployer",
    description: "OP Stack chain deployment and upgrade tool",
    intro: "**op-deployer** automates deploying and upgrading OP Stack smart contracts. See the [usage docs](https://docs.optimism.io/chain-operators/tools/op-deployer/overview) for full configuration reference.",
    group: "Tooling",
    icon: "rocket",
    repo: "optimism",
  },
  // ── infra repo ─────────────────────────────────────────────────────────────
  {
    slug: "proxyd",
    prefix: "proxyd/v",
    label: "proxyd",
    description: "RPC request router and proxy",
    intro: "**proxyd** is an RPC request router and proxy with method whitelisting, backend load balancing, automatic retries, and consensus state tracking across multiple backend nodes.",
    group: "Tooling",
    icon: "network-wired",
    repo: "infra",
  },
  {
    slug: "op-acceptor",
    prefix: "op-acceptor/v",
    label: "op-acceptor",
    description: "Network acceptance tester for OP Stack devnets",
    intro: "**op-acceptor** runs validation checks against a network to determine if it is ready for production, executing standard Go tests against configurable gate-based validation scenarios.",
    group: "Tooling",
    icon: "circle-check",
    repo: "infra",
  },
];

// ─── GitHub API ───────────────────────────────────────────────────────────────

async function fetchReleasesForRepo(repo: string): Promise<GitHubRelease[]> {
  const token = process.env.GITHUB_TOKEN;
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    "User-Agent": "optimism-docs-generator",
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const releases: GitHubRelease[] = [];
  const maxPages = 10;
  const apiBase = `${GITHUB_API}/repos/${GITHUB_ORG}/${repo}`;

  for (let page = 1; page <= maxPages; page++) {
    const url = `${apiBase}/releases?per_page=100&page=${page}`;
    console.log(`  Fetching releases page ${page}...`);

    const response = await fetch(url, { headers });

    if (!response.ok) {
      if (response.status === 403) {
        const remaining = response.headers.get("X-RateLimit-Remaining");
        if (remaining === "0") {
          console.warn(
            "  ⚠ GitHub API rate limit exceeded. Set GITHUB_TOKEN env var to increase the limit."
          );
          break;
        }
      }
      throw new Error(
        `GitHub API error: ${response.status} ${response.statusText}`
      );
    }

    const page_releases = (await response.json()) as GitHubRelease[];
    if (page_releases.length === 0) break;

    // Exclude drafts; include prereleases (RC builds are user-relevant)
    releases.push(...page_releases.filter((r) => !r.draft));

    if (page_releases.length < 100) break;
  }

  console.log(`  Fetched ${releases.length} total releases`);
  return releases;
}

// ─── MDX helpers ─────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

// Escape JSX expression delimiters in non-code text so the MDX parser treats
// them as literal characters.
function escapeOutsideCodeBlocks(text: string): string {
  // Split on fenced code blocks and inline code spans; leave those untouched.
  const parts = text.split(/(```[\s\S]*?```|`[^`\n]+`)/);
  return parts
    .map((part, idx) => {
      if (idx % 2 === 1) return part; // code block — leave as-is
      return part.replace(/\{/g, "\\{").replace(/\}/g, "\\}");
    })
    .join("");
}

function processBody(body: string | null): string {
  if (!body || body.trim() === "") return "";

  // Remove <details>...</details> blocks (common in auto-generated GH notes)
  let out = body.replace(/<details[\s\S]*?<\/details>/gi, "");

  // Remove HTML comments
  out = out.replace(/<!--[\s\S]*?-->/g, "");

  // Strip remaining HTML tags
  out = out.replace(/<[^>]+>/g, "");

  // Escape JSX expression delimiters outside code blocks
  out = escapeOutsideCodeBlocks(out);

  // Truncate at MAX_BODY_LINES or the first ## section heading after line 0
  const lines = out.split("\n");
  let truncateAt = Math.min(lines.length, MAX_BODY_LINES);
  for (let i = 1; i < lines.length; i++) {
    if (lines[i].startsWith("## ")) {
      truncateAt = i;
      break;
    }
  }

  return lines.slice(0, truncateAt).join("\n").trim();
}

// Strip the component prefix so "op-node/v1.18.2" → "v1.18.2".
// All prefixes end in "/v", so slicing at prefix.length - 1 keeps the "v".
function versionOnly(tagName: string, prefix: string): string {
  return tagName.startsWith(prefix) ? tagName.slice(prefix.length - 1) : tagName;
}

// Treat as a release candidate if GitHub marked it as prerelease OR if the
// tag name contains a pre-release qualifier (-rc., -alpha., -beta.).
function isRC(release: GitHubRelease): boolean {
  return release.prerelease || /-(rc|alpha|beta)\.\d/i.test(release.tag_name);
}

const AUTO_GENERATED_HEADER = `{/*
  ⚠️ WARNING: DO NOT EDIT THIS FILE DIRECTLY

  This file is auto-generated by scripts/generate-releases.ts.
  Re-run \`pnpm prebuild\` to refresh.
*/}

`;

// ─── Per-component page ───────────────────────────────────────────────────────

function generateComponentMdx(
  component: Component,
  releases: GitHubRelease[]
): string {
  const all = releases.filter((r) => r.tag_name.startsWith(component.prefix));
  const filtered = all.slice(0, MAX_RELEASES_PER_COMPONENT);

  // Always include the most recent RC so the Release Candidate filter renders
  // even when no RC falls within the top MAX_RELEASES_PER_COMPONENT window.
  const latestRC = all.find((r) => isRC(r));
  if (latestRC && !filtered.some((r) => r.id === latestRC.id)) {
    filtered.push(latestRC);
    filtered.sort(
      (a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime()
    );
  }

  const lines: string[] = [
    `---`,
    `title: "${component.label} Releases"`,
    `sidebarTitle: "${component.label}"`,
    `description: "Release history for ${component.label}. ${component.description}."`,
    `rss: true`,
    `---`,
    ``,
    AUTO_GENERATED_HEADER.trimEnd(),
    ``,
    component.intro,
    ``,
  ];

  if (filtered.length === 0) {
    lines.push(`No releases found for \`${component.prefix}*\`.`);
    return lines.join("\n");
  }

  const latestStableIdx = filtered.findIndex((r) => !isRC(r));

  for (let i = 0; i < filtered.length; i++) {
    const release = filtered[i];
    const label = formatDate(release.published_at);
    const version = versionOnly(release.tag_name, component.prefix);
    const tags = isRC(release)
      ? ["Release Candidate"]
      : i === latestStableIdx
        ? ["Latest", "Stable"]
        : ["Stable"];
    const body = processBody(release.body);

    lines.push(`<Update label="${label}" description="${version}" tags={[${tags.map((t) => `"${t}"`).join(", ")}]}>`);
    lines.push(``);
    if (body) {
      lines.push(body);
      lines.push(``);
    }
    lines.push(`[View full release on GitHub →](${release.html_url})`);
    lines.push(``);
    lines.push(`</Update>`);
    lines.push(``);
  }

  return lines.join("\n");
}

// ─── Index page ───────────────────────────────────────────────────────────────

function generateIndexMdx(
  latestByComponent: Map<string, GitHubRelease>
): string {
  const lines: string[] = [
    `---`,
    `title: "Releases"`,
    `sidebarTitle: "Overview"`,
    `description: "Latest stable releases for all OP Stack components. Select a component to view its full release history."`,
    `---`,
    ``,
    AUTO_GENERATED_HEADER.trimEnd(),
    ``,
    `<Info>`,
    `  We always recommend running the latest stable release for each component. Select a component below to view its full release history and changelog.`,
    `</Info>`,
    ``,
    `## Latest Releases`,
    ``,
    `<CardGroup cols={1}>`,
  ];

  for (const component of COMPONENTS) {
    const latest = latestByComponent.get(component.slug);
    if (!latest) continue;
    const date = formatDate(latest.published_at);
    const version = versionOnly(latest.tag_name, component.prefix);
    lines.push(
      `  <Card title="${component.label} ${version}" href="/releases/${component.slug}" icon="${component.icon}" horizontal>`
    );
    lines.push(`    Released ${date}`);
    lines.push(`  </Card>`);
  }

  lines.push(`</CardGroup>`);

  return lines.join("\n");
}

// ─── Main ─────────────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  console.log("Generating releases pages...");

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  // Fetch from every distinct repo referenced by COMPONENTS.
  const repos = [...new Set(COMPONENTS.map((c) => c.repo))];
  const releasesByRepo = new Map<string, GitHubRelease[]>();
  for (const repo of repos) {
    console.log(`\nFetching ${GITHUB_ORG}/${repo}...`);
    releasesByRepo.set(repo, await fetchReleasesForRepo(repo));
  }

  // Merge into a flat array; each component only sees its own repo's releases.
  const releases = (repo: string) => releasesByRepo.get(repo) ?? [];

  // Find the most recent release for each component
  const latestByComponent = new Map<string, GitHubRelease>();
  for (const component of COMPONENTS) {
    const latest = releases(component.repo).find((r: GitHubRelease) => r.tag_name.startsWith(component.prefix));
    if (latest) {
      latestByComponent.set(component.slug, latest);
    }
  }

  // Per-component pages
  for (const component of COMPONENTS) {
    const mdx = generateComponentMdx(component, releases(component.repo));
    const outPath = path.join(OUTPUT_DIR, `${component.slug}.mdx`);
    fs.writeFileSync(outPath, mdx, "utf-8");
    const latest = latestByComponent.get(component.slug);
    const tag = latest ? ` (latest: ${latest.tag_name})` : " (no releases found)";
    console.log(`  ✓ ${component.slug}${tag}`);
  }

  // Index page
  const indexMdx = generateIndexMdx(latestByComponent);
  fs.writeFileSync(path.join(OUTPUT_DIR, "index.mdx"), indexMdx, "utf-8");
  console.log(`  ✓ index`);

  console.log(
    `Done. Generated ${COMPONENTS.length + 1} files in releases/`
  );
}

main().catch((err) => {
  console.error("Error generating releases pages:", err);
  process.exit(1);
});
