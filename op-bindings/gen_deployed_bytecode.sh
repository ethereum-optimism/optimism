#/bin/bash
set -eu


if [ "$#" -ne 2 ]; then
	echo "This script takes 2 arguments - CONTRACT_NAME PACKAGE"
	exit 1
fi


TYPE=$1
PACKAGE=$2


# Convert to lower case to respect golang package naming conventions
TYPE_LOWER=$(echo ${TYPE} | tr '[:upper:]' '[:lower:]')
FILENAME="${TYPE_LOWER}_deployed.go"
FILE="${PACKAGE}/${FILENAME}"
DEPLOYED_BYTECODE=$(cat "bin/${TYPE_LOWER}_deployed.hex")


echo "// Code generated - DO NOT EDIT." > ${FILE}
echo "// This file is a generated binding and any manual changes will be lost." >> ${FILE}
echo "package ${PACKAGE}" >> ${FILE}
echo "var ${TYPE}DeployedBin = \"${DEPLOYED_BYTECODE}\""  >> ${FILE}
gofmt -s -w ${FILE}
