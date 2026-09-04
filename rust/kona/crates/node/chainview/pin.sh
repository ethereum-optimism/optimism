#!/usr/bin/env bash
# Prints the sql-to-dbsp compiler pin build.rs enforces, as one line:
#   <version> <jar file name> <sha256> <jar path>
# The JAR path honours KONA_SQL2DBSP_JAR, then XDG_CACHE_HOME/HOME, exactly like build.rs.
set -euo pipefail
build_rs="$(dirname "${BASH_SOURCE[0]}")/build.rs"
version=$(grep -oE 'FELDERA_VERSION: &str = "[0-9.]+"' "$build_rs" | grep -oE '[0-9.]+')
jar_file=$(grep -oE 'JAR_FILE: &str = "[^"]+"' "$build_rs" | sed -E 's/.*= "([^"]+)"/\1/')
sha=$(grep -oE 'JAR_SHA256: &str = "[0-9a-f]{64}"' "$build_rs" | grep -oE '[0-9a-f]{64}')
cache_dir="${XDG_CACHE_HOME:-${HOME:-/tmp}/.cache}/kona-chainview"
jar="${KONA_SQL2DBSP_JAR:-$cache_dir/$jar_file}"
echo "$version $jar_file $sha $jar"
