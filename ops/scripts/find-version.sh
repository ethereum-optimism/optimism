#!/usr/bin/env bash
set -euo pipefail

NAME=${1:?Must specify release name}

git describe --tags --candidates=100 --match="${NAME}/*" | sed "s/${NAME}\///"
