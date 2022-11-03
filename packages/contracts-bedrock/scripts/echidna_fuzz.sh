#!/bin/bash

SCRIPTNAME=`basename "$0"`

printf "[$SCRIPTNAME] Cleaning hardhat build directory!\n"
npx hardhat clean
printf "\n\n"

printf "[$SCRIPTNAME] Compiling with hardhat!\n"
COMPILATION=$(npx hardhat compile 2>&1)
echo "$COMPILATION" | awk '1;/Error: Must compile with ast/{exit}'
printf "\n\n"

if [ -z "$(ls -A ./artifacts/build-info 2> /dev/null)" ]; then
   printf "Failed to compile, halting!\n"
   exit 1
fi

printf "[$SCRIPTNAME] Invoking echidna!\n"
echidna-test --contract $1 --crytic-args --hardhat-ignore-compile .