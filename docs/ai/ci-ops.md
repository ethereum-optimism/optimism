# CI/CD Operations

This document provides guidance for AI agents working with CI/CD operational tasks in the Optimism monorepo.

## TODO Checker Failures

The repo runs a scheduled CircleCI job every 4 hours that validates TODO comments don't reference closed GitHub issues. When this job fails, an issue needs to be reopened.

### Trigger Phrases

When you see phrases like:
- "Fix the latest TODO checker failure"
- "Resolve the TODO checker CI failure"
- "Handle the TODO checker issue"

Follow the workflow below.

### Prerequisites

- `CIRCLECI_TOKEN` environment variable (for accessing CircleCI API)
- `gh` CLI authenticated with GitHub

### Workflow

#### Step 1: Find the latest scheduled TODO checker job

```bash
LATEST_PIPELINE=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline?branch=develop" \
  -H "Circle-Token: ${CIRCLECI_TOKEN:-}" | \
  jq -r '.items[] | select(.trigger.type == "scheduled_pipeline") | {id, number, created_at} | @json' | head -1)

PIPELINE_ID=$(echo "$LATEST_PIPELINE" | jq -r '.id')
PIPELINE_NUMBER=$(echo "$LATEST_PIPELINE" | jq -r '.number')
```

#### Step 2: Get the workflow and job details

```bash
WORKFLOW_ID=$(curl -s "https://circleci.com/api/v2/pipeline/$PIPELINE_ID/workflow" \
  -H "Circle-Token: ${CIRCLECI_TOKEN:-}" | \
  jq -r '.items[] | select(.name | contains("todo")) | .id')

JOB_NUMBER=$(curl -s "https://circleci.com/api/v2/workflow/$WORKFLOW_ID/job" \
  -H "Circle-Token: ${CIRCLECI_TOKEN:-}" | \
  jq -r '.items[] | .job_number')
```

Check if the workflow status is "failed". If it's "success" or "running", inform the user there's no failure to fix or to wait for completion.

#### Step 3: Fetch the job output to find closed issues

```bash
OUTPUT_URL=$(curl -s "https://circleci.com/api/v1.1/project/gh/ethereum-optimism/optimism/$JOB_NUMBER" \
  -H "Circle-Token: ${CIRCLECI_TOKEN:-}" | \
  jq -r '.steps[] | select(.name | contains("TODO")) | .actions[0].output_url')

curl -s "$OUTPUT_URL" | jq -r '.[].message'
```

The output will show a table of closed issues. Look for the `[Error] Closed issue details:` section at the end which shows:
- Repository & Issue (e.g., "ethereum-optimism/optimism #18616")
- Issue Title
- Location (e.g., "op-acceptance-tests/tests/isthmus/preinterop/interop_readiness_test.go:106")

#### Step 4: Parse the closed issue information

Extract from the "Closed issue details" table:
- Issue number (e.g., #18616)
- File path and line number (e.g., `op-acceptance-tests/tests/isthmus/preinterop/interop_readiness_test.go:106`)
- Issue title

#### Step 5: Find who closed the issue

```bash
ISSUE_NUM="<issue_number>"
CLOSING_PR=$(gh issue view $ISSUE_NUM --json closedByPullRequestsReferences --jq '.closedByPullRequestsReferences[0].number')
PR_AUTHOR=$(gh pr view $CLOSING_PR --json author --jq '.author.login')
```

#### Step 6: Read the actual TODO line from the file

Read the file at the location specified in the error to get the exact TODO comment text.

#### Step 7: Reopen the issue with proper attribution

Format the reopening comment following this template:

```bash
gh issue reopen $ISSUE_NUM --comment "@${PR_AUTHOR} Reopening because this issue was closed but there's still a TODO/skip referencing it in the codebase.

[Brief context about what was completed vs what remains]

The [TestName] at \`<file>:<line>\` is still skipped with:

\`\`\`<language>
<actual TODO line from code>
\`\`\`

Discovered by the TODO check in CI: https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/${PIPELINE_NUMBER}/workflows/${WORKFLOW_ID}/jobs/${JOB_NUMBER}"
```

### Requirements

- **Always tag the person who closed the issue** using their GitHub handle (found via the closing PR)
- **Include the exact file location** where the TODO exists
- **Include the CircleCI job URL** for traceability
- **Read and include the actual TODO line** from the code
- **Provide context** about what was completed vs what remains (if determinable from the issue)

### Output Format

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

### TODO Comment Formats

The TODO checker validates these formats:
- `TODO(#<number>)` - references ethereum-optimism/optimism
- `TODO(<repo>#<number>)` - references ethereum-optimism/<repo>
- `TODO(<org>/<repo>#<number>)` - full reference

### Error Handling

**No CIRCLECI_TOKEN**: Ask the user to set the environment variable or provide the CircleCI job URL directly.

**Multiple closed issues**: Process each one sequentially, asking for confirmation before reopening each.

**Issue already reopened**: Check if there's already a comment about the TODO. If not, add a comment with the location.

### About the TODO Checker

The TODO checker runs via `.circleci/continue/main.yml` as a scheduled workflow named `scheduled-todo-issues`. It executes `ops/scripts/todo-checker.sh --verbose --strict --check-closed`.
