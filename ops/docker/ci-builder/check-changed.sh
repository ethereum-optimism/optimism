#!/usr/bin/env -S bash -euET -o pipefail -O inherit_errexit

# Usage: check-changed.sh <diff-pattern>.
#
# This script compares the files changed in the <diff-pattern> to the git diff,
# and writes TRUE or FALSE to stdout if the diff matches/does not match. It is
# used by CircleCI jobs to determine if they need to run.

echoerr() { echo "$@" 1>&2; }

# Check if this is a CircleCI PR.
if [[ -z ${CIRCLE_PULL_REQUEST+x} ]]; then
	# CIRCLE_PULL_REQUEST is unbound here
	# Non-PR builds always require a rebuild.
	echoerr "Not a PR build, requiring a total rebuild."
	echo "TRUE"
else
	# CIRCLE_PULL_REQUEST is bound here
	PACKAGE=$1
	# Craft the URL to the GitHub API. The access token is optional for the monorepo since it's an open-source repo.
	GITHUB_API_URL="https://api.github.com/repos/ethereum-optimism/optimism/pulls/${CIRCLE_PULL_REQUEST/https:\/\/github.com\/ethereum-optimism\/optimism\/pull\//}"
	echoerr "GitHub URL:"
	echoerr "$GITHUB_API_URL"
	# Grab the PR's base ref using the GitHub API.
	PR=$(curl -H "Authorization: token $GITHUB_ACCESS_TOKEN" -H "Accept: application/vnd.github.v3+json" --retry 3 --retry-delay 1 -s "$GITHUB_API_URL")
	echoerr "PR data:"
	echoerr "$PR"
	REF=$(echo "$PR" | jq -r ".base.ref")

	if [ "$REF" = "master" ]; then
		echoerr "Base ref is master, requiring a total rebuild."
		echo "TRUE"
		exit 0
	fi

	if [ "$REF" = "null" ]; then
		echoerr "Bad ref, requiring a total rebuild."
		echo "TRUE"
		exit 1
	fi


	echoerr "Base Ref:     $REF"
	echoerr "Base Ref SHA: $(git show-branch --sha1-name "$REF")"
 	echoerr "Curr Ref:     $(git rev-parse --short HEAD)"

 	DIFF=$(git diff --dirstat=files,0 "$REF...HEAD")

 	# Compare HEAD to the PR's base ref, stripping out the change percentages that come with git diff --dirstat.
 	# Pass in the diff pattern to grep, and echo TRUE if there's a match. False otherwise.
 	(echo "$DIFF" | sed 's/^[ 0-9.]\+% //g' | grep -q -E "$PACKAGE" && echo "TRUE") || echo "FALSE"
fi
