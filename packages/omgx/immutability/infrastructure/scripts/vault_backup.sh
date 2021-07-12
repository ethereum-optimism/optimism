#!/bin/bash

set -u
set -o pipefail

###
### vault_backup.sh - perform a rolling backup of the vault data
###
### Usage:
###   vault_backup.sh [options]
###
### Required Options:
###   -d | --dest-dir <str>     Filesytem path to the directory containing the vault backups
###
### Additional Options:
###   -p | --file-prefix <str>  The filename prefix for the backups (default: vault_snapshot)
###   -m | --max-backups <int>  The maximum number of backups to keep (default: 4)
###   -h | --help               Show help / usage
###
### Notes:
###   In order to perform the backup, you must point VAULT_ADDR at the RAFT leader. You can
###   use `vault operator raft list-peers` to determine this, then set up a port-forward to
###   that pod, and then execute this script against that specific pod.
###
###   In order to perform the vault backup, you need a Vault Token that has permissions to
###   initiate the backup.
### 

DEST_DIR=""
BACKUP_FILENAME=""
FILE_PREFIX="vault_snapshot"

(( MAX_BACKUPS=4 ))

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

# cleanup makes sure the temporary file is deleted
cleanup() {
    if [[ "${BACKUP_FILENAME}" != "" && -f "${BACKUP_FILENAME}" ]]; then
        rm -f "${BACKUP_FILENAME}"
    fi
}

# fail terminates the script and prints a message
fail() {
	MESSAGE=$1

	echo "[backup] ERROR: ${MESSAGE}"

    cleanup

	exit 254
}

# validate_config ensures that required variables are set
validate_config() {
	if [[ "${DEST_DIR}" == "" ]]; then
		usage "The Backup Destination Directory (-d) was not specified"
	fi

    if [[ ${MAX_BACKUPS} -lt 0 ]]; then
        fail "${MAX_BACKUPS} cannot be less than 0"
    fi

    if [[ "${VAULT_ADDR}" == "" ]]; then
        fail "VAULT_ADDR is not set"
    fi
}

# prepare_destination ensures that the destination exists and that it
# has the correct permissions
prepare_destination() {
    local dest_dir=$1

    if [[ ! -d "${dest_dir}" ]]; then
        if ! mkdir -p "${dest_dir}"; then
            fail "${dest_dir} does not exist and can't be created"
        fi
    fi

    chmod 750 "${dest_dir}"
}

# perform_backup takes the RAFT snapshot
perform_backup() {
    RESULT_VARIABLE=$1
    local dest_dir=$2
    local file_prefix=$3

    local backup_filename="${file_prefix}_${RANDOM}".raft
    local backup_size

    cd "${dest_dir}" || fail "perform_backup: Cannot change to directory ${dest_dir}"

    echo "[backup] Performing backup of RAFT data"

    if ! vault operator raft snapshot save "${backup_filename}"; then
        cd - > /dev/null 2>&1 || fail "perform_backup: Cannot return to previous directory"
        fail "Unable to perform backup of Vault RAFT data"
    fi

    backup_size=$(wc -c < "${backup_filename}")
    if [[ ${backup_size} -eq 0 ]]; then
        cd - > /dev/null 2>&1 || fail "perform_backup: Cannot return to previous directory"
        fail "The backup generated a 0-length file"
    fi

    cd - > /dev/null 2>&1 || fail "perform_backup: Cannot return to previous directory"

    eval "${RESULT_VARIABLE}"="${backup_filename}"
}

# cycle_backups ensures that "max_backups" backups are maintained
cycle_backups() {
    local backup_filename=$1
    local dest_dir=$2
    local max_backups
    local file_prefix=$4

    local last_file
    local cur_file
    local next_file

    local next_backup

    (( max_backups=$3 ))

    cd "${dest_dir}" || fail "cycle_backups: Cannot change to directory ${dest_dir}"

    echo "[backup] Cycling backup files"

    last_file="${file_prefix}-${max_backups}.raft"
    if [[ -f "${last_file}" ]]; then
        if ! rm -f "${last_file}"; then
            cd - > /dev/null 2>&1 || fail "cycle_backups: Cannot return to previous directory"
            fail "Cannot remove backup file ${last_file}"
        fi
    fi

    (( cur_backup=max_backups-1 ))

    while [[ ${cur_backup} -gt 0 ]]; do
        cur_file="${file_prefix}-${cur_backup}.raft"
        if [[ -f "${cur_file}" ]]; then
            (( next_backup=cur_backup+1 ))
            next_file="${file_prefix}-${next_backup}.raft"

            if ! mv -f "${cur_file}" "${next_file}"; then
                cd - > /dev/null 2>&1 || fail "cycle_backups: Cannot return to previous directory"
                fail "Cannot move backup file ${cur_file} to ${next_file}"
            fi
        fi

        (( cur_backup-=1 ))
    done

    cur_file="${file_prefix}.raft"
    if [[ -f "${cur_file}" ]]; then
        (( next_backup=1 ))
        next_file="${file_prefix}-${next_backup}.raft"

        if ! mv -f "${cur_file}" "${next_file}"; then
            cd - > /dev/null 2>&1 || fail "cycle_backups: Cannot return to previous directory"
            fail "Cannot move backup file ${cur_file} to ${next_file}"
        fi
    fi

    mv -f "${backup_filename}" "${cur_file}" 2> /dev/null

    cd - > /dev/null 2>&1 || fail "cycle_backups: Cannot return to previous directory"
}

# sighandler performs cleanup in the case of an interrupted process
function sighandler() {
    cleanup

    echo "[backup] Process Interrupted"

    exit 1
}

##
## main
##
while [[ $# -gt 0 ]]; do
	case $1 in 
	-d | --dest-dir) 
		DEST_DIR=$2
		shift
	;;
	-m | --max-backups) 
		(( MAX_BACKUPS=$2 ))
		shift
	;;
	-p | --file-prefix) 
		FILE_PREFIX=$2
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

prepare_destination "${DEST_DIR}"
perform_backup BACKUP_FILENAME "${DEST_DIR}" "${FILE_PREFIX}"
cycle_backups "${BACKUP_FILENAME}" "${DEST_DIR}" ${MAX_BACKUPS} "${FILE_PREFIX}"
cleanup

echo "[backup] Done"

exit 0