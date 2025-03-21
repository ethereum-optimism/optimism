#!/bin/bash

{
    # Define a lock file
    LOCKFILE="/tmp/go-ffi.lock"

    # Function to clean up the lock file on exit
    cleanup() {
        rm -rf "$LOCKFILE"
    }

    # Get the directory of the script
    dir=$(dirname "$(realpath "$0")")
    # Check if the target binary exists
    if [[ ! -f "$dir/$1" ]]; then
        # Wait for the lock file to be released
        while ! mkdir "$LOCKFILE" > /dev/null; do
            sleep 0.3s
        done

        # Set a trap to call cleanup on exit
        trap cleanup EXIT

        # Ensure only one process is building the binaries
        if [[ ! -f "$dir/$1" ]]; then
            pushd "$dir/../.."
            just build-go-ffi > /dev/null
            popd
        fi
        cleanup
        # Remove the trap since it's no longer needed
        trap - EXIT
    fi
} > /dev/null 2>&1 # Ensures nothing is printed to the console during the build process


target_binary="$dir/$1"
shift  # This removes the first argument
# Call the target binary with remaining arguments
$target_binary "$@"