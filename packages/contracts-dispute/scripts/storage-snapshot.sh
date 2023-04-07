#!/usr/bin/env bash

set -e

if ! command -v forge &> /dev/null
then
    echo "forge could not be found. Please install forge by running:"
    echo "curl -L https://foundry.paradigm.xyz | bash"
    exit
fi

contracts=(
  src/AttestationDisputeGame.sol:AttestationDisputeGame
  src/BondManager.sol:BondManager
  src/DisputeGameFactory.sol:DisputeGameFactory
  src/FaultDisputeGame.sol:FaultDisputeGame
)

dir=$(dirname "$0")

echo "Creating storage layout diagrams.."

echo "=======================" > $dir/../.storage-layout
echo "👁👁 STORAGE LAYOUT snapshot 👁👁" >> $dir/../.storage-layout
echo "=======================" >> $dir/../.storage-layout

for contract in ${contracts[@]}
do
  echo -e "\n=======================" >> $dir/../.storage-layout
  echo "➡ $contract">> $dir/../.storage-layout
  echo -e "=======================\n" >> $dir/../.storage-layout
  forge inspect --pretty $contract storage-layout >> $dir/../.storage-layout
done
echo "Storage layout snapshot stored at $dir/../.storage-layout"
