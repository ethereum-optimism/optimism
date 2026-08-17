#!/usr/bin/env bun

/**
 * Authoritative post-merge gate for webhook pushes to develop.
 *
 * CircleCI supplies the check identity: this script runs as the only job in the
 * `post-merge-gate` workflow, so the CircleCI GitHub App publishes the stable
 * `circleci-checks / post-merge-gate` check. The script polls the exact current
 * CircleCI pipeline plus exact-SHA GitHub Actions push runs and fails closed.
 */

const DEFAULT_CIRCLECI_API_BASE = "https://circleci.com/api/v2";
const DEFAULT_GITHUB_API_BASE = "https://api.github.com";
const DEFAULT_TIMEOUT_SECONDS = 4 * 60 * 60;
const DEFAULT_POLL_INTERVAL_SECONDS = 60;
const MAX_API_PAGES = 20;
const MAX_API_ATTEMPTS = 4;

export interface CirclePolicyEntry {
  route: string;
  workflow?: string;
  required_jobs?: string[];
  excluded?: string;
}

export interface GithubPolicyEntry {
  workflow: string;
  path: string;
}

export interface PostMergePolicy {
  circleci: {
    always: CirclePolicyEntry[];
    rust_changed: CirclePolicyEntry[];
    rust_unchanged: CirclePolicyEntry[];
  };
  github_actions: {
    always: GithubPolicyEntry[];
    security_changed: GithubPolicyEntry[];
  };
}

export interface ExpectedCircleWorkflow {
  name: string;
  requiredJobs: string[];
}

export interface ExpectedGithubWorkflow {
  name: string;
  path: string;
}

export interface ExpectedWorkflows {
  circleci: ExpectedCircleWorkflow[];
  github: ExpectedGithubWorkflow[];
}

export interface CirclePipeline {
  id: string;
  trigger?: {
    type?: string;
    received_at?: string;
  };
  vcs?: {
    revision?: string;
    branch?: string;
  };
}

export interface CircleWorkflow {
  id: string;
  name: string;
  status: string;
  created_at?: string;
  stopped_at?: string | null;
}

export interface CircleJob {
  id?: string;
  name: string;
  status: string;
  started_at?: string;
  stopped_at?: string | null;
}

export interface GithubWorkflowRun {
  id: number;
  name: string;
  path: string;
  event: string;
  status: string;
  conclusion?: string | null;
  head_sha: string;
  head_branch?: string | null;
  run_attempt?: number;
  created_at?: string;
  updated_at?: string;
  html_url?: string;
}

export type MemberResult = "pending" | "success" | "failure";

export interface MemberState {
  provider: "CircleCI" | "GitHub Actions";
  name: string;
  result: MemberResult;
  status: string;
  details?: string;
  url?: string;
}

export interface GateSnapshot {
  members: MemberState[];
  allTerminal: boolean;
  allSuccessful: boolean;
}

export interface CircleApi {
  getPipeline(id: string): Promise<CirclePipeline>;
  listPipelineWorkflows(id: string): Promise<CircleWorkflow[]>;
  listWorkflowJobs(id: string): Promise<CircleJob[]>;
}

export interface GithubApi {
  listWorkflowRuns(sha: string): Promise<GithubWorkflowRun[]>;
}

export function parseBoolean(value: string | undefined, name: string): boolean {
  if (value === "true" || value === "1") return true;
  if (value === "false" || value === "0") return false;
  throw new Error(`${name} must be one of true, false, 1, or 0; got ${JSON.stringify(value)}`);
}

function requireNonEmptyString(value: unknown, field: string): asserts value is string {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`post-merge policy field ${field} must be a non-empty string`);
  }
}

