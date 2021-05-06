#!/bin/bash

set -e

yarn build:contracts:ovm
yarn build:typescript &
yarn build:contracts
