#!/bin/bash

set -u
set -o pipefail

###
### vault_restore.sh - perform a restore of the vault data
###
### Usage:
###   vault_restore.sh [options]
###
### Required Options:
###   -s | --src-dir <str>      Filesytem path to the directory containing the vault backups
###
### Additional Options:
###   -p | --file-prefix <str>  The filename prefix for the backups (default: vault_snapshot)
###   -b | --backup <int>       The specific backup to restore from (default: -1)
###   -h | --help               Show help / usage
###
### Notes:
###   In order to perform the restore, you must point VAULT_ADDR at the RAFT leader. You can
###   use `vault operator raft list-peers` to determine this, then set up a port-forward to
###   that pod, and then execute this script against that specific pod.
###
###   In order to perform the vault restore, you need a Vault Token that has appropriate 
###   permissions.
### 

SRC_DIR=""
FILE_PREFIX="vault_snapshot"

(( BACKUP_NUMBER=-1 ))

# usage displays some helpful information about the script and any errors that need
# to be emitted
usage() {
	MESSAGE=${1:-}

	awk -F'### ' '/^###/ { print $2 }' $0 >&2

	if [[ "${MESSAGE}" != "" ]]; then
		echo "" >&2
		echo "${MESSAGE}" >&2
		echo "" >&2
	fi

	exit 255
}

# fail terminates the script and prints a message
fail() {
	MESSAGE=$1

	echo "[restore] ERROR: ${MESSAGE}"

	exit 254
}

# validate_config ensures that required variables are set
validate_config() {
	if [[ "${SRC_DIR}" == "" ]]; then
		usage "The Backup Source Directory (-s) was not specified"
	fi

    if [[ "${VAULT_ADDR}" == "" ]]; then
        fail "VAULT_ADDR is not set"
    fi
}

# perform_restore takes the specified RAFT snapshot and restores the RAFT data 
# from that backup.
perform_restore() {
    local src_dir=$1
    local file_prefix=$2
    local backup_number

    (( backup_number=$3 ))

    local backup_filename

    if [[ ${backup_number} -eq -1 ]]; then
        backup_filename="${file_prefix}.raft"
    else
        backup_filename="${file_prefix}-${backup_number}.raft"
    fi

    cd "${src_dir}" || fail "perform_restore: Cannot change to directory ${src_dir}"

    if [[ ! "${backup_filename}" ]]; then
        cd - > /dev/null 2>&1 || fail "perform_restore: Cannot return to previous directory"
        fail "Specified file ${src_dir}/${backup_filename} does not exist or cannot be read"
    fi

    echo "[restore] Performing restore of RAFT data from ${backup_filename}"

    if ! vault operator raft snapshot restore "${backup_filename}"; then
        cd - > /dev/null 2>&1 || fail "perform_restore: Cannot return to previous directory"
        fail "Unable to perform restore of Vault RAFT data"
    fi

    cd - > /dev/null 2>&1 || fail "perform_restore: Cannot return to previous directory"
}

# sighandler performs cleanup in the case of an interrupted process
function sighandler() {
    echo "[restore] Process Interrupted"

    exit 1
}

##
## main
##
while [[ $# -gt 0 ]]; do
	case $1 in 
	-s | --src-dir) 
		SRC_DIR=$2
		shift
	;;
	-p | --file-prefix) 
		FILE_PREFIX=$2
		shift
	;;
	-b | --backup-number) 
		(( BACKUP_NUMBER=${2} ))
		shift
	;;
	-h | --help) 
		usage
	;;
	--)
		shift 
		break
		;;
	*) usage "Invalid argument: $1" 1>&2 ;;
	esac
	shift
done

validate_config

trap sighandler INT

perform_restore "${SRC_DIR}" "${FILE_PREFIX}" ${BACKUP_NUMBER}

echo "[restore] Done"

exit 0