#!/bin/bash
# split-test-logs.sh — Split a gotestsum JSON log file into per-test log files.
#
# Usage: split-test-logs.sh <jsonfile>
#
# Reads a gotestsum --jsonfile output and writes one file per test containing
# that test's output lines. Output goes to a "per-test" sibling directory next
# to the JSON file, organized as <package>/<TestName>.log.
#
# Fails if the jsonfile doesn't exist or python3 is not available.

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <jsonfile>" >&2
  exit 1
fi

json_file="$1"

if [ ! -f "$json_file" ]; then
  echo "Error: JSON log file not found: $json_file" >&2
  exit 1
fi

if ! command -v python3 &>/dev/null; then
  echo "Error: python3 is required but not found" >&2
  exit 1
fi

output_dir="$(dirname "$json_file")/per-test"

python3 - "$json_file" "$output_dir" <<'PYEOF'
import json, os, sys, re

json_file = sys.argv[1]
output_dir = sys.argv[2]

handles = {}

def get_handle(path):
    if path not in handles:
        os.makedirs(os.path.dirname(path), exist_ok=True)
        handles[path] = open(path, "w")
    return handles[path]

def sanitize(name):
    return re.sub(r'[<>:"|?*]', '_', name)

count = 0
with open(json_file) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue

        test = ev.get("Test")
        action = ev.get("Action")
        output = ev.get("Output")
        package = ev.get("Package", "")

        if not test or action != "output" or output is None:
            continue

        pkg_dir = sanitize(package.replace("/", "."))
        test_name = sanitize(test)
        file_path = os.path.join(output_dir, pkg_dir, f"{test_name}.log")

        fh = get_handle(file_path)
        fh.write(output)
        count += 1

for fh in handles.values():
    fh.close()

print(f"Split {count} output lines across {len(handles)} test files in {output_dir}")
PYEOF
