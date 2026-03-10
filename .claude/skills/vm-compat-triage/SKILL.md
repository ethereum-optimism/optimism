# vm-compat-triage

Triage `analyze-op-program-client` CI failures by reviewing new syscall/opcode findings, determining reachability, and updating the baseline.

## When to Use

Use when the `analyze-op-program-client` CI job fails on a PR. The job runs `vm-compat` to detect syscalls and opcodes in `op-program` that are not supported by Cannon. New findings need human triage to determine if they are reachable at runtime.

### Trigger Phrases

- "Triage the vm-compat failure"
- "analyze-op-program-client failed"
- "vm-compat CI failure"
- "op-program compatibility test failed"

## Prerequisites

- `gh` CLI authenticated with GitHub
- `jq` available
- `vm-compat` binary (install: `mise use -g ubi:ChainSafe/vm-compat@1.1.0`, or download from GitHub releases — binary name is `analyzer-linux-arm64` / `analyzer-linux-amd64`)
- `llvm-objdump` — **Linux only** (install: `sudo apt-get install -y llvm`). Not available on macOS. On macOS, use `make run-vm-compat` in the `op-program` directory which runs the analysis inside Docker.
- The PR URL or number (ask the user if not provided)

## MIPS64 Syscall Reference

Syscall numbers in findings are Linux MIPS64 ABI (start at 5000). Use this lookup:

| Number | Name | Number | Name | Number | Name |
|--------|------|--------|------|--------|------|
| 5000 | read | 5001 | write | 5002 | open |
| 5003 | close | 5004 | stat | 5005 | fstat |
| 5006 | lstat | 5007 | poll | 5008 | lseek |
| 5009 | mmap | 5010 | mprotect | 5011 | munmap |
| 5012 | brk | 5013 | ioctl | 5014 | pread64 |
| 5015 | pwrite64 | 5016 | readv | 5017 | writev |
| 5018 | access | 5019 | pipe | 5020 | select |
| 5021 | sched_yield | 5022 | mremap | 5023 | msync |
| 5024 | mincore | 5025 | madvise | 5026 | shmget |
| 5027 | shmat | 5028 | shmctl | 5029 | dup |
| 5030 | dup2 | 5031 | pause | 5032 | nanosleep |
| 5033 | getitimer | 5034 | setitimer | 5035 | alarm |
| 5036 | getpid | 5037 | sendfile | 5038 | socket |
| 5039 | connect | 5040 | accept | 5041 | sendto |
| 5042 | recvfrom | 5043 | sendmsg | 5044 | recvmsg |
| 5045 | shutdown | 5046 | bind | 5047 | listen |
| 5048 | getsockname | 5049 | getpeername | 5050 | socketpair |
| 5051 | setsockopt | 5052 | getsockopt | 5053 | clone |
| 5054 | fork | 5055 | execve | 5056 | exit |
| 5057 | wait4 | 5058 | kill | 5059 | uname |
| 5060 | semget | 5061 | semop | 5062 | semctl |
| 5063 | shmdt | 5064 | msgget | 5065 | msgsnd |
| 5066 | msgrcv | 5067 | msgctl | 5068 | fcntl |
| 5069 | flock | 5070 | fsync | 5071 | fdatasync |
| 5072 | truncate | 5073 | ftruncate | 5074 | getdents |
| 5075 | getcwd | 5076 | chdir | 5077 | fchdir |
| 5078 | rename | 5079 | mkdir | 5080 | rmdir |
| 5081 | creat | 5082 | link | 5083 | unlink |
| 5084 | symlink | 5085 | readlink | 5086 | chmod |
| 5087 | fchmod | 5088 | chown | 5089 | fchown |
| 5090 | lchown | 5091 | umask | 5092 | gettimeofday |
| 5093 | getrlimit | 5094 | getrusage | 5095 | sysinfo |
| 5096 | times | 5097 | ptrace | 5098 | getuid |
| 5099 | syslog | 5100 | getgid | 5101 | setuid |
| 5102 | setgid | 5103 | geteuid | 5104 | getegid |
| 5105 | setpgid | 5106 | getppid | 5107 | getpgrp |
| 5108 | setsid | 5109 | setreuid | 5110 | setregid |
| 5111 | getgroups | 5112 | setgroups | 5113 | setresuid |
| 5114 | getresuid | 5115 | setresgid | 5116 | getresgid |
| 5117 | getpgid | 5118 | setfsuid | 5119 | setfsgid |
| 5120 | getsid | 5121 | capget | 5122 | capset |
| 5129 | rt_sigqueueinfo | 5130 | rt_sigsuspend | 5131 | sigaltstack |
| 5132 | utime | 5133 | mknod | 5134 | personality |
| 5135 | ustat | 5136 | statfs | 5137 | fstatfs |
| 5138 | sysfs | 5139 | getpriority | 5140 | setpriority |
| 5141 | sched_setparam | 5142 | sched_getparam | 5143 | sched_setscheduler |
| 5144 | sched_getscheduler | 5145 | sched_get_priority_max | 5146 | sched_get_priority_min |
| 5147 | sched_rr_get_interval | 5148 | mlock | 5149 | munlock |
| 5150 | mlockall | 5151 | munlockall | 5152 | vhangup |
| 5153 | pivot_root | 5154 | _sysctl | 5155 | prctl |
| 5190 | semtimedop | 5196 | fadvise64 | 5205 | epoll_create |
| 5206 | epoll_ctl | 5207 | epoll_wait | 5208 | remap_file_pages |
| 5209 | rt_sigreturn | 5210 | set_tid_address | 5211 | restart_syscall |
| 5215 | clock_gettime | 5216 | clock_getres | 5217 | clock_nanosleep |
| 5220 | exit_group | 5223 | tgkill | 5225 | openat |
| 5247 | waitid | 5248 | set_robust_list | 5249 | get_robust_list |
| 5253 | unlinkat | 5254 | renameat | 5257 | fchmodat |
| 5261 | futimesat | 5272 | utimensat | 5279 | epoll_create1 |
| 5284 | preadv | 5285 | pwritev | 5288 | prlimit64 |
| 5297 | getrandom | 5308 | mlock2 | 5316 | copy_file_range |
| 5317 | preadv2 | 5318 | pwritev2 | | |

