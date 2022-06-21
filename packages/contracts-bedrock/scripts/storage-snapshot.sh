#!/usr/bin/env bash

set -e

if ! command -v forge &> /dev/null
then
    echo "forge could not be found. Please install forge by running:"
    echo "curl -L https://foundry.paradigm.xyz | bash"
    exit
fi

contracts=(
  L1CrossDomainMessenger
  L1StandardBridge
  L2OutputOracle
  OptimismPortal
  DeployerWhitelist
  GasPriceOracle
  L1Block
  L1BlockNumber
  L2CrossDomainMessenger
  L2StandardBridge
  L2ToL1MessagePasser
  OVM_ETH
  SequencerFeeVault
  WETH9
  ProxyAdmin
  Proxy
  L1ChugSplashProxy
  OptimismMintableERC20
  OptimismMintableTokenFactory
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
