#!/bin/bash -e
(cd ../ && npx hardhat compile > /dev/null)
go build && (cd .. && ./mipsevm/mipsevm $@)
