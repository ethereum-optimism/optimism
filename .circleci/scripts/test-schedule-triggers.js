#!/usr/bin/env bun
import { $ } from "bun";
import { readdirSync } from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dir, "../..");
const routingPath = path.join(repoRoot, ".circleci/routing.yml");
const continuationDir = path.join(repoRoot, ".circleci/continue");
const apiBase = process.env.CIRCLECI_API_BASE ?? "https://circleci.com/api/v2";
const projectSlug =
  process.env.CIRCLECI_PROJECT_SLUG ??
  `gh/${process.env.CIRCLE_PROJECT_USERNAME ?? "ethereum-optimism"}/${process.env.CIRCLE_PROJECT_REPONAME ?? "optimism"}`;
const tokenVar = process.env.CIRCLECI_API_TOKEN_ENV ?? "CIRCLE_API_TOKEN";

function sorted(values) {
  return [...new Set(values)].sort();
}

function difference(left, right) {
  const rightSet = new Set(right);
  return sorted([...left].filter((value) => !rightSet.has(value)));
}

function printList(title, values) {
  console.log(title);
  for (const value of sorted(values)) {
    console.log(`  - ${value}`);
  }
}

function failWithList(message, values) {
  console.error(`ERROR: ${message}`);
  for (const value of sorted(values)) {
    console.error(`  - ${value}`);
  }
  process.exit(1);
}

async function yqText(expression, file) {
  return await $`yq -r ${expression} ${file}`.text();
}

// routing.yml is the source of truth for schedule -> workflow lists.
async function readScheduleMappings() {
  const output = await yqText('.schedules | to_entries[] | .key + " " + (.value | join(" "))', routingPath);
  const mappings = [];
  for (const line of output.split("\n")) {
    const [scheduleName, ...workflows] = line.trim().split(/\s+/).filter(Boolean);
    if (scheduleName === undefined) {
      continue;
    }
    mappings.push({ scheduleName, workflows });
  }
  return mappings;
}

async function extractContinuationParams() {
  const files = readdirSync(continuationDir)
    .filter((file) => file.endsWith(".yml"))
    .map((file) => path.join(continuationDir, file));
  const params = new Set();
  const expression = '.workflows | to_entries[] | select(.value.when | type == "!!str") | .value.when';

  for (const file of files) {
    const output = await yqText(expression, file);
    for (const line of output.split("\n")) {
      const match = line.match(/^<<\s*pipeline\.parameters\.(c-run_[A-Za-z0-9_]+)\s*>>$/);
      if (match) {
        params.add(match[1]);
      }
    }
  }

  return sorted(params);
}

async function fetchJsonWithRetry(url, token) {
  const maxAttempts = 4;
  let lastError;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const response = await fetch(url, {
      headers: {
        "Circle-Token": token,
      },
    }).then(
      (result) => result,
      (error) => {
        lastError = error;
        return undefined;
      },
    );

    if (response?.ok) {
      return await response.json();
    }

    if (response) {
      lastError = new Error(`HTTP ${response.status}: ${await response.text()}`);
    }

    if (attempt < maxAttempts) {
      await Bun.sleep(2000);
    }
  }

  throw lastError;
}

async function fetchScheduleNames(token) {
  const maxPages = 10;
  const schedules = new Set();
  let pageToken = "";

  for (let pageNumber = 1; pageNumber <= maxPages; pageNumber++) {
    const url = new URL(`${apiBase}/project/${projectSlug}/schedule`);
    if (pageToken !== "") {
      url.searchParams.set("page-token", pageToken);
    }

    const page = await fetchJsonWithRetry(url.toString(), token);
    for (const item of page.items ?? []) {
      if (typeof item.name === "string") {
        schedules.add(item.name);
      }
    }

    pageToken = page.next_page_token ?? "";
    if (pageToken === "") {
      return sorted(schedules);
    }
  }

  console.error(`ERROR: CircleCI schedule API returned more than ${maxPages} pages`);
  process.exit(1);
}
const mappings = await readScheduleMappings();
if (mappings.length === 0) {
  console.error(`ERROR: no schedules defined in ${routingPath}`);
  process.exit(1);
}

const configuredSchedules = sorted(mappings.map((mapping) => mapping.scheduleName));
const mappedParams = sorted(mappings.flatMap((mapping) => mapping.workflows.map((workflow) => `c-run_${workflow}`)));
const allContinuationParams = await extractContinuationParams();
const scheduledContinuationParams = sorted(allContinuationParams.filter((param) => param.startsWith("c-run_scheduled_")));

printList("Configured CircleCI schedule names:", configuredSchedules);
printList("Scheduled continuation workflow params:", scheduledContinuationParams);

const unknownMappedParams = difference(mappedParams, allContinuationParams);
if (unknownMappedParams.length > 0) {
  failWithList("scheduled_pipeline enables params that do not gate any continuation workflow:", unknownMappedParams);
}

const unmappedScheduledParams = difference(scheduledContinuationParams, mappedParams);
if (unmappedScheduledParams.length > 0) {
  failWithList("scheduled continuation workflows are not enabled by any configured schedule:", unmappedScheduledParams);
}

const token = process.env[tokenVar];
if (!token) {
  console.error(`ERROR: ${tokenVar} is not set; cannot query CircleCI schedule triggers`);
  process.exit(1);
}

const actualSchedules = await fetchScheduleNames(token);
printList(`CircleCI schedule names for ${projectSlug}:`, actualSchedules);

const missingSchedules = difference(configuredSchedules, actualSchedules);
if (missingSchedules.length > 0) {
  failWithList("configured schedule names do not exist in CircleCI:", missingSchedules);
}

console.log("Live CircleCI schedule trigger checks passed.");