export function validatePolicy(policy: PostMergePolicy): void {
  const variants = ["always", "rust_changed", "rust_unchanged"] as const;
  const routes = new Set<string>();
  const workflows = new Set<string>();

  for (const variant of variants) {
    const entries = policy?.circleci?.[variant];
    if (!Array.isArray(entries)) {
      throw new Error(`post-merge policy circleci.${variant} must be an array`);
    }
    for (const [index, entry] of entries.entries()) {
      const prefix = `circleci.${variant}[${index}]`;
      requireNonEmptyString(entry.route, `${prefix}.route`);
      if (routes.has(entry.route)) {
        throw new Error(`duplicate post-merge CircleCI route: ${entry.route}`);
      }
      routes.add(entry.route);

      const isGated = entry.workflow !== undefined;
      const isExcluded = entry.excluded !== undefined;
      if (isGated === isExcluded) {
        throw new Error(`${prefix} must have exactly one of workflow or excluded`);
      }
      if (isGated) {
        requireNonEmptyString(entry.workflow, `${prefix}.workflow`);
        if (workflows.has(entry.workflow)) {
          throw new Error(`duplicate gated CircleCI workflow: ${entry.workflow}`);
        }
        workflows.add(entry.workflow);
        if (entry.required_jobs !== undefined) {
          if (!Array.isArray(entry.required_jobs) || entry.required_jobs.length === 0) {
            throw new Error(`${prefix}.required_jobs must be a non-empty array when set`);
          }
          for (const [jobIndex, job] of entry.required_jobs.entries()) {
            requireNonEmptyString(job, `${prefix}.required_jobs[${jobIndex}]`);
          }
        }
      } else {
        requireNonEmptyString(entry.excluded, `${prefix}.excluded`);
        if (entry.required_jobs !== undefined) {
          throw new Error(`${prefix} cannot set required_jobs on an excluded route`);
        }
      }
    }
  }

  const githubPaths = new Set<string>();
  for (const variant of ["always", "security_changed"] as const) {
    const entries = policy?.github_actions?.[variant];
    if (!Array.isArray(entries)) {
      throw new Error(`post-merge policy github_actions.${variant} must be an array`);
    }
    for (const [index, entry] of entries.entries()) {
      const prefix = `github_actions.${variant}[${index}]`;
      requireNonEmptyString(entry.workflow, `${prefix}.workflow`);
      requireNonEmptyString(entry.path, `${prefix}.path`);
      if (githubPaths.has(entry.path)) {
        throw new Error(`duplicate gated GitHub Actions workflow path: ${entry.path}`);
      }
      githubPaths.add(entry.path);
    }
  }
}

export function loadPolicy(path: string): PostMergePolicy {
  const result = Bun.spawnSync(["yq", "-o=json", ".post_merge_gate", path]);
  if (!result.success) {
    const stderr = new TextDecoder().decode(result.stderr).trim();
    throw new Error(`failed to read post-merge policy from ${path}: ${stderr || `exit ${result.exitCode}`}`);
  }
  const output = new TextDecoder().decode(result.stdout);
  const policy = JSON.parse(output) as PostMergePolicy;
  validatePolicy(policy);
  return policy;
}

export function buildExpectedWorkflows(
  policy: PostMergePolicy,
  options: { rustChanged: boolean; securityChanged: boolean },
): ExpectedWorkflows {
  validatePolicy(policy);
  const circleEntries = [
    ...policy.circleci.always,
    ...(options.rustChanged ? policy.circleci.rust_changed : policy.circleci.rust_unchanged),
  ];
  const githubEntries = [
    ...policy.github_actions.always,
    ...(options.securityChanged ? policy.github_actions.security_changed : []),
  ];

  return {
    circleci: circleEntries
      .filter((entry): entry is CirclePolicyEntry & { workflow: string } => entry.workflow !== undefined)
      .map((entry) => ({ name: entry.workflow, requiredJobs: entry.required_jobs ?? [] })),
    github: githubEntries.map((entry) => ({ name: entry.workflow, path: entry.path })),
  };
}

function timestamp(value: string | undefined): number {
  if (!value) return 0;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}

