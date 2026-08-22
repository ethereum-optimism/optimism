#!/usr/bin/env bash
# Annotate every PR in a git-cliff release-notes draft with what it actually touched,
# and whether that code is compiled into the component's binary.
#
# `just release-notes` filters by include-path (go.*, op-core/**, op-service/**, or all of
# rust/kona/**, rust/op-alloy/**, rust/alloy-op*/**), which is far broader than any one
# binary. But a change under op-service/ or op-core/ can absolutely change how op-batcher
# behaves — txmgr, bgpo and fees are all linked in — so "did it touch op-batcher/" is the
# wrong question. The right one is "is the changed package in this binary's transitive
# dependency set", which `go list -deps` and `cargo tree` answer exactly.
#
# Usage: pr-facts.sh <draft-file> [component]
#
# Output, one row per PR in draft order, tab-separated:
#   <tag>  #<number>  <author>  <n> files  <title>  <paths>
#
#   LINKED  changed a package compiled into the binary; <paths> lists just those
#           packages — that is the reason the PR may belong in the notes
#   DEPS    changed only the dependency manifests (go.mod/go.sum, Cargo.toml/Cargo.lock)
#   --      touched nothing the binary compiles; <paths> shows what it did touch
#   ?       no component given, or dependencies could not be resolved; <paths> shows
#           everything touched and the call is yours
#
# Resolving the Rust dependency set takes ~2 minutes, so it is cached per component under
# $TMPDIR and reused until rust/Cargo.lock changes.
set -euo pipefail

REPO=${REPO:-ethereum-optimism/optimism}
MODULE=github.com/ethereum-optimism/optimism
JOBS=8

if [ $# -lt 1 ] || [ ! -r "${1:-}" ]; then
    echo "usage: pr-facts.sh <draft-file> [component]" >&2
    exit 1
fi
draft=$1
component=${2:-}
root=$(git rev-parse --show-toplevel)

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
: > "$workdir/deps"
: > "$workdir/pkgmap"
mode=none

# --- dependency sets -------------------------------------------------------------------
# Which units of code end up in the component's binary. Empty leaves every row tagged '?'.

resolve_go() {
    for target in "./$component/cmd" "./$component/..."; do
        if (cd "$root" && go list -deps "$target" 2>/dev/null) |
            sed -n "s|^$MODULE/||p" | sort -u > "$workdir/deps" && [ -s "$workdir/deps" ]; then
            return 0
        fi
    done
    return 1
}

resolve_rust() {
    local cache="${TMPDIR:-/tmp}/pr-facts-deps-$component.txt"
    if [ ! -s "$cache" ] || [ "$root/rust/Cargo.lock" -nt "$cache" ]; then
        echo "resolving Rust dependencies for '$component' (~2 min, cached afterwards)..." >&2
        (cd "$root/rust" && cargo tree -p "$component" -e normal --prefix none 2>/dev/null) |
            awk 'NF {print $1}' | sort -u > "$cache" || return 1
    fi
    [ -s "$cache" ] || return 1
    cp "$cache" "$workdir/deps"

    # Crate names are what cargo reports, so map each workspace member's directory back to
    # its crate to classify changed file paths.
    (cd "$root/rust" && cargo metadata --no-deps --format-version 1 2>/dev/null) |
        jq -r '.packages[] | [.manifest_path, .name] | @tsv' |
        awk -F'\t' -v root="$root/" 'BEGIN { OFS = "\t" }
            { sub("^" root, "", $1); sub(/\/Cargo\.toml$/, "", $1); print $1, $2 }' > "$workdir/pkgmap"
    [ -s "$workdir/pkgmap" ]
}

case "$component" in
    '') ;;
    kona-*|op-reth|op-rbuilder|rollup-boost)
        if resolve_rust; then mode=rust; fi ;;
    *)
        if resolve_go; then mode=go; fi ;;
esac

