# fix-todo

Resolve TODO checker CI failures by reopening GitHub issues that still have active TODOs in the codebase.

## When to Use

Use this skill when the scheduled TODO checker CI job fails. The TODO checker validates that TODO comments in the codebase don't reference closed GitHub issues.

### Trigger Phrases

- "Fix the latest TODO checker failure"
- "Resolve the TODO checker CI failure"
- "Handle the TODO checker issue"
- "Reopen issues from TODO checker"

## Background

The repository runs a scheduled CircleCI workflow (`scheduled-todo-issues`) every 4 hours that validates TODO comments. TODO comments can reference issues in formats like:
- `TODO(#1234)` - references ethereum-optimism/optimism
- `TODO(repo#1234)` - references ethereum-optimism/repo
- `TODO(org/repo#1234)` - full reference

When an issue is closed but TODOs still reference it, the job fails and issues need to be reopened to track the remaining work.

## Prerequisites

- `gh` CLI authenticated with GitHub
- Note: CircleCI API is publicly accessible for this repository, no token required

## Workflow

### Step 1: Find the latest scheduled TODO checker job

```bash
LATEST_PIPELINE=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline?branch=develop" | \
  jq -r '.items[] | select(.trigger.type == "scheduled_pipeline") | {id, number, created_at} | @json' | head -1)

PIPELINE_ID=$(echo "$LATEST_PIPELINE" | jq -r '.id')
PIPELINE_NUMBER=$(echo "$LATEST_PIPELINE" | jq -r '.number')
```

### Step 2: Get the workflow and job details

Note: The latest scheduled pipeline may only contain a "setup" workflow. Search through recent scheduled pipelines to find one with the "scheduled-todo-issues" workflow.

```bash
# Find a pipeline with the TODO workflow
PIPELINE_WITH_TODO=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline?branch=develop" | \
  jq -r '.items[] | select(.trigger.type == "scheduled_pipeline") | .id' | while read pid; do
    workflows=$(curl -s "https://circleci.com/api/v2/pipeline/$pid/workflow" | jq -r '.items[] | .name')
    if echo "$workflows" | grep -q "scheduled-todo-issues"; then
      echo "$pid"
      break
    fi
  done)

PIPELINE_ID="$PIPELINE_WITH_TODO"
PIPELINE_NUMBER=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline?branch=develop" | \
  jq -r ".items[] | select(.id == \"$PIPELINE_ID\") | .number")

WORKFLOW_DATA=$(curl -s "https://circleci.com/api/v2/pipeline/$PIPELINE_ID/workflow" | \
  jq '.items[] | select(.name == "scheduled-todo-issues")')
WORKFLOW_ID=$(echo "$WORKFLOW_DATA" | jq -r '.id')
WORKFLOW_STATUS=$(echo "$WORKFLOW_DATA" | jq -r '.status')

JOB_NUMBER=$(curl -s "https://circleci.com/api/v2/workflow/$WORKFLOW_ID/job" | \
  jq -r '.items[] | .job_number')
```

Check if the workflow status is "failed". If it's "success" or "running", inform the user there's no failure to fix or to wait for completion.

### Step 3: Fetch the job output to find closed issues

```bash
OUTPUT_URL=$(curl -s "https://circleci.com/api/v1.1/project/gh/ethereum-optimism/optimism/$JOB_NUMBER" | \
  jq -r '.steps[] | select(.name | contains("TODO")) | .actions[0].output_url')

curl -s "$OUTPUT_URL" | jq -r '.[].message'
```

The output will show a table of closed issues. Look for the `[Error] Closed issue details:` section at the end which shows:
- Repository & Issue (e.g., "ethereum-optimism/optimism #18616")
- Issue Title
- Location (e.g., "op-acceptance-tests/tests/isthmus/preinterop/interop_readiness_test.go:106")

### Step 4: Parse the closed issue information

Extract from the "Closed issue details" table:
- Issue number (e.g., #18616)
- File path and line number (e.g., `op-acceptance-tests/tests/isthmus/preinterop/interop_readiness_test.go:106`)
- Issue title

### Step 5: Find who closed the issue

Issues can be closed via PR or directly by a user. Check the timeline to find the most recent person who closed it:

```bash
ISSUE_NUM="<issue_number>"

# Use GraphQL to get the timeline and find the most recent close event
CLOSER=$(gh api graphql -f query="
query {
  repository(owner: \"ethereum-optimism\", name: \"optimism\") {
    issue(number: $ISSUE_NUM) {
      timelineItems(last: 20, itemTypes: [CLOSED_EVENT, REOPENED_EVENT]) {
        nodes {
          ... on ClosedEvent {
            __typename
            createdAt
            actor {
              login
            }
            closer {
              __typename
            }
          }
          ... on ReopenedEvent {
            __typename
            createdAt
            actor {
              login
            }
          }
        }
      }
    }
  }
}" --jq '.data.repository.issue.timelineItems.nodes | reverse | .[] | select(.__typename == "ClosedEvent") | .actor.login' | head -1)

echo "Issue closed by: @$CLOSER"
```

This finds the most recent ClosedEvent in the timeline, which correctly handles cases where an issue was:
- Closed via PR, then reopened, then closed directly by a user
- Closed multiple times by different people

Always tag the person from the most recent close event.

### Step 6: Read the actual TODO line from the file

Read the file at the location specified in the error to get the exact TODO comment text.

### Step 7: Reopen the issue with proper attribution

Format the reopening comment following this template:

```bash
gh issue reopen $ISSUE_NUM --comment "@${CLOSER} Reopening because this issue was closed but there's still a TODO/skip referencing it in the codebase.

[Brief context about what was completed vs what remains]

The [TestName] at \`<file>:<line>\` is still skipped with:

\`\`\`<language>
<actual TODO line from code>
\`\`\`

Discovered by the TODO check in CI: https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/${PIPELINE_NUMBER}/workflows/${WORKFLOW_ID}/jobs/${JOB_NUMBER}"
```

## Requirements

- **Always tag the person who closed the issue** using their GitHub handle (found via the most recent close event in the timeline)
- **Include the exact file location** where the TODO exists
- **Include the CircleCI job URL** for traceability
- **Read and include the actual TODO line** from the code
- **Provide context** about what was completed vs what remains (if determinable from the issue)

## Output Format

After successfully reopening, report:

```
✓ TODO checker failure resolved

Issue: #<number> - <title>
Status: Reopened
Tagged: @<username>
Location: <file>:<line>

View issue: https://github.com/ethereum-optimism/optimism/issues/<number>
CircleCI job: https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/<pipeline>/workflows/<workflow>/jobs/<job>
```

## TODO Comment Formats

The TODO checker validates these formats:
- `TODO(#<number>)` - references ethereum-optimism/optimism
- `TODO(<repo>#<number>)` - references ethereum-optimism/<repo>
- `TODO(<org>/<repo>#<number>)` - full reference

## Error Handling

**Multiple closed issues**: Process each one sequentially, asking for confirmation before reopening each.

**Issue already reopened**: Check if there's already a comment about the TODO. If not, add a comment with the location.

## About the TODO Checker

The TODO checker runs via `.circleci/continue/main.yml` as a scheduled workflow named `scheduled-todo-issues`. It executes `ops/scripts/todo-checker.sh --verbose --strict --check-closed`.
