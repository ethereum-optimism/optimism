#!/usr/bin/env bash
# Show the hand-written parts of recently published releases, so each run starts from the
# conventions actually in use rather than from what this skill did last time.
#
# Release managers edit these notes after the skill drafts them. Those edits are the real
# house style, and they are invisible unless someone looks. Read this output before
# drafting, and when it disagrees with reference/house-style.md, the published releases
# win — then offer to update the reference doc.
#
# Prints, per release, the header (everything above `## What's Changed`) and the trailing
# image/footer lines. The PR list in between is omitted; it is not where style lives.
#
# Usage: recent-style.sh [component] [count]     (default: all components, 6 releases)
set -euo pipefail

REPO=${REPO:-ethereum-optimism/optimism}
component=${1:-}
count=${2:-6}

# Over-fetch, then filter: the component's own releases are what matter most, but a train
# shares standing notices across components, so fall back to the newest releases overall.
mapfile_tags=$(gh release list --repo "$REPO" --limit 60 \
    --exclude-drafts --exclude-pre-releases \
    --json tagName,publishedAt --jq '.[] | .tagName' 2>/dev/null || true)

if [ -z "$mapfile_tags" ]; then
    echo "could not list releases for $REPO" >&2
    exit 1
fi

if [ -n "$component" ]; then
    tags=$(printf '%s\n' "$mapfile_tags" | grep "^$component/" | head -"$count" || true)
    # A brand-new component may have no published release yet.
    if [ -z "$tags" ]; then
        echo "note: no published releases for '$component'; showing the newest overall" >&2
        tags=$(printf '%s\n' "$mapfile_tags" | head -"$count")
    fi
else
    tags=$(printf '%s\n' "$mapfile_tags" | head -"$count")
fi

printf '%s\n' "$tags" | while IFS= read -r tag; do
    [ -n "$tag" ] || continue
    echo "═══════════════════════════════════════════════════════════════════"
    echo "  $tag"
    echo "═══════════════════════════════════════════════════════════════════"
    gh release view "$tag" --repo "$REPO" --json body -q .body 2>/dev/null | awk '
        /^## What.s Changed/ { inlist = 1 }
        # The footer starts at the changelog link; everything from there is convention too.
        /^\*\*Full Changelog\*\*/ { inlist = 0 }
        !inlist { print }
        inlist && !shown { print "  [... PR list omitted ...]"; shown = 1 }
    '
    echo
done