export function selectLatestCircleWorkflow(
  workflows: CircleWorkflow[],
  name: string,
): CircleWorkflow | undefined {
  return workflows
    .filter((workflow) => workflow.name === name)
    .sort((left, right) => timestamp(right.created_at) - timestamp(left.created_at) || right.id.localeCompare(left.id))[0];
}

export function selectLatestGithubRun(
  runs: GithubWorkflowRun[],
  expected: ExpectedGithubWorkflow,
  options: { sha: string; branch: string; notBeforeMs: number },
): GithubWorkflowRun | undefined {
  return runs
    .filter(
      (run) =>
        run.name === expected.name &&
        run.path === expected.path &&
        run.event === "push" &&
        run.head_sha === options.sha &&
        run.head_branch === options.branch &&
        timestamp(run.created_at) >= options.notBeforeMs,
    )
    .sort(
      (left, right) =>
        (right.run_attempt ?? 0) - (left.run_attempt ?? 0) ||
        timestamp(right.updated_at ?? right.created_at) - timestamp(left.updated_at ?? left.created_at) ||
        right.id - left.id,
    )[0];
}

const CIRCLE_TERMINAL_STATUSES = new Set([
  "success",
  "failed",
  "canceled",
  "error",
  "unauthorized",
  "not_run",
]);

function circleWorkflowIsTerminal(workflow: CircleWorkflow): boolean {
  return workflow.stopped_at != null || CIRCLE_TERMINAL_STATUSES.has(workflow.status);
}

function circleWorkflowUrl(id: string): string {
  return `https://app.circleci.com/workflow/${id}`;
}

export class GateInspector {
  constructor(
    private readonly circle: CircleApi,
    private readonly github: GithubApi,
    private readonly expected: ExpectedWorkflows,
    private readonly options: {
      pipelineId: string;
      sha: string;
      branch: string;
      githubNotBeforeMs: number;
    },
  ) {}

  async inspect(): Promise<GateSnapshot> {
    const [circleWorkflows, githubRuns] = await Promise.all([
      this.circle.listPipelineWorkflows(this.options.pipelineId),
      this.github.listWorkflowRuns(this.options.sha),
    ]);

    const circleStates = await Promise.all(
      this.expected.circleci.map(async (expected): Promise<MemberState> => {
        const workflow = selectLatestCircleWorkflow(circleWorkflows, expected.name);
        if (!workflow) {
          return {
            provider: "CircleCI",
            name: expected.name,
            result: "pending",
            status: "missing",
          };
        }

        const url = circleWorkflowUrl(workflow.id);
        if (!circleWorkflowIsTerminal(workflow)) {
          return {
            provider: "CircleCI",
            name: expected.name,
            result: "pending",
            status: workflow.status,
            url,
          };
        }
        if (workflow.status !== "success") {
          return {
            provider: "CircleCI",
            name: expected.name,
            result: "failure",
            status: workflow.status,
            url,
          };
        }

        if (expected.requiredJobs.length === 0) {
          return {
            provider: "CircleCI",
            name: expected.name,
            result: "success",
            status: workflow.status,
            url,
          };
        }

        const jobs = await this.circle.listWorkflowJobs(workflow.id);
        const jobDetails: string[] = [];
        let jobsSuccessful = true;
        for (const requiredJob of expected.requiredJobs) {
          const candidates = jobs
            .filter((job) => job.name === requiredJob)
            .sort(
              (left, right) =>
                timestamp(right.started_at) - timestamp(left.started_at) ||
                (right.id ?? "").localeCompare(left.id ?? ""),
            );
          const job = candidates[0];
          const status = job?.status ?? "missing";
          jobDetails.push(`${requiredJob}=${status}`);
          if (status !== "success") jobsSuccessful = false;
        }

        return {
          provider: "CircleCI",
          name: expected.name,
          result: jobsSuccessful ? "success" : "failure",
          status: workflow.status,
          details: jobDetails.join(", "),
          url,
        };
      }),
    );

    const githubStates = this.expected.github.map((expected): MemberState => {
      const run = selectLatestGithubRun(githubRuns, expected, {
        sha: this.options.sha,
        branch: this.options.branch,
        notBeforeMs: this.options.githubNotBeforeMs,
      });
      if (!run) {
        return {
          provider: "GitHub Actions",
          name: expected.name,
          result: "pending",
          status: "missing",
          details: expected.path,
        };
      }
      if (run.status !== "completed") {
        return {
          provider: "GitHub Actions",
          name: expected.name,
          result: "pending",
          status: run.status,
          details: `${expected.path}, attempt ${run.run_attempt ?? 1}`,
          url: run.html_url,
        };
      }
      const successful = run.conclusion === "success";
      return {
        provider: "GitHub Actions",
        name: expected.name,
        result: successful ? "success" : "failure",
        status: `${run.status}/${run.conclusion ?? "no conclusion"}`,
        details: `${expected.path}, attempt ${run.run_attempt ?? 1}`,
        url: run.html_url,
      };
    });

    const members = [...circleStates, ...githubStates];
    return {
      members,
      allTerminal: members.every((member) => member.result !== "pending"),
      allSuccessful: members.every((member) => member.result === "success"),
    };
  }
}

