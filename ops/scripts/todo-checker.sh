#!/bin/bash

set -uo pipefail

# Flags
FAIL_INVALID_FMT=false
VERBOSE=false
CHECK_CLOSED=false

# Github API access token (Optional - necessary for private repositories.)
GH_API_TOKEN="${CI_TODO_CHECKER_PAT:-""}"
AUTH=""
if [[ $GH_API_TOKEN != "" ]]; then
    AUTH="Authorization: token $GH_API_TOKEN"
fi

# Default org and repo
ORG="ethereum-optimism"
REPO="optimism"

# Counter for issues that were not found and issues that are still open.
NOT_FOUND_COUNT=0
MISMATCH_COUNT=0
OPEN_COUNT=0
CLOSED_COUNT=0
declare -a OPEN_ISSUES
declare -a CLOSED_ISSUES

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
GREY='\033[1;30m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Parse flags
#
# `--strict`: Toggle strict mode; Will fail if any TODOs are found that don't match the expected
# `--verbose`: Toggle verbose mode; Will print out details about each TODO
# `--check-closed`: Check for closed issues and error out if found
for arg in "$@"; do
  case $arg in
    --strict)
    FAIL_INVALID_FMT=true
    VERBOSE=true
    shift
    ;;
    --verbose)
    VERBOSE=true
    shift
    ;;
    --check-closed)
    CHECK_CLOSED=true
    shift
    ;;
  esac
done

# Use ripgrep to search for the pattern in all files within the repo
todos=$(rg -o --with-filename -i -n -g '!ops/scripts/todo-checker.sh' -g '!packages/contracts-bedrock/lib' 'TODO\(([^)]+)\):? [^,;]*')

