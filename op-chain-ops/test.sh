set -o errexit   # abort on nonzero exitstatus
set -o nounset   # abort on unbound variable
set -o pipefail  # don't hide errors within pipes
set -x

# Setup vars properly here

echo "Starting Migration"

./bin/op-migrate \
  --l1-rpc-url="http://127.0.0.1:8546" \
  --db-path="/Users/paul/Projects/celo-blockchain/tmp/testenv/validator-00/celo/" \
  --rollup-config-out="rollup.json" \
  --dry-run