export function formatSnapshot(snapshot: GateSnapshot): string {
  const lines = [`[${new Date().toISOString()}] Post-merge gate status:`];
  for (const member of snapshot.members) {
    const details = member.details ? ` (${member.details})` : "";
    const url = member.url ? ` ${member.url}` : "";
    lines.push(`  [${member.result}] ${member.provider} / ${member.name}: ${member.status}${details}${url}`);
  }
  return lines.join("\n");
}

export async function waitForGate(options: {
  inspect: () => Promise<GateSnapshot>;
  timeoutMs: number;
  pollIntervalMs: number;
  now?: () => number;
  sleep?: (milliseconds: number) => Promise<void>;
  report?: (snapshot: GateSnapshot) => void;
}): Promise<GateSnapshot> {
  const now = options.now ?? Date.now;
  const sleep = options.sleep ?? ((milliseconds: number) => Bun.sleep(milliseconds));
  const report = options.report ?? ((snapshot: GateSnapshot) => console.log(formatSnapshot(snapshot)));
  const deadline = now() + options.timeoutMs;
  let latest: GateSnapshot | undefined;

  while (true) {
    latest = await options.inspect();
    report(latest);
    if (latest.allTerminal) {
      if (latest.allSuccessful) return latest;
      const failures = latest.members
        .filter((member) => member.result === "failure")
        .map((member) => `${member.provider} / ${member.name}: ${member.status}`)
        .join("; ");
      throw new Error(`post-merge validation failed: ${failures}`);
    }

    const remaining = deadline - now();
    if (remaining <= 0) {
      const nonSuccessful = latest.members
        .filter((member) => member.result !== "success")
        .map((member) => `${member.provider} / ${member.name}: ${member.status}`)
        .join("; ");
      throw new Error(`post-merge gate timed out: ${nonSuccessful}`);
    }
    await sleep(Math.min(options.pollIntervalMs, remaining));
  }
}

async function fetchJsonWithRetry<T>(
  url: string,
  headers: Record<string, string>,
  sleep: (milliseconds: number) => Promise<void> = (milliseconds) => Bun.sleep(milliseconds),
): Promise<T> {
  let lastError: Error | undefined;
  for (let attempt = 1; attempt <= MAX_API_ATTEMPTS; attempt++) {
    const response = await fetch(url, { headers }).then(
      (result) => result,
      (error) => {
        lastError = error instanceof Error ? error : new Error(String(error));
        return undefined;
      },
    );
    if (response?.ok) return (await response.json()) as T;
    if (response) {
      const body = (await response.text()).slice(0, 500);
      lastError = new Error(`HTTP ${response.status} from ${url}: ${body}`);
    }
    if (attempt < MAX_API_ATTEMPTS) await sleep(1000 * 2 ** (attempt - 1));
  }
  throw new Error(`API request failed after ${MAX_API_ATTEMPTS} attempts: ${lastError?.message ?? url}`);
}

