#!/bin/bash

set -u
set -o pipefail

###
### gen_storage.sh - sets up the storage classes required by vault
###
### Usage:
###   gen_storage.sh [options]
###
### Options:
###   -h | --help                Show help / usage
###

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

	exit -1
}

# gen_storage makes sure that the storage classes are created properly
gen_storage() {
	echo "> Generate Storage Classes" >&2

    cd k8s/storage
	kubectl apply -f storage-class-data.yaml
	kubectl apply -f storage-class-audit.yaml

    cd ..
}

##
## main
##

while [[ $# -gt 0 ]]; do
	case $1 in 
	-h | --help) 
		usage
	;;
	--)
		shift 
		break
		;;
	-*) usage "Invalid argument: $1" 1>&2 ;;
	*) usage "Invalid argument: $1" 1>&2 ;;
	esac
	shift
done

gen_storage