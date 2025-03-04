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
declare -a OPEN_ISSUES

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
todos=$(rg -o --with-filename -i -n -g '!ops/scripts/todo-checker.sh' 'TODO\(([^)]+)\): [^,;]*')

# Check each TODO comment in the repo
IFS=$'\n' # Set Internal Field Separator to newline for iteration
for todo in $todos; do
    # Extract the text inside the parenthesis
    FILE=$(echo "$todo" | awk -F':' '{print $1}')
    LINE_NUM=$(echo "$todo" | awk -F':' '{print $2}')
    ISSUE_REFERENCE=$(echo "$todo" | sed -n 's/.*TODO(\([^)]*\)).*/\1/p')

    # Parse the format of the TODO comment. There are 3 supported formats:
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
            echo -e "${YELLOW}[Warning]:${NC} Invalid TODO format: $todo"
            if $FAIL_INVALID_FMT; then
                exit 1
            fi
        fi
        ((MISMATCH_COUNT++))
        continue
    fi

    # Use GitHub API to fetch issue details
    GH_URL_PATH="$REPO_FULL/issues/$ISSUE_NUM"
    RESPONSE=$(curl -sL -H "$AUTH" --request GET "https://api.github.com/repos/$GH_URL_PATH")

    # Check if issue was found
    if echo "$RESPONSE" | rg -q "Not Found"; then
        if [[ $VERBOSE ]]; then
            echo -e "${YELLOW}[Warning]:${NC} Issue not found: ${RED}$REPO_FULL/$ISSUE_NUM${NC}"
        fi
        ((NOT_FOUND_COUNT++))
        continue
    fi

    # Check issue state
    STATE=$(echo "$RESPONSE" | jq -r .state)

    if [[ "$STATE" == "closed" ]] && $CHECK_CLOSED; then
        echo -e "${RED}[Error]:${NC} Issue #$ISSUE_NUM is closed. Please remove the TODO in ${GREEN}$FILE:$LINE_NUM${NC} referencing ${YELLOW}$ISSUE_REFERENCE${NC} (${CYAN}https://github.com/$GH_URL_PATH${NC})"
        exit 1
    fi

    if [[ "$STATE" == "open" ]]; then
        ((OPEN_COUNT++))
        TITLE=$(echo "$RESPONSE" | jq -r .title)
        OPEN_ISSUES+=("$REPO_FULL/issues/$ISSUE_NUM|$TITLE|$FILE:$LINE_NUM")
    fi
done

# Print summary
if [[ $NOT_FOUND_COUNT -gt 0 ]]; then
    echo -e "${YELLOW}[Warning]:${NC} ${CYAN}$NOT_FOUND_COUNT${NC} TODOs referred to issues that were not found."
fi
if [[ $MISMATCH_COUNT -gt 0 ]]; then
    echo -e "${YELLOW}[Warning]:${NC} ${CYAN}$MISMATCH_COUNT${NC} TODOs did not match the expected pattern. Run with ${RED}\`--verbose\`${NC} to show details."
fi
if [[ $OPEN_COUNT -gt 0 ]]; then
    echo -e "${GREEN}[Info]:${NC} ${CYAN}$OPEN_COUNT${NC} TODOs refer to issues that are still open."
    echo -e "${GREEN}[Info]:${NC} Open issue details:"
    printf "\n${PURPLE}%-50s${NC} ${GREY}|${NC} ${GREEN}%-55s${NC} ${GREY}|${NC} ${YELLOW}%-30s${NC}\n" "Repository & Issue" "Title" "Location"
    echo -e "$GREY$(printf '%0.s-' {1..51})+$(printf '%0.s-' {1..57})+$(printf '%0.s-' {1..31})$NC"
    for issue in "${OPEN_ISSUES[@]}"; do
        REPO_ISSUE="https://github.com/${issue%%|*}"  # up to the first |
        REMAINING="${issue#*|}"                       # after the first |
        TITLE="${REMAINING%%|*}"                      # up to the second |
        LOC="${REMAINING#*|}"                         # after the second |

        # Truncate if necessary
        if [ ${#REPO_ISSUE} -gt 47 ]; then
            REPO_ISSUE=$(printf "%.47s..." "$REPO_ISSUE")
        fi
        if [ ${#TITLE} -gt 47 ]; then
            TITLE=$(printf "%.52s..." "$TITLE")
        fi
        if [ ${#LOC} -gt 27 ]; then
            LOC=$(printf "%.24s..." "$LOC")
        fi

        printf "${CYAN}%-50s${NC} ${GREY}|${NC} %-55s ${GREY}|${NC} ${YELLOW}%-30s${NC}\n" "$REPO_ISSUE" "$TITLE" "$LOC"
    done
fi

echo -e "${GREEN}[Info]:${NC} Done checking issues."