export class CircleHttpApi implements CircleApi {
  private readonly headers: Record<string, string>;

  constructor(
    token: string,
    private readonly apiBase = DEFAULT_CIRCLECI_API_BASE,
  ) {
    this.headers = { "Circle-Token": token, Accept: "application/json" };
  }

  getPipeline(id: string): Promise<CirclePipeline> {
    return fetchJsonWithRetry(`${this.apiBase}/pipeline/${encodeURIComponent(id)}`, this.headers);
  }

  listPipelineWorkflows(id: string): Promise<CircleWorkflow[]> {
    return this.listCirclePages<CircleWorkflow>(`${this.apiBase}/pipeline/${encodeURIComponent(id)}/workflow`);
  }

  listWorkflowJobs(id: string): Promise<CircleJob[]> {
    return this.listCirclePages<CircleJob>(`${this.apiBase}/workflow/${encodeURIComponent(id)}/job`);
  }

  private async listCirclePages<T>(baseUrl: string): Promise<T[]> {
    const items: T[] = [];
    let pageToken = "";
    for (let page = 1; page <= MAX_API_PAGES; page++) {
      const url = new URL(baseUrl);
      if (pageToken) url.searchParams.set("page-token", pageToken);
      const response = await fetchJsonWithRetry<{ items?: T[]; next_page_token?: string | null }>(
        url.toString(),
        this.headers,
      );
      items.push(...(response.items ?? []));
      pageToken = response.next_page_token ?? "";
      if (!pageToken) return items;
    }
    throw new Error(`CircleCI API exceeded ${MAX_API_PAGES} pages for ${baseUrl}`);
  }
}

export class GithubHttpApi implements GithubApi {
  private readonly headers: Record<string, string>;

  constructor(
    token: string,
    private readonly owner: string,
    private readonly repo: string,
    private readonly branch: string,
    private readonly apiBase = DEFAULT_GITHUB_API_BASE,
  ) {
    this.headers = {
      Accept: "application/vnd.github+json",
      Authorization: `Bearer ${token}`,
      "X-GitHub-Api-Version": "2022-11-28",
      "User-Agent": "optimism-post-merge-gate",
    };
  }

  async listWorkflowRuns(sha: string): Promise<GithubWorkflowRun[]> {
    const runs: GithubWorkflowRun[] = [];
    for (let page = 1; page <= MAX_API_PAGES; page++) {
      const url = new URL(`${this.apiBase}/repos/${encodeURIComponent(this.owner)}/${encodeURIComponent(this.repo)}/actions/runs`);
      url.searchParams.set("head_sha", sha);
      url.searchParams.set("branch", this.branch);
      url.searchParams.set("event", "push");
      url.searchParams.set("per_page", "100");
      url.searchParams.set("page", String(page));
      const response = await fetchJsonWithRetry<{ workflow_runs?: GithubWorkflowRun[] }>(url.toString(), this.headers);
      const pageRuns = response.workflow_runs ?? [];
      runs.push(...pageRuns);
      if (pageRuns.length < 100) return runs;
    }
    throw new Error(`GitHub Actions API exceeded ${MAX_API_PAGES} pages for ${sha}`);
  }
}

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function positiveNumberEnv(name: string, defaultValue: number): number {
  const raw = process.env[name];
  if (raw === undefined || raw === "") return defaultValue;
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`${name} must be a positive number; got ${JSON.stringify(raw)}`);
  }
  return parsed;
}