# Check each TODO comment in the repo
IFS=$'\n' # Set Internal Field Separator to newline for iteration
for todo in $todos; do
    # Extract the text inside the parenthesis
    FILE=$(echo "$todo" | awk -F':' '{print $1}')
    LINE_NUM=$(echo "$todo" | awk -F':' '{print $2}')
    ISSUE_REFERENCE=$(echo "$todo" | sed -n 's/.*TODO(\([^)]*\)).*/\1/p')

    # Parse the format of the TODO comment. There are 3 supported formats (the colon is optional in all of them):
    # * TODO(<issue_number>): <description> (Default org & repo: "ethereum-optimism/monorepo")
    # * TODO(repo#<issue_number>): <description> (Default org "ethereum-optimism")
    # * TODO(org/repo#<issue_number>): <description>
    #
    # Check if it's just a number or a number with a leading #
    if [[ $ISSUE_REFERENCE =~ ^[0-9]+$ ]] || [[ $ISSUE_REFERENCE =~ ^#([0-9]+)$ ]]; then
        REPO_FULL="$ORG/$REPO"
        ISSUE_NUM="${ISSUE_REFERENCE#\#}"  # Remove leading # if present
    # Check for org_name/repo_name#number format
    elif [[ $ISSUE_REFERENCE =~ ^([^/]+)/([^#]+)#([0-9]+)$ ]]; then
        REPO_FULL="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
        ISSUE_NUM="${BASH_REMATCH[3]}"
    # Check for repo_name#number format
    elif [[ $ISSUE_REFERENCE =~ ^([^#]+)#([0-9]+)$ ]]; then
        REPO_FULL="$ORG/${BASH_REMATCH[1]}"
        ISSUE_NUM="${BASH_REMATCH[2]}"
    else
        if $FAIL_INVALID_FMT || $VERBOSE; then
            echo -e "${YELLOW}[Warning]${NC} Invalid TODO format: $todo"
        fi
        ((MISMATCH_COUNT++))
        continue
    fi

    # Don't fetch issue status if we aren't checking for closed issues.
    if  ! $CHECK_CLOSED; then
      continue
    fi
    # Use GitHub API to fetch issue details
    GH_URL_PATH="$REPO_FULL/issues/$ISSUE_NUM"
    # Grab the status code and response as a two item array [response, status]
    RESPONSE="[$(curl -sL -w ", %{http_code}" -H "$AUTH" --request GET "https://api.github.com/repos/$GH_URL_PATH")]"
    # Split the two values out
    STATUS=$(echo "$RESPONSE" | jq -r '.[1]')
    RESPONSE=$(echo "$RESPONSE" | jq -r '.[0]')

    # Check if issue was found
    if [[ "$STATUS" == "404" ]]; then
        if [[ $VERBOSE ]]; then
            echo -e "${YELLOW}[Warning]${NC} Issue not found: ${RED}$REPO_FULL/$ISSUE_NUM${NC}"
        fi
        ((NOT_FOUND_COUNT++))
        continue
    fi
    if [[ "$STATUS" != "200" ]]; then
      echo -e "${RED}[Error]${NC} Failed to retrieve issue ${YELLOW}$ISSUE_REFERENCE${NC}"
      echo "Status: ${STATUS}"
      echo "${RESPONSE}"
      exit 1
    fi

    # Check issue state
    STATE=$(echo "$RESPONSE" | jq -r .state)
    if [[ "$STATE" == "closed" ]] && $CHECK_CLOSED; then
        TITLE=$(echo "$RESPONSE" | jq -r .title)
        echo -e "${RED}[Error]${NC} Issue #$ISSUE_NUM is closed. Please remove the TODO in ${GREEN}$FILE:$LINE_NUM${NC} referencing ${YELLOW}$ISSUE_REFERENCE${NC} (${CYAN}https://github.com/$GH_URL_PATH${NC})"
        ((CLOSED_COUNT++))
        CLOSED_ISSUES+=("$REPO_FULL #$ISSUE_NUM|$TITLE|$FILE:$LINE_NUM")
    fi

    if [[ "$STATE" == "open" ]]; then
        ((OPEN_COUNT++))
        TITLE=$(echo "$RESPONSE" | jq -r .title)
        OPEN_ISSUES+=("$REPO_FULL #$ISSUE_NUM|$TITLE|$FILE:$LINE_NUM")
    fi
done

function printIssueTitle() {
    printf "\n${PURPLE}%-40s${NC} ${GREY}|${NC} ${GREEN}%-65s${NC} ${GREY}|${NC} ${YELLOW}%-40s${NC}\n" "Repository & Issue" "Title" "Location"
    echo -e "$GREY$(printf '%0.s-' {1..41})+$(printf '%0.s-' {1..67})+$(printf '%0.s-' {1..51})$NC"
}

function printIssue() {
    issue=${1}
    REPO_ISSUE="${issue%%|*}"  # up to the first |
    REMAINING="${issue#*|}"                       # after the first |
    TITLE="${REMAINING%%|*}"                      # up to the second |
    LOC="${REMAINING#*|}"                         # after the second |

    # Truncate if necessary
    if [ ${#REPO_ISSUE} -gt 37 ]; then
        REPO_ISSUE=$(printf "%.37s..." "$REPO_ISSUE")
    fi
    if [ ${#TITLE} -gt 62 ]; then
        TITLE=$(printf "%.62s..." "$TITLE")
    fi
    # Don't truncate LOC - we always want to be able to find the to do so need the full info here even if it wraps.

    printf "${CYAN}%-40s${NC} ${GREY}|${NC} %-65s ${GREY}|${NC} ${YELLOW}%-50s${NC}\n" "$REPO_ISSUE" "$TITLE" "$LOC"
}

# Print summary
if [[ $NOT_FOUND_COUNT -gt 0 ]]; then
    echo -e "${YELLOW}[Warning]${NC} ${CYAN}$NOT_FOUND_COUNT${NC} TODOs referred to issues that were not found."
fi
if [[ $MISMATCH_COUNT -gt 0 ]]; then
    echo -e "${RED}[Error]${NC} ${CYAN}$MISMATCH_COUNT${NC} TODOs did not match the expected pattern. Run with ${RED}\`--verbose\`${NC} to show details."
    if $FAIL_INVALID_FMT; then
        exit 1
    fi
fi
if [[ $OPEN_COUNT -gt 0 ]]; then
    echo -e "${GREEN}[Info]${NC} ${CYAN}$OPEN_COUNT${NC} TODOs refer to issues that are still open."
    echo -e "${GREEN}[Info]${NC} Open issue details:"
    printIssueTitle
    for issue in "${OPEN_ISSUES[@]}"; do
        printIssue "${issue}"
    done
    echo
fi

if [[ $CLOSED_COUNT -gt 0 ]]; then
    echo -e "${RED}[Error]${NC} ${CYAN}$CLOSED_COUNT${NC} TODOs refer to issues that are closed."
    echo -e "${RED}[Error]${NC} Closed issue details:"
    printIssueTitle
    for issue in "${CLOSED_ISSUES[@]}"; do
        printIssue "${issue}"
    done
    exit 1
fi
echo -e "${GREEN}[Info]${NC} Done checking issues."
