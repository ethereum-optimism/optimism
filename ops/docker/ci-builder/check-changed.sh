#!/usr/bin/env bash

set -e

echoerr() { echo "$@" 1>&2; }

if [[ -n $CIRCLE_PULL_REQUEST ]]; then
	PACKAGE=$1
	GITHUB_API_URL=$(echo "https://api.github.com/repos/${CIRCLE_PULL_REQUEST:19}?access_token=$GITHUB_ACCESS_TOKEN" | sed "s/\/pull\//\/pulls\//")
	REF=$(curl -s "$GITHUB_API_URL" | jq -r ".base.ref")

	echoerr "Base Ref:     $REF"
	echoerr "Base Ref SHA: $(git show-branch --sha1-name "$REF")"
 	echoerr "Curr Ref:     $(git rev-parse --short HEAD)"

 	(git diff --dirstat=files,0 "$REF...HEAD" | sed 's/^[ 0-9.]\+% //g' | grep -q "$PACKAGE" && echo "TRUE") || echo "FALSE"
else
	echoerr "Not a PR build, requiring a total rebuild."
	echo "TRUE"
fi