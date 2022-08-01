#!/bin/bash

rm -rf artifacts forge-artifacts

# Handle slither bug unable to work with the foundry tests
TEMP=$(mktemp -d)
mv contracts/test $TEMP/test

slither .

mv $TEMP/test contracts/test
