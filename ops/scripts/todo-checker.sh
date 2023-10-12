#!/bin/bash

# Github API access token (Optional - necessary for private repositories.)
TOKEN=""
if [[ $TOKEN != "" ]]; then
    AUTH="Authorization: token $TOKEN"
fi

# Default org and repo
ORG="ethereum-optimism"
REPO="client-pod"

# Counter for issues that were not found and issues that are still open.
NOT_FOUND_COUNT=0
MISMATCH_COUNT=0
OPEN_COUNT=0
declare -a OPEN_ISSUES

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
GREY='\033[1;30m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Toggle strict mode; Will fail if any TODOs are found that don't match the expected
# formats:
# * TODO(<issue_number>): <description> (Default org & repo: "ethereum-optimism/client-pod")
# * TODO(repo#<issue_number>): <description> (Default org "ethereum-optimism")
# * TODO(org/repo#<issue_number>): <description>
for arg in "$@"; do
  case $arg in
    --strict)
    FAIL_INVALID_FMT=true
    shift
    ;;
    --verbose)
    VERBOSE=true
    shift
    ;;
  esac
done

# Use ripgrep to search for the pattern in all files within the repo
todos=$(rg -o --no-filename --no-line-number -g '!ops/scripts/todo-checker.sh' 'TODO\(([^)]+)\): [^,;]*')

# Check each TODO comment in the repo
IFS=$'\n' # Set Internal Field Separator to newline for iteration
for todo in $todos; do
    # Extract the text inside the parenthesis
    ISSUE_REFERENCE=$(echo $todo | sed -n 's/.*TODO(\([^)]*\)).*/\1/p')

    # Check if it's just a number
    if [[ $ISSUE_REFERENCE =~ ^[0-9]+$ ]]; then
        REPO_FULL="$ORG/$REPO"
        ISSUE_NUM="$ISSUE_REFERENCE"
    # Check for org_name/repo_name#number format
    elif [[ $ISSUE_REFERENCE =~ ^([^/]+)/([^#]+)#([0-9]+)$ ]]; then
        REPO_FULL="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
        ISSUE_NUM="${BASH_REMATCH[3]}"
    # Check for repo_name#number format
    elif [[ $ISSUE_REFERENCE =~ ^([^#]+)#([0-9]+)$ ]]; then
        REPO_FULL="$ORG/${BASH_REMATCH[1]}"
        ISSUE_NUM="${BASH_REMATCH[2]}"
    else
        if [[ $FAIL_INVALID_FMT || $VERBOSE ]]; then
            echo -e "$YELLOW[Warning]:$NC Invalid TODO format: $todo"
            if [[ $FAIL_INVALID_FMT ]]; then
                exit 1
            fi
        fi
        ((MISMATCH_COUNT++))
        continue
    fi

    # Use GitHub API to fetch issue details
    RESPONSE=$(curl -sL -H "$AUTH" --request GET "https://api.github.com/repos/$REPO_FULL/issues/$ISSUE_NUM")

    # Check if issue was found
    if echo "$RESPONSE" | rg -q "Not Found"; then
        if [[ $VERBOSE ]]; then
            echo -e "$YELLOW[Warning]:$NC Issue not found: $RED$REPO_FULL/$ISSUE_NUM$NC"
        fi
        ((NOT_FOUND_COUNT++))
        continue
    fi

    # Check issue state
    STATE=$(echo "$RESPONSE" | jq -r .state)

    if [[ "$STATE" == "closed" ]]; then
        echo -e "$RED[Error]:$NC Issue #$issue_num is closed. Please remove the TODO: $todo"
        exit 1
    fi

    ((OPEN_COUNT++))
    TITLE=$(echo "$RESPONSE" | jq -r .title)
    OPEN_ISSUES+=("$REPO_FULL/issues/$ISSUE_NUM|$TITLE")
done

# Print summary
if [[ $NOT_FOUND_COUNT -gt 0 ]]; then
    echo -e "$YELLOW[Warning]:$NC $NOT_FOUND_COUNT TODOs referred to issues that were not found."
fi
if [[ $MISMATCH_COUNT -gt 0 ]]; then
    echo -e "$YELLOW[Warning]:$NC $MISMATCH_COUNT TODOs did not match the expected pattern."
fi
if [[ $OPEN_COUNT -gt 0 ]]; then
    echo -e "$GREEN[Info]:$NC $OPEN_COUNT TODOs refer to issues that are still open."
    echo -e "$GREEN[Info]:$NC Open issue details:"
    printf "\n${PURPLE}%-59s${NC} ${GREY}|${NC} ${GREEN}%-75s${NC}\n" "Repository & Issue" "Title"
    echo -e "$GREY------------------------------------------------------------+---------------------------------------------------------------------------$NC"
    for issue in "${OPEN_ISSUES[@]}"; do
        REPO_ISSUE="${issue%|*}"
        TITLE="${issue#*|}"
        printf "${CYAN}%-59s${NC} ${GREY}|${NC} %-75s\n" "https://github.com/$REPO_ISSUE" "$TITLE"
    done
fi
