#!/bin/bash

set -u
set -o pipefail

###
### get_loadbalancer.sh - find the load balancer associated with the vault service
###
### Usage:
###   get_loadbalancer.sh [options]
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

	exit 255
}

function getlb {
	echo -n "> Vault Load Balancer: " >&2

	local output
	local vault
	local ip

	if ! output=$(kubectl get services); then
		echo "ERROR: Could not execute kubectl command"
		exit 255
	fi

	vault=$(echo "${output}" | egrep "^vault-active ")
	if [[ "${vault}" == "" ]]; then
		echo "Not Found"
		exit 0
	fi

	ip=$(echo "${vault}" | awk '{print $4}')
	echo ${ip}
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

getlb

exit 0