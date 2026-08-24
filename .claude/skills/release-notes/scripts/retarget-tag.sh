#!/usr/bin/env bash
# Point a note's own tag references at the finalized version instead of the RC.
#
# git-cliff generates a draft against the RC tag, so the heading, the right-hand side of
# the compare link and the image tag all read `vX.Y.Z-rc.N`. A finalized release publishes
# those as `vX.Y.Z`. The compare link's LEFT side stays an RC — it is the previous
# release's base, and published notes leave it alone.
#
# Refuses to rewrite unless the finalized tag exists and points at the same commit as the
# RC: retagging a note whose binary came from a different commit is worse than leaving the
# RC references visible.
#
# Usage: retarget-tag.sh <notes-file> <component> [final-version]
#        final-version defaults to the RC version with -rc.N stripped.
# Edits the file in place. Exits non-zero, leaving the file untouched, if it cannot verify.
set -euo pipefail

REPO=${REPO:-ethereum-optimism/optimism}

if [ $# -lt 2 ] || [ ! -w "${1:-}" ]; then
    echo "usage: retarget-tag.sh <notes-file> <component> [final-version]" >&2
    exit 2
fi
notes=$1
component=$2

# The highest RC of the newest version in the file is the release's own tag; any other RC
# reference is the previous release's base in the compare link.
rc=$(grep -oE "$component/v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+" "$notes" |
     sed "s|^$component/||" | sort -V | tail -1 || true)

if [ -z "$rc" ]; then
    echo "no RC references found in $notes — nothing to retarget" >&2
    exit 0
fi

final=${3:-${rc%-rc.*}}

# A missing ref makes gh print a 422 body on stdout, so accept only a real commit sha.
resolve() {
    gh api "repos/$REPO/commits/$1" --jq .sha 2>/dev/null |
        grep -xE '[0-9a-f]{40}' || true
}
rc_sha=$(resolve "$component/$rc")
final_sha=$(resolve "$component/$final")

if [ -z "$final_sha" ]; then
    cat >&2 <<EOF
WARNING: $component/$final does not exist yet — left $notes unchanged.
  The notes still advertise the RC ($rc) in the heading, the compare link and the image
  tag. Either finalize the release and re-run this script, or publish knowing that the
  body points operators at an RC image.
EOF
    exit 1
fi

if [ -n "$rc_sha" ] && [ "$rc_sha" != "$final_sha" ]; then
    cat >&2 <<EOF
WARNING: $component/$final and $component/$rc are different commits — left $notes unchanged.
  final: $final_sha
  rc:    $rc_sha
  The PR list was generated for the RC, so it may not describe what the finalized tag
  contains. Regenerate the draft against $final rather than retagging this one.
EOF
    exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

awk -v comp="$component" -v rc="$rc" -v final="$final" '
    # Literal (non-regex) replace, so version dots need no escaping.
    function rep(s, old, new,   out, i) {
        while ((i = index(s, old)) > 0) {
            out = out substr(s, 1, i - 1) new
            s = substr(s, i + length(old))
            n++
        }
        return out s
    }
    {
        line = rep($0, comp "/" rc, comp "/" final)   # headings, compare link
        line = rep(line, comp ":" rc, comp ":" final) # registry image paths
        print line
    }
    END { printf "retargeted %d reference(s): %s -> %s\n", n, rc, final > "/dev/stderr" }
' "$notes" > "$tmp"

cat "$tmp" > "$notes"
