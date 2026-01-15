#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
mkdir -p ${SCRIPT_DIR}/jwttoken
if [[ ! -f ${SCRIPT_DIR}/jwttoken/jwt.hex ]]
then
  openssl rand -hex 32 | tr -d "\n" | tee > ${SCRIPT_DIR}/jwttoken/jwt.hex
  echo "Generated a JWT secret at ${SCRIPT_DIR}/jwttoken/jwt.hex"
else
  echo "${SCRIPT_DIR}/jwttoken/jwt.hex already exists!"
fi
