#!/bin/bash
# Borrowed from EthStaker's prepare for the merge guide
# See https://github.com/eth-educators/ethstaker-guides/blob/main/docs/prepare-for-the-merge.md#configuring-a-jwt-token-file

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
mkdir -p "${SCRIPT_DIR}/jwttoken"
if [[ ! -f "${SCRIPT_DIR}/jwttoken/jwt.hex" ]]
then
  openssl rand -hex 32 | tr -d "\n" | tee > "${SCRIPT_DIR}/jwttoken/jwt.hex"
else
  echo "${SCRIPT_DIR}/jwttoken/jwt.hex already exists!"
fi
