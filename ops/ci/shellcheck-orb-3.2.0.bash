#!/bin/bash
Run_ShellCheck() {
    input="$1"
    set --
    
    if [ -n "$SC_PARAM_EXCLUDE" ]; then
        set -- "$@" "--exclude=$SC_PARAM_EXCLUDE"
    fi
    
    if [ "$SC_PARAM_EXTERNAL_SOURCES" == "1" ]; then
        set -- "$@" "--external-sources"
    fi
    
    if [ -n "$SC_PARAM_SHELL" ]; then
        set -- "$@" "--shell=$SC_PARAM_SHELL"
    fi
    
    shellcheck "$@" --severity="$SC_PARAM_SEVERITY" --format="$SC_PARAM_FORMAT" "$input" >> "$SC_PARAM_OUTPUT"
}

Check_For_ShellCheck() {
    if ! command -v shellcheck &> /dev/null
    then
        echo "Shellcheck not installed"
        exit 1
    fi
}

ShellCheck_Files() {
    IFS=$'\n'
    for file in ${SC_PARAM_IGNORE_DIRS}; do
        trimmed=$(echo "${file}" | awk '{$1=$1};1')
        
        if [ -e "${trimmed}" ]; then
            set -- "$@" "!" "-path" "${trimmed}/*.sh"
        fi
    done
    
    SC_PARAM_PATTERN="${SC_PARAM_PATTERN:-"*.sh"}"
    SC_INPUT_FILES=/tmp/sc-input-files
    find "$SC_PARAM_DIR" ! -name "$(printf "*\n*")" -name "$SC_PARAM_PATTERN" -type f "$@" | tee "${SC_INPUT_FILES}"
    set +e
    while IFS= read -r script
    do
        Run_ShellCheck "$script"
    done < "${SC_INPUT_FILES}"
    set -eo pipefail
}

Catch_SC_Errors() {
    if [ -s "$SC_PARAM_OUTPUT" ]; then
        printf '\e[1;31mShellCheck Errors Found\e[0m\n'
        cat "$SC_PARAM_OUTPUT"
        exit 1
    else
        printf '\e[1;32mNo ShellCheck Errors Found\e[0m\n'
    fi
}

SC_Main() {
    Check_For_ShellCheck
    ShellCheck_Files
    Catch_SC_Errors
    rm /tmp/sc-input-files
}

# Will not run if sourced for bats.
# View src/tests for more information.
TEST_ENV="bats-core"
if [ "${0#*"$TEST_ENV"}" == "$0" ]; then
    SC_Main
fi
