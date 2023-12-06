#!/usr/bin/env bash
set -exuo pipefail

blank_line() { echo '' >&2 ; }
notif() { echo "== $0: $@" >&2 ; }
quit()  { notif "[QUITTING]: $@" ; exit 0 ; }
fatal() { notif "[FATAL]: $@" ; exit 1 ; }
debug() { if ${do_debug}; then notif "[DEBUG]: $@"; fi ; }
safe_executor() { if ${do_debug}; then debug $@ ; else "$@" ; fi ; }

# Set Script Directory Variables
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Set Directory for run-kontrol.sh
KONTROL_DIR=$CURENT_DIR=$SCRIPT_DIR/../../test/kontrol/kontrol
CONTRACTS_DIR=$SCRIPT_DIR/..

notif "Run Directory Set to $RUN_DIR"

# Setup Kontrol 
