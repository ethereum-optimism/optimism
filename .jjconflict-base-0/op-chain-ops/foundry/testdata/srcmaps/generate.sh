#!/bin/sh

set -euo

# Don't include previous build outputs
forge clean

forge build