if [ -n "$component" ] && [ "$mode" = none ]; then
    echo "warning: could not resolve dependencies for '$component'; rows will be tagged '?'" >&2
fi

# --- PR facts --------------------------------------------------------------------------

fetch() {
    local tries=0
    while [ "$tries" -lt 2 ]; do
        gh pr view "$2" --repo "$REPO" --json number,title,author,files \
            --jq '"\(.number)\t\(.author.login)\t\(.files | length)\t\(.title)", (.files[].path)' \
            > "$1" 2>/dev/null && return 0
        tries=$((tries + 1))
    done
    # Never let a failed fetch look like a PR that touched nothing: it must be judged,
    # not silently dropped.
    printf '%s\t?\t0\t(could not fetch PR %s — check by hand)\n' "$2" "$2" > "$1"
}
export -f fetch
export REPO

# Index the work items so parallel fetches still print in draft order.
grep -oE '/pull/[0-9]+' "$draft" | grep -oE '[0-9]+' | awk '!seen[$0]++' |
    awk -v d="$workdir" '{ printf "%s/pr-%04d\t%s\n", d, NR, $0 }' > "$workdir/work"

if [ ! -s "$workdir/work" ]; then
    echo "no PR references found in $draft" >&2
    exit 1
fi

# $0 and $1 are the positional parameters xargs passes to each `bash -c`, so they must
# reach the child unexpanded.
# shellcheck disable=SC2016
xargs -P "$JOBS" -n 2 bash -c 'fetch "$0" "$1"' < "$workdir/work"

for f in "$workdir"/pr-*; do
    awk -F'\t' -v depfile="$workdir/deps" -v pkgfile="$workdir/pkgmap" -v mode="$mode" '
        BEGIN {
            while ((getline dep < depfile) > 0) linked[dep] = 1
            while ((getline line < pkgfile) > 0) {
                split(line, kv, "\t")
                pkgdir[kv[1]] = kv[2]
            }
        }
        # The compilation unit a changed file belongs to: its package directory for Go,
        # its owning workspace crate (longest matching member directory) for Rust.
        function unit(path,   d, best, n, seg) {
            if (mode == "go") {
                # Test files are not compiled into the binary, so a PR that only adds
                # coverage to a linked package does not change what ships.
                if (path !~ /\.go$/ || path ~ /_test\.go$/) return ""
                d = path; sub(/\/[^\/]*$/, "", d)
                return d
            }
            best = ""
            for (d in pkgdir)
                if (index(path, d "/") == 1 && length(d) > length(best)) best = d
            return best == "" ? "" : pkgdir[best]
        }
        NR == 1 { num = $1; author = $2; count = $3; title = $4; next }
        {
            all[$0] = 1
            if ($0 ~ /^(go\.(mod|sum)|rust\/Cargo\.(toml|lock))$/) { manifest = 1; next }
            other_files++
            if (mode == "none") next
            u = unit($0)
            if (u != "" && u in linked) hits[u] = 1
        }
        END {
            # Collapse unlinked paths to two segments so the row stays readable.
            for (p in all) {
                n = split(p, seg, "/")
                shorts[(n > 1) ? seg[1] "/" seg[2] : seg[1]] = 1
            }
            if (title ~ /^\(could not fetch/)   { tag = "?" }
            else if (mode == "none")            { tag = "?";      for (p in shorts) out = out " " p }
            else if (length(hits))              { tag = "LINKED"; for (p in hits)   out = out " " p }
            else if (manifest && !other_files)  { tag = "DEPS";   out = " (manifest only)" }
            else                                { tag = "--";     for (p in shorts) out = out " " p }
            # gh returns at most 100 files, so a larger PR may hide its linked packages.
            if (count >= 100) count = count " (truncated, verify by hand)"
            printf "%s\t#%s\t%s\t%s files\t%s\t%s\n", tag, num, author, count, title, substr(out, 2)
        }
    ' "$f"
done
