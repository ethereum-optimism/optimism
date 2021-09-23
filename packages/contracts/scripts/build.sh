#!/bin/bash

set -e

yarn build:contracts
yarn build:contracts:ovm
yarn generate:artifacts
yarn build:typescript
