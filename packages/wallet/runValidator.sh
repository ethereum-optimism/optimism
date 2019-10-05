#!/bin/bash

mkdir -p log
yarn run validator 2>&1 | tee log/validator.$(uuidgen).log
