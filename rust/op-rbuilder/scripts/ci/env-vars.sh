#!/bin/bash

# General information
export DATE=$(date -u -Is) # UTC
export REPO_URL="https://github.com/flashbots/op-rbuilder"
export PUBLIC_URL_HOST="https://flashbots-rbuilder-ci-stats.s3.us-east-2.amazonaws.com"
export PUBLIC_URL_REPORT="$PUBLIC_URL_HOST/${S3_UPLOAD_DIR}" # S3_UPLOAD_DIR is set in the GitHub workflow
export PUBLIC_URL_STATIC="$PUBLIC_URL_HOST/static"

# HEAD_SHA is supposed to be the commit hash of the current branch. When run in
# GitHub CI, this is actually set to the commit hash of the merge branch instead.
#
# This code gets the actual commit hash of the current branch.
#
# See also https://github.com/orgs/community/discussions/26325
export HEAD_SHA=$( git log -n 1 --pretty=format:"%H" )
if [ "$GITHUB_EVENT_NAME" == "pull_request" ]; then
    export HEAD_SHA=$(cat $GITHUB_EVENT_PATH | jq -r .pull_request.head.sha)
fi
export HEAD_SHA_SHORT=$( echo $HEAD_SHA | cut -c1-7 )

# HEAD_BRANCH is "HEAD" when run in GitHub CI. Use GITHUB_HEAD_REF instead (which is the PR branch name).
export HEAD_BRANCH=$( git branch --show-current )
if [ -z "$HEAD_BRANCH" ]; then
    export HEAD_BRANCH="${GITHUB_HEAD_REF}"
fi

# BASE_REF and BASE_SHA are the baseline (what the PR is pointing to, and what Criterion can compoare).
export BASE_REF=${GITHUB_BASE_REF:=develop} # "develop" branch is the default
export BASE_SHA=$( git log -n 1 --pretty=format:"%H" origin/${BASE_REF} ) # get hash of base branch
export BASE_SHA_SHORT=$( echo $BASE_SHA | cut -c1-7 )

#
# DEV_OVERRIDES: custom head and base sha
#
if [ -n "$X_HEAD_SHA" ]; then
    export HEAD_SHA=$X_HEAD_SHA
    export HEAD_SHA_SHORT=$( echo $HEAD_SHA | cut -c1-7 )
    export HEAD_BRANCH=""
fi

if [ -n "$X_BASE_SHA" ]; then
    export BASE_SHA=$X_BASE_SHA
    export BASE_SHA_SHORT=$( echo $BASE_SHA | cut -c1-7 )
    export BASE_REF=""
fi