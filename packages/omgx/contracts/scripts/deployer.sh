#!/bin/bash
set -e

SERVE_ONLY=${SERVE_ONLY:-0} #so if you don't set, it defaults to deploying new contracts 

if [ $SERVE_ONLY == 1 ]
then
	IF_SERVE_ONLY_EQ_1_THEN_SERVE=${IF_SERVE_ONLY_EQ_1_THEN_SERVE:-rinkeby}
    echo "Not deploying contracts - serving static addresses in /deployment/$IF_SERVE_ONLY_EQ_1_THEN_SERVE only"
else
	#this is what deploys all the right OMGX contracts
    yarn run deploy

    # Register the deployed addresses with DTL
    if [ -n "$DTL_REGISTRY_URL" ] ; then
      echo "Will upload addresses.json to DTL"
      curl \
	  --fail \
	  --show-error \
	  --silent \
	  -H "Content-Type: application/json" \
	  --retry-connrefused \
	  --retry $RETRIES \
	  --retry-delay 5 \
	  -T dist/dumps/addresses.json \
	  "$DTL_REGISTRY_URL"
    fi
fi

# serve the addresses.json
exec ./bin/serve_dump.sh