export function validatePipeline(pipeline: CirclePipeline, expected: { id: string; sha: string; branch: string }): void {
  if (pipeline.id !== expected.id) {
    throw new Error(`CircleCI pipeline ID mismatch: expected ${expected.id}, got ${pipeline.id}`);
  }
  if (pipeline.vcs?.revision !== expected.sha) {
    throw new Error(`CircleCI pipeline SHA mismatch: expected ${expected.sha}, got ${pipeline.vcs?.revision ?? "missing"}`);
  }
  if (pipeline.vcs?.branch !== expected.branch) {
    throw new Error(`CircleCI pipeline branch mismatch: expected ${expected.branch}, got ${pipeline.vcs?.branch ?? "missing"}`);
  }
  if (pipeline.trigger?.type !== "webhook") {
    throw new Error(`CircleCI pipeline trigger must be webhook, got ${pipeline.trigger?.type ?? "missing"}`);
  }
}

async function main(): Promise<void> {
  const pipelineId = requiredEnv("CIRCLE_PIPELINE_ID");
  const sha = requiredEnv("CIRCLE_SHA1");
  const branch = requiredEnv("CIRCLE_BRANCH");
  if (branch !== "develop") throw new Error(`post-merge-gate only supports develop, got ${branch}`);

  const circleToken = requiredEnv("CIRCLE_API_TOKEN");
  const githubToken = process.env.GITHUB_ACCESS_TOKEN ?? process.env.GH_TOKEN;
  if (!githubToken) throw new Error("GITHUB_ACCESS_TOKEN (or GH_TOKEN) is required");
  const owner = requiredEnv("CIRCLE_PROJECT_USERNAME");
  const repo = requiredEnv("CIRCLE_PROJECT_REPONAME");

  const rustChanged = parseBoolean(process.env.POST_MERGE_RUST_CHANGED, "POST_MERGE_RUST_CHANGED");
  const securityChanged = parseBoolean(process.env.POST_MERGE_SECURITY_CHANGED, "POST_MERGE_SECURITY_CHANGED");
  const policyPath = process.env.POST_MERGE_GATE_POLICY ?? ".circleci/routing.yml";
  const policy = loadPolicy(policyPath);
  const expected = buildExpectedWorkflows(policy, { rustChanged, securityChanged });

  const circle = new CircleHttpApi(circleToken, process.env.CIRCLECI_API_BASE);
  const github = new GithubHttpApi(githubToken, owner, repo, branch, process.env.GITHUB_API_BASE);
  const pipeline = await circle.getPipeline(pipelineId);
  validatePipeline(pipeline, { id: pipelineId, sha, branch });

  const receivedAt = timestamp(pipeline.trigger?.received_at);
  if (receivedAt === 0) throw new Error("CircleCI webhook pipeline is missing trigger.received_at");
  // Allow for webhook delivery skew while excluding older push runs if the same
  // commit is pushed to develop more than once.
  const githubNotBeforeMs = receivedAt - 5 * 60 * 1000;

  console.log(`Post-merge gate for ${owner}/${repo}@${sha}`);
  console.log(`CircleCI pipeline: ${pipelineId}`);
  console.log(`Conditional routing: rust_changed=${rustChanged}, security_changed=${securityChanged}`);
  console.log(`Expected CircleCI workflows: ${expected.circleci.map((workflow) => workflow.name).join(", ")}`);
  console.log(`Expected GitHub Actions workflows: ${expected.github.map((workflow) => workflow.name).join(", ")}`);

  const inspector = new GateInspector(circle, github, expected, {
    pipelineId,
    sha,
    branch,
    githubNotBeforeMs,
  });
  await waitForGate({
    inspect: () => inspector.inspect(),
    timeoutMs: positiveNumberEnv("POST_MERGE_GATE_TIMEOUT_SECONDS", DEFAULT_TIMEOUT_SECONDS) * 1000,
    pollIntervalMs: positiveNumberEnv("POST_MERGE_GATE_POLL_INTERVAL_SECONDS", DEFAULT_POLL_INTERVAL_SECONDS) * 1000,
  });
  console.log("All expected post-merge workflows succeeded.");
}

if (import.meta.main) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.stack ?? error.message : error);
    process.exit(1);
  });
}
