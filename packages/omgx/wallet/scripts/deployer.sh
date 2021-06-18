#!/bin/bash
set -e

yarn run deploy

# serve the addrs and the state dump
exec ./bin/serve_dump.sh
