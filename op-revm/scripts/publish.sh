#!/bin/bash

# stop on error
set -e

cargo publish --package op-revm