For syscall numbers not in this table, look up the number at the MIPS64 syscall table in the Linux kernel source (`arch/mips/kernel/syscalls/syscall_n64.tbl`), or search the web for "linux mips64 syscall {number}".

## Workflow

### Step 1: Get the PR and failure data

If the user hasn't provided a PR URL, ask for it.

Extract the PR number and fetch the failing check run. The `analyze-op-program-client` job runs inside the CircleCI `main` workflow, not as a GitHub check run directly. Look it up via CircleCI API:

```bash
PR_NUM="<number>"
BRANCH=$(gh pr view "$PR_NUM" --repo ethereum-optimism/optimism --json headRefName -q '.headRefName')

# Get the latest pipeline for this branch
PIPELINE_ID=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline?branch=$BRANCH" | \
  python3 -c "import json,sys; print(json.load(sys.stdin)['items'][0]['id'])")

# Find the main workflow
WORKFLOW_ID=$(curl -s "https://circleci.com/api/v2/pipeline/$PIPELINE_ID/workflow" | \
  python3 -c "import json,sys; [print(w['id']) for w in json.load(sys.stdin)['items'] if w['name']=='main']")

# Find the failed job
JOB_NUMBER=$(curl -s "https://circleci.com/api/v2/workflow/$WORKFLOW_ID/job" | \
  python3 -c "import json,sys; [print(j['job_number']) for j in json.load(sys.stdin)['items'] if j['name']=='analyze-op-program-client']")
```

### Step 2: Get the findings

Try these methods in order. Use the first one that works.

**1. CI artifact (preferred when available).** Check if the CI job stored the findings JSON as an artifact. This is the fastest path — no local tooling needed.

```bash
# Get artifacts for the failed job
ARTIFACTS=$(curl -s "https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/$JOB_NUMBER/artifacts")

# Look for the findings JSON artifact
ARTIFACT_URL=$(echo "$ARTIFACTS" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for item in data.get('items', []):
    if 'findings' in item.get('path', '') and item['path'].endswith('.json'):
        print(item['url'])
        break
")

if [ -n "$ARTIFACT_URL" ]; then
    curl -sL "$ARTIFACT_URL" -o /tmp/vm-compat-full-findings.json
fi
```

