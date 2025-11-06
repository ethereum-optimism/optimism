#!/usr/bin/env bash
set -euo pipefail

# This is a shim for kona-host used as the --cannon-kona-server value when running op-challenger in acceptance tests
# op-challenger checks that the server executable exists at startup which would prevent running any tests without
# generating the kona-host binary if we used it directly. Since most tests don't use that binary we use this shim
# to make things work for any tests that don't actually play out cannon-kona games.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/bin/kona-host" "$@"
