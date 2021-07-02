#!/bin/bash
set -e

#this is what deploys all the right OMGX contracts
yarn run deploy

# serve the addresses and the state dump
exec ./bin/serve_dump.sh
