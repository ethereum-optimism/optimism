#!/bin/bash

set -e

yarn run build:typescript
yarn run build:contracts
yarn run build:contracts:ovm
