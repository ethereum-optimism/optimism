#!/bin/bash -e
(cd minigeth/ && go build)
# 0 tx:         13284491
# low tx:       13284469
# delete issue: 13284053

minigeth/go-ethereum 13284469

