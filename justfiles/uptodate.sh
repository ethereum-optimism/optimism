#!/usr/bin/env bash
# Check if a target file is up to date relative to source files/directories.
# Exits 0 (true) if the target exists and is newer than all sources.
# Exits 1 (false) if the target needs rebuilding.
#
# Usage: uptodate.sh TARGET SOURCE [SOURCE...]
#
# SOURCE can be a file or a directory. Directories are searched recursively.
#
# Example in a justfile:
#   build:
#       #!/usr/bin/env bash
#       set -euo pipefail
#       if ../justfiles/uptodate.sh bin/mybinary src/ config.yml; then
#           echo "bin/mybinary is up to date"
#           exit 0
#       fi
#       go build -o bin/mybinary .

set -euo pipefail

if [ $# -lt 2 ]; then
    echo "usage: uptodate.sh TARGET SOURCE [SOURCE...]" >&2
    exit 1
fi

target="$1"
shift

if [ ! -f "$target" ]; then
    exit 1
fi

for src in "$@"; do
    if [ ! -e "$src" ]; then
        echo "warning: source '$src' does not exist, forcing rebuild" >&2
        exit 1
    elif [ -d "$src" ]; then
        newer=$(find "$src" -type f -newer "$target" -print -quit 2>/dev/null)
        if [ -n "$newer" ]; then
            exit 1
        fi
    else
        if [ "$src" -nt "$target" ]; then
            exit 1
        fi
    fi
done

exit 0