If the artifact exists and contains findings, use it. No auth is needed for public repos.

**2. Run locally (if no artifact available).**

**On Linux** (requires `vm-compat` and `llvm-objdump` in PATH):
```bash
# Checkout the PR branch in a worktree, then from the op-program directory:
vm-compat analyze \
  --with-trace=true \
  --skip-warnings=false \
  --format=json \
  --vm-profile-config vm-profiles/cannon-multithreaded-64.yaml \
  --baseline-report compatibility-test/baseline-cannon-multithreaded-64.json \
  --report-output-path /tmp/vm-compat-full-findings.json \
  ./client/cmd/main.go
```

**On macOS** (`llvm-objdump` is not available natively — use Docker):
```bash
# From the op-program directory in the PR branch worktree:
make run-vm-compat
```
This builds and runs the analysis inside Docker. The findings JSON will be in the Docker build output (and as a CI artifact if the artifact capture PR is merged).

**Do NOT use CI log output.** CircleCI truncates large log output, which silently drops findings from the beginning of the JSON array. Triage based on incomplete data is worse than useless — it gives false confidence. If neither CI artifacts nor local execution are available, stop and tell the user to set up one of those options.

### Step 3: Load and parse findings

Parse the JSON findings. Each finding has:
- `callStack`: Nested object with `function` (and optionally `file`, `line`, `absPath`) and `callStack` fields forming the call chain. The outermost level is the leaf (syscall), innermost is `main.main`.
- `message`: e.g. "Potential Incompatible Syscall Detected: 5043"
- `severity`: "CRITICAL" or "WARNING"
- `hash`: Unique identifier

### Step 4: Load the baseline and compare

Read the baseline file (`op-program/compatibility-test/baseline-cannon-multithreaded-64.json`). Also check if the failure is for the `-next` variant.

Flatten each callStack into an ordered list of function names (ignoring `line`, `file`, `absPath`). A finding **matches** a baseline entry if the function name sequence is identical.

Mark matched findings as **existing/accepted**.

### Step 5: Present each new finding to the user

Present findings one at a time. Do NOT group or summarize multiple findings — let the user make individual decisions. When the user marks a function as unreachable, auto-resolve other findings that share that unreachable path.

#### Display format:

Always show the **full call stack from main.main to the leaf syscall**, with main.main at the top. This gives the user the execution context they need to judge reachability.

Include the **source file path** (no line number) for each function in the stack so the user can navigate to the right file. The `line` and `file` fields in the vm-compat JSON are assembly output positions, NOT Go source lines — they are useless. vm-compat does not provide Go source line numbers, and function definition lines would be misleading since we want call sites, not definitions.

To resolve file paths:

