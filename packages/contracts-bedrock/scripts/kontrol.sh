#!/usr/bin/env bash
set -exuo pipefail
# Setup Kontrol 
export KONTROL_VERSION=$(cat .kontrolrc)
docker run --name optimism-ci \
        --rm \
        -v $(pwd)/kout/proofs:/home/user/workspace/packages/contracts-bedrock/kout/proofs/ \
        --interactive \
        --tty \
        --detach \
        --user root \
        --workdir /home/user/workspace \
        runtimeverificationinc/kontrol:ubuntu-jammy-${KONTROL_VERSION}
# Copy in current test envionrment
docker cp . optimism-ci:/home/user/workspace
docker exec optimism-ci chown -R user:user /home/user
# Run Kontrol Tests
docker exec -u user optimism-ci bash -c 'cd packages/contracts-bedrock && ./test/kontrol/kontrol/run-kontrol.sh'
