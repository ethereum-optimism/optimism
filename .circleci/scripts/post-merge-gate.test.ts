import { afterEach, describe, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import {
  buildExpectedWorkflows,
  GateInspector,
  loadPolicy,
  parseBoolean,
  selectLatestCircleWorkflow,
  selectLatestGithubRun,
  validatePipeline,
  validatePolicy,
  waitForGate,
  type CircleApi,
  type CircleJob,
  type CirclePipeline,
  type CircleWorkflow,
  type ExpectedWorkflows,
  type GateSnapshot,
  type GithubApi,
  type GithubWorkflowRun,
  type PostMergePolicy,
} from "./post-merge-gate";

const repoRoot = path.resolve(import.meta.dir, "../..");
const routingPath = path.join(repoRoot, ".circleci/routing.yml");
const continuationDir = path.join(repoRoot, ".circleci/continue");
const workflowsDir = path.join(repoRoot, ".github/workflows");
const tempDirs: string[] = [];

afterEach(() => {
  while (tempDirs.length > 0) rmSync(tempDirs.pop()!, { recursive: true, force: true });
});

function yamlJson<T>(file: string, expression = "."): T {
  const result = Bun.spawnSync(["yq", "-o=json", expression, file], { cwd: repoRoot });
  if (!result.success) throw new Error(new TextDecoder().decode(result.stderr));
  return JSON.parse(new TextDecoder().decode(result.stdout)) as T;
}

function patternMatches(value: string, pattern: string): boolean {
  if (pattern.startsWith("/") && pattern.endsWith("/")) {
    return new RegExp(pattern.slice(1, -1)).test(value);
  }
  return value === pattern;
}

function circleJobAllowsBranch(job: string | Record<string, Record<string, unknown> | null>, branch: string): boolean {
  if (typeof job === "string") return true;
  const config = Object.values(job)[0];
  if (!config) return true;
  const filters = config.filters as
    | { branches?: { only?: string | string[]; ignore?: string | string[] } }
    | undefined;
  const branchFilters = filters?.branches;
  if (!branchFilters) return true;
  const asArray = (value: string | string[] | undefined): string[] =>
    value === undefined ? [] : Array.isArray(value) ? value : [value];
  const only = asArray(branchFilters.only);
  if (only.length > 0) return only.some((pattern) => patternMatches(branch, pattern));
  const ignore = asArray(branchFilters.ignore);
  return !ignore.some((pattern) => patternMatches(branch, pattern));
}

function fixturePolicy(): PostMergePolicy {
  return {
    circleci: {
      always: [
        { route: "main", workflow: "main", required_jobs: ["go-tests-full"] },
        { route: "gate", excluded: "self" },
      ],
      rust_changed: [{ route: "rust", workflow: "rust-ci" }],
      rust_unchanged: [{ route: "rust_skip", workflow: "rust-ci-gate-short" }],
    },
    github_actions: {
      always: [{ workflow: "build images", path: ".github/workflows/build-images.yaml" }],
      security_changed: [{ workflow: "security", path: ".github/workflows/security.yml" }],
    },
  };
}

function githubRun(overrides: Partial<GithubWorkflowRun> = {}): GithubWorkflowRun {
  return {
    id: 1,
    name: "build images",
    path: ".github/workflows/build-images.yaml",
    event: "push",
    status: "completed",
    conclusion: "success",
    head_sha: "abc",
    head_branch: "develop",
    run_attempt: 1,
    created_at: "2026-01-01T00:01:00Z",
    updated_at: "2026-01-01T00:02:00Z",
    html_url: "https://github.example/run/1",
    ...overrides,
  };
}

class FakeCircleApi implements CircleApi {
  pipeline: CirclePipeline = {
    id: "pipeline",
    trigger: { type: "webhook", received_at: "2026-01-01T00:00:00Z" },
    vcs: { revision: "abc", branch: "develop" },
  };
  workflows: CircleWorkflow[] = [];
  jobs = new Map<string, CircleJob[]>();

  async getPipeline(): Promise<CirclePipeline> {
    return this.pipeline;
  }
  async listPipelineWorkflows(): Promise<CircleWorkflow[]> {
    return this.workflows;
  }
  async listWorkflowJobs(id: string): Promise<CircleJob[]> {
    return this.jobs.get(id) ?? [];
  }
}

class FakeGithubApi implements GithubApi {
  runs: GithubWorkflowRun[] = [];
  async listWorkflowRuns(): Promise<GithubWorkflowRun[]> {
    return this.runs;
  }
}

function snapshot(results: Array<"pending" | "success" | "failure">): GateSnapshot {
  const members = results.map((result, index) => ({
    provider: "CircleCI" as const,
    name: `workflow-${index}`,
    result,
    status: result,
  }));
  return {
    members,
    allTerminal: members.every((member) => member.result !== "pending"),
    allSuccessful: members.every((member) => member.result === "success"),
  };
}

describe("policy", () => {
  test("builds exact conditional expected sets", () => {
    const policy = fixturePolicy();
    expect(buildExpectedWorkflows(policy, { rustChanged: true, securityChanged: false })).toEqual({
      circleci: [
        { name: "main", requiredJobs: ["go-tests-full"] },
        { name: "rust-ci", requiredJobs: [] },
      ],
      github: [{ name: "build images", path: ".github/workflows/build-images.yaml" }],
    });
    expect(buildExpectedWorkflows(policy, { rustChanged: false, securityChanged: true })).toEqual({
      circleci: [
        { name: "main", requiredJobs: ["go-tests-full"] },
        { name: "rust-ci-gate-short", requiredJobs: [] },
      ],
      github: [
        { name: "build images", path: ".github/workflows/build-images.yaml" },
        { name: "security", path: ".github/workflows/security.yml" },
      ],
    });
  });

  test("requires every CircleCI route to be gated or explicitly excluded", () => {
    const policy = fixturePolicy();
    policy.circleci.always.push({ route: "unclassified" });
    expect(() => validatePolicy(policy)).toThrow("exactly one of workflow or excluded");
  });

  test("parses CircleCI-rendered booleans strictly", () => {
    expect(parseBoolean("true", "x")).toBe(true);
    expect(parseBoolean("1", "x")).toBe(true);
    expect(parseBoolean("false", "x")).toBe(false);
    expect(parseBoolean("0", "x")).toBe(false);
    expect(() => parseBoolean(undefined, "x")).toThrow("x must be one of");
  });
});

describe("rerun and trigger selection", () => {
  test("selects the newest CircleCI workflow rerun", () => {
    const selected = selectLatestCircleWorkflow(
      [
        { id: "old", name: "main", status: "failed", created_at: "2026-01-01T00:00:00Z" },
        { id: "other", name: "rust-ci", status: "success", created_at: "2026-01-01T00:02:00Z" },
        { id: "new", name: "main", status: "success", created_at: "2026-01-01T00:03:00Z" },
      ],
      "main",
    );
    expect(selected?.id).toBe("new");
  });

  test("selects only the newest exact-SHA develop push run", () => {
    const expected = { name: "build images", path: ".github/workflows/build-images.yaml" };
    const selected = selectLatestGithubRun(
      [
        githubRun({ id: 2, event: "schedule", run_attempt: 9 }),
        githubRun({ id: 3, head_sha: "wrong", run_attempt: 9 }),
        githubRun({ id: 4, head_branch: "feature", run_attempt: 9 }),
        githubRun({ id: 5, created_at: "2025-12-31T23:00:00Z", run_attempt: 9 }),
        githubRun({ id: 6, run_attempt: 1 }),
        githubRun({ id: 7, run_attempt: 2, updated_at: "2026-01-01T00:03:00Z" }),
      ],
      expected,
      { sha: "abc", branch: "develop", notBeforeMs: Date.parse("2026-01-01T00:00:00Z") },
    );
    expect(selected?.id).toBe(7);
  });
});

describe("gate inspection", () => {
  const expected: ExpectedWorkflows = {
    circleci: [
      { name: "main", requiredJobs: ["go-tests-full"] },
      { name: "rust-ci-gate-short", requiredJobs: [] },
    ],
    github: [{ name: "build images", path: ".github/workflows/build-images.yaml" }],
  };

  test("accepts only successful workflows and the required full Go test job", async () => {
    const circle = new FakeCircleApi();
    circle.workflows = [
      { id: "main", name: "main", status: "success", stopped_at: "2026-01-01T00:05:00Z" },
      { id: "rust", name: "rust-ci-gate-short", status: "success", stopped_at: "2026-01-01T00:01:00Z" },
    ];
    circle.jobs.set("main", [{ name: "go-tests-full", status: "success" }]);
    const github = new FakeGithubApi();
    github.runs = [githubRun()];

    const result = await new GateInspector(circle, github, expected, {
      pipelineId: "pipeline",
      sha: "abc",
      branch: "develop",
      githubNotBeforeMs: Date.parse("2026-01-01T00:00:00Z"),
    }).inspect();

    expect(result.allTerminal).toBe(true);
    expect(result.allSuccessful).toBe(true);
    expect(result.members.find((member) => member.name === "main")?.details).toBe("go-tests-full=success");
  });

  test("fails a successful main workflow when go-tests-full never appeared", async () => {
    const circle = new FakeCircleApi();
    circle.workflows = [
      { id: "main", name: "main", status: "success", stopped_at: "2026-01-01T00:05:00Z" },
      { id: "rust", name: "rust-ci-gate-short", status: "success", stopped_at: "2026-01-01T00:01:00Z" },
    ];
    const github = new FakeGithubApi();
    github.runs = [githubRun()];

    const result = await new GateInspector(circle, github, expected, {
      pipelineId: "pipeline",
      sha: "abc",
      branch: "develop",
      githubNotBeforeMs: 0,
    }).inspect();

    expect(result.allTerminal).toBe(true);
    expect(result.allSuccessful).toBe(false);
    expect(result.members.find((member) => member.name === "main")).toMatchObject({
      result: "failure",
      details: "go-tests-full=missing",
    });
  });

  test("keeps missing, running, and failing-but-not-stopped workflows pending", async () => {
    const circle = new FakeCircleApi();
    circle.workflows = [{ id: "main", name: "main", status: "failing", stopped_at: null }];
    const github = new FakeGithubApi();

    const result = await new GateInspector(circle, github, expected, {
      pipelineId: "pipeline",
      sha: "abc",
      branch: "develop",
      githubNotBeforeMs: 0,
    }).inspect();

    expect(result.allTerminal).toBe(false);
    expect(result.members.map((member) => member.result)).toEqual(["pending", "pending", "pending"]);
  });

  test("treats canceled provider results as terminal failures", async () => {
    const circle = new FakeCircleApi();
    circle.workflows = [
      { id: "main", name: "main", status: "canceled", stopped_at: "2026-01-01T00:05:00Z" },
      { id: "rust", name: "rust-ci-gate-short", status: "success", stopped_at: "2026-01-01T00:01:00Z" },
    ];
    const github = new FakeGithubApi();
    github.runs = [githubRun({ conclusion: "cancelled" })];

    const result = await new GateInspector(circle, github, expected, {
      pipelineId: "pipeline",
      sha: "abc",
      branch: "develop",
      githubNotBeforeMs: 0,
    }).inspect();

    expect(result.allTerminal).toBe(true);
    expect(result.allSuccessful).toBe(false);
    expect(result.members.filter((member) => member.result === "failure")).toHaveLength(2);
  });
});

describe("bounded waiting", () => {
  test("waits for delayed workflows", async () => {
    let now = 0;
    let calls = 0;
    const result = await waitForGate({
      inspect: async () => (++calls === 1 ? snapshot(["pending"]) : snapshot(["success"])),
      timeoutMs: 100,
      pollIntervalMs: 10,
      now: () => now,
      sleep: async (milliseconds) => {
        now += milliseconds;
      },
      report: () => {},
    });
    expect(calls).toBe(2);
    expect(result.allSuccessful).toBe(true);
  });

  test("does not fail early while another expected workflow is pending", async () => {
    let now = 0;
    let calls = 0;
    await expect(
      waitForGate({
        inspect: async () => (++calls === 1 ? snapshot(["failure", "pending"]) : snapshot(["failure", "success"])),
        timeoutMs: 100,
        pollIntervalMs: 10,
        now: () => now,
        sleep: async (milliseconds) => {
          now += milliseconds;
        },
        report: () => {},
      }),
    ).rejects.toThrow("post-merge validation failed");
    expect(calls).toBe(2);
  });

  test("fails closed when an expected workflow never appears", async () => {
    let now = 0;
    await expect(
      waitForGate({
        inspect: async () => snapshot(["pending"]),
        timeoutMs: 25,
        pollIntervalMs: 10,
        now: () => now,
        sleep: async (milliseconds) => {
          now += milliseconds;
        },
        report: () => {},
      }),
    ).rejects.toThrow("post-merge gate timed out");
  });

  test("fails closed immediately when an API inspection exhausts retries", async () => {
    await expect(
      waitForGate({
        inspect: async () => {
          throw new Error("API request failed after 4 attempts");
        },
        timeoutMs: 100,
        pollIntervalMs: 10,
        report: () => {},
      }),
    ).rejects.toThrow("API request failed after 4 attempts");
  });
});

describe("pipeline correlation", () => {
  test("requires the exact webhook develop pipeline and SHA", () => {
    const pipeline: CirclePipeline = {
      id: "pipeline",
      trigger: { type: "webhook", received_at: "2026-01-01T00:00:00Z" },
      vcs: { revision: "abc", branch: "develop" },
    };
    expect(() => validatePipeline(pipeline, { id: "pipeline", sha: "abc", branch: "develop" })).not.toThrow();
    expect(() =>
      validatePipeline(
        { ...pipeline, trigger: { type: "scheduled_pipeline" } },
        { id: "pipeline", sha: "abc", branch: "develop" },
      ),
    ).toThrow("must be webhook");
    expect(() => validatePipeline(pipeline, { id: "pipeline", sha: "wrong", branch: "develop" })).toThrow("SHA mismatch");
  });
});

describe("repository policy drift guards", () => {
  test("develop change detection uses the pushed commit instead of an empty origin/develop diff", () => {
    const dir = mkdtempSync(path.join(tmpdir(), "post-merge-changes-"));
    tempDirs.push(dir);
    const git = (...args: string[]) => {
      const result = Bun.spawnSync(["git", ...args], { cwd: dir });
      if (!result.success) throw new Error(new TextDecoder().decode(result.stderr));
    };
    git("init", "-q");
    git("config", "user.email", "ci@example.com");
    git("config", "user.name", "CI Test");
    mkdirSync(path.join(dir, ".circleci"));
    writeFileSync(path.join(dir, ".circleci/config.yml"), "version: old\n");
    git("add", ".circleci/config.yml");
    git("commit", "-qm", "base");
    writeFileSync(path.join(dir, ".circleci/config.yml"), "version: new\n");
    git("add", ".circleci/config.yml");
    git("commit", "-qm", "change");
    // Reproduce CircleCI checkout: origin/develop already points at HEAD.
    git("update-ref", "refs/remotes/origin/develop", "HEAD");

    const output = path.join(dir, "params.json");
    writeFileSync(output, "{}");
    const result = Bun.spawnSync(["bash", path.join(repoRoot, ".circleci/scripts/collect-params.sh"), "detect"], {
      cwd: dir,
      env: {
        ...process.env,
        OUTPUT: output,
        BASE_REVISION: "develop",
        CURRENT_BRANCH: "develop",
        TRIGGER_SOURCE: "webhook",
      },
    });
    expect(new TextDecoder().decode(result.stderr)).toBe("");
    expect(result.exitCode).toBe(0);
    const params = JSON.parse(readFileSync(output, "utf8")) as Record<string, boolean>;
    expect(params["c-rust_changes_detected"]).toBe(true);
    expect(params["c-security_changed"]).toBe(true);
  });

  test("the develop router enables exactly the policy routes for both Rust variants", () => {
    const policy = loadPolicy(routingPath);
    for (const rustChanged of [true, false]) {
      const dir = mkdtempSync(path.join(tmpdir(), "post-merge-routing-"));
      tempDirs.push(dir);
      const output = path.join(dir, "params.json");
      writeFileSync(
        output,
        JSON.stringify({
          "c-rust_changes_detected": rustChanged,
          "c-security_changed": false,
        }),
      );
      const result = Bun.spawnSync(["bash", ".circleci/scripts/compute-workflow-conditions.sh"], {
        cwd: repoRoot,
        env: {
          ...process.env,
          OUTPUT: output,
          TRIGGER_SOURCE: "webhook",
          BRANCH: "develop",
          TAG: "",
          SCHEDULE_NAME: "",
        },
      });
      expect(new TextDecoder().decode(result.stderr)).toBe("");
      expect(result.exitCode).toBe(0);
      const routed = Object.entries(JSON.parse(readFileSync(output, "utf8")) as Record<string, unknown>)
        .filter(([key, value]) => key.startsWith("c-run_") && value === true)
        .map(([key]) => key.slice("c-run_".length))
        .sort();
      const expectedRoutes = [
        ...policy.circleci.always,
        ...(rustChanged ? policy.circleci.rust_changed : policy.circleci.rust_unchanged),
      ]
        .map((entry) => entry.route)
        .sort();
      expect(routed).toEqual(expectedRoutes);
    }
  });

  test("every post-merge CircleCI route exists and points at its classified workflow", () => {
    const policy = loadPolicy(routingPath);
    const files = readdirSync(continuationDir)
      .filter((file) => file.endsWith(".yml"))
      .map((file) => path.join(continuationDir, file));
    const parameters = new Set<string>();
    const workflows = new Map<string, Record<string, unknown>>();
    for (const file of files) {
      const config = yamlJson<{ parameters?: Record<string, unknown>; workflows?: Record<string, Record<string, unknown>> }>(file);
      for (const parameter of Object.keys(config.parameters ?? {})) parameters.add(parameter);
      for (const [name, workflow] of Object.entries(config.workflows ?? {})) workflows.set(name, workflow);
    }

    const entries = [
      ...policy.circleci.always,
      ...policy.circleci.rust_changed,
      ...policy.circleci.rust_unchanged,
    ];
    for (const entry of entries) {
      const parameter = `c-run_${entry.route}`;
      expect(parameters.has(parameter)).toBe(true);
      if (entry.workflow) {
        const workflow = workflows.get(entry.workflow);
        expect(workflow).toBeDefined();
        expect(workflow?.when).toBe(`<< pipeline.parameters.${parameter} >>`);
        const jobs = (workflow?.jobs ?? []) as Array<string | Record<string, Record<string, unknown> | null>>;
        const effectiveJobNames = jobs.map((job) => {
          if (typeof job === "string") return job;
          const [jobType, config] = Object.entries(job)[0];
          return (config && typeof config.name === "string" ? config.name : jobType) as string;
        });
        for (const requiredJob of entry.required_jobs ?? []) expect(effectiveJobNames).toContain(requiredJob);
      }
    }
  });

  test("continuation workflows cannot bypass the classified c-run router on develop", () => {
    const files = readdirSync(continuationDir)
      .filter((file) => file.endsWith(".yml"))
      .map((file) => path.join(continuationDir, file));
    const parameters = new Map<string, { type?: string; default?: unknown }>();
    const workflows = new Map<string, Record<string, unknown>>();
    for (const file of files) {
      const config = yamlJson<{
        parameters?: Record<string, { type?: string; default?: unknown }>;
        workflows?: Record<string, Record<string, unknown>>;
      }>(file);
      for (const [name, parameter] of Object.entries(config.parameters ?? {})) parameters.set(name, parameter);
      for (const [name, workflow] of Object.entries(config.workflows ?? {})) workflows.set(name, workflow);
    }

    for (const [name, parameter] of parameters) {
      if (!name.startsWith("c-run_")) continue;
      expect(parameter.type).toBe("boolean");
      expect(parameter.default).toBe(false);
    }

    const bypasses: string[] = [];
    for (const [name, workflow] of workflows) {
      if (workflow.when !== undefined && workflow.when !== null) {
        if (typeof workflow.when !== "string") {
          bypasses.push(`${name}: non-parameter when condition`);
          continue;
        }
        const match = workflow.when.match(/^<< pipeline\.parameters\.(c-run_[A-Za-z0-9_]+) >>$/);
        if (!match || !parameters.has(match[1])) bypasses.push(`${name}: unclassified when condition`);
        continue;
      }

      const jobs = (workflow.jobs ?? []) as Array<string | Record<string, Record<string, unknown> | null>>;
      if (jobs.some((job) => circleJobAllowsBranch(job, "develop"))) {
        bypasses.push(`${name}: unguarded job allows develop`);
      }
    }
    expect(bypasses).toEqual([]);
  });

  test("every repository GHA workflow that pushes on develop is classified", () => {
    const policy = loadPolicy(routingPath);
    const classified = new Map(
      [...policy.github_actions.always, ...policy.github_actions.security_changed].map((entry) => [entry.path, entry.workflow]),
    );
    const developPushWorkflows = new Map<string, string>();

    for (const filename of readdirSync(workflowsDir).filter((file) => file.endsWith(".yml") || file.endsWith(".yaml"))) {
      const fullPath = path.join(workflowsDir, filename);
      const config = yamlJson<{
        name?: string;
        on?: string | string[] | Record<string, unknown>;
      }>(fullPath);
      const events = config.on;
      if (!events) continue;
      let runsOnDevelop = false;
      if (events === "push" || (Array.isArray(events) && events.includes("push"))) {
        runsOnDevelop = true;
      } else if (!Array.isArray(events) && typeof events === "object" && Object.hasOwn(events, "push")) {
        const push = events.push as null | { branches?: string | string[]; tags?: string | string[] };
        if (push === null) {
          runsOnDevelop = true;
        } else {
          const branches =
            push.branches === undefined ? [] : Array.isArray(push.branches) ? push.branches : [push.branches];
          if (branches.length > 0) {
            runsOnDevelop = branches.some((pattern) => new Bun.Glob(pattern).match("develop"));
          } else {
            // Defining only tag filters disables branch pushes; otherwise an
            // omitted branches filter means all branches.
            runsOnDevelop = push.tags === undefined;
          }
        }
      }
      if (!runsOnDevelop) continue;
      const relative = path.relative(repoRoot, fullPath);
      developPushWorkflows.set(relative, config.name ?? "");
    }

    expect([...developPushWorkflows.keys()].sort()).toEqual([...classified.keys()].sort());
    for (const [workflowPath, name] of developPushWorkflows) expect(classified.get(workflowPath)).toBe(name);

    const routing = yamlJson<{ change_patterns: { any: { security_changed: string } } }>(routingPath);
    const security = yamlJson<{ on: { push: { paths: string[] } } }>(path.join(repoRoot, ".github/workflows/security.yml"));
    const securityPattern = new RegExp(routing.change_patterns.any.security_changed);
    for (const filteredPath of security.on.push.paths) expect(securityPattern.test(filteredPath)).toBe(true);
  });
});