1. **Use the PR branch worktree for all lookups** — the source must match the code that produced the findings. Never look up locations from develop or another branch.
2. **Preferred: use `go_search` (gopls MCP)** to find function definitions. This resolves symbols accurately including methods on types, handles vendored/replaced modules, and works across the full dependency tree. However, gopls must be running from a Go workspace — if Claude was started from a non-Go directory (e.g., op-claude), gopls won't work and you must fall back to grep.
3. **Fallback: `grep -rn "func <name>"`** in the PR worktree for optimism code, and in `go env GOMODCACHE` for geth/dependency code (find the exact module path from the `replace` directive in go.mod).
4. For stdlib functions (syscall.*, os.*, internal/*): omit the file path.
5. Cache the lookups — many findings share the same functions.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Finding [N of TOTAL] | SEVERITY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Type: Incompatible Syscall: <number> (<name>)
  or: Incompatible Opcode: <details>

Call Stack (main → leaf):
   1. main.main
   2. client.Main                                    op-program/client/program.go
   3. client.RunProgram                              op-program/client/program.go
   ...
  14. pathdb.repairHistory                           triedb/pathdb/history.go
  15. rawdb.NewStateFreezer                          core/rawdb/ancient_scheme.go
  ...
  21. os.Remove
  22. syscall.unlinkat
```

**Display conventions:**
- Show full package paths in the raw data but use shortened names (last path component) in the display for readability
- Number every line so the user can easily reference by number
- main.main is always line 1, leaf syscall is always the last line
- Show source file path (no line number) for optimism and geth functions; omit for stdlib/runtime
- For geth functions, show paths relative to the geth module root (e.g., `core/rawdb/freezer.go`)

#### User options:

After showing each finding, ask:

- **Number (1-N)** — That line is the first unreachable point in the call stack
- **A** — Acceptable (reachable but Cannon handles it correctly)
- **?** — Needs investigation (allow follow-up questions before deciding)

#### When user provides a line number (unreachable):

Ask for the reason it's unreachable (or offer a default like "Function is not called in the Cannon execution environment").

Then scan ALL remaining unreviewed findings for any that pass through the same function at the same position in their call stack (i.e., the path from main.main to that function is identical). Mark all matches as unreachable with the same reason. Report:

```
Marked N additional findings as unreachable (same path through <function name>)
```

Proceed to the next unresolved finding.

#### When user selects "Acceptable":

Record the finding as acceptable. Proceed to the next finding.

#### When user asks clarifying questions (?):

Help the user investigate. Common queries:
- Show the source code of a function in the call stack
- Find what calls a particular function
- Check if a function is used in Cannon's execution path
- Look at the vm-profile YAML to see allowed/noop syscalls

Return to the options prompt after answering.

### Step 6: Update the baseline

Only proceed if ALL findings are marked as either unreachable or acceptable (none remaining as "needs investigation").

**IMPORTANT:** Update the baseline in the PR branch worktree — the same code that produced the findings. The baseline must match the code it will be committed with. Never update the baseline on develop or a different branch.

**Do NOT manually add entries to the existing baseline.** The baseline must be regenerated from scratch so that stale entries (from code paths that no longer exist) are removed.

To regenerate:

1. Run vm-compat with **no baseline** to get the complete report for the current code:

```bash
cd op-program && vm-compat analyze \
  --with-trace=true --skip-warnings=false --format=json \
  --vm-profile-config vm-profiles/cannon-multithreaded-64.yaml \
  --report-output-path /tmp/vm-compat-full-report.json \
  ./client/cmd/main.go
```

2. Normalize the output by stripping `line`, `file`, and `absPath` fields (these are assembly positions, not Go source lines, and cause false positives when they change):

```bash
cat /tmp/vm-compat-full-report.json | jq 'walk(
  if type == "object" and has("line") then del(.line) else . end |
  if type == "object" and has("absPath") then del(.absPath) else . end |
  if type == "object" and has("file") then del(.file) else . end
)' > op-program/compatibility-test/baseline-cannon-multithreaded-64.json
```

3. This replaces the entire baseline with the current state. The old baseline is not merged — it is replaced.

### Step 7: Verify

After regenerating the baseline, re-run `vm-compat` with the new baseline to confirm zero new findings:

```bash
cd op-program && vm-compat analyze \
  --with-trace=true --skip-warnings=false --format=json \
  --vm-profile-config vm-profiles/cannon-multithreaded-64.yaml \
  --baseline-report compatibility-test/baseline-cannon-multithreaded-64.json \
  --report-output-path /tmp/verify.json \
  ./client/cmd/main.go
```

If the output file contains an empty array `[]`, the baseline is complete.

## Notes

- The `baseline-cannon-multithreaded-64-next.json` file is for a future Cannon version. If the failure is from the `-next` variant, use that baseline instead. Ask the user if unclear.
- Findings with severity "WARNING" are typically less critical than "CRITICAL" but still need triage.
- The vm-compat tool performs static analysis — it cannot determine runtime reachability. Many flagged call paths are through library code that op-program never actually invokes.
- Common sources of unreachable code: p2p networking (geth's node package), disk I/O (freezer, database compaction), OS-level features (signals, process management).
- When a dependency upgrade (e.g., geth) changes internal call paths, many findings may have zero baseline matches even though the conceptual paths are the same. The user must still triage each one individually — do not assume they are safe just because similar paths existed before.
