#!/bin/bash

set -e

yarn build:contracts
yarn generate:artifacts
yarn build:typescript
