#!/usr/bin/env bash

set -e

if ! command -v forge &> /dev/null
then
    echo "forge could not be found. Please install forge by running:"
    echo "curl -L https://foundry.paradigm.xyz | bash"
    exit
fi

contracts=(
  src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger
  src/L1/L1StandardBridge.sol:L1StandardBridge
  src/L1/L2OutputOracle.sol:L2OutputOracle
  src/L1/OptimismPortal.sol:OptimismPortal
  src/L1/SystemConfig.sol:SystemConfig
  src/legacy/DeployerWhitelist.sol:DeployerWhitelist
  src/L2/L1Block.sol:L1Block
  src/legacy/L1BlockNumber.sol:L1BlockNumber
  src/L2/L2CrossDomainMessenger.sol:L2CrossDomainMessenger
  src/L2/L2StandardBridge.sol:L2StandardBridge
  src/L2/L2ToL1MessagePasser.sol:L2ToL1MessagePasser
  src/legacy/LegacyERC20ETH.sol:LegacyERC20ETH
  src/L2/SequencerFeeVault.sol:SequencerFeeVault
  src/L2/BaseFeeVault.sol:BaseFeeVault
  src/L2/L1FeeVault.sol:L1FeeVault
  src/vendor/WETH9.sol:WETH9
  src/universal/ProxyAdmin.sol:ProxyAdmin
  src/universal/Proxy.sol:Proxy
  src/legacy/L1ChugSplashProxy.sol:L1ChugSplashProxy
  src/universal/OptimismMintableERC20.sol:OptimismMintableERC20
  src/universal/OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory
  src/dispute/DisputeGameFactory.sol:DisputeGameFactory
)

dir=$(dirname "$0")

echo "Creating storage layout diagrams.."

echo "=======================" > $dir/../locks/storage-lock
echo "👁👁 STORAGE LAYOUT LOCK 👁👁" >> $dir/../locks/storage-lock
echo "=======================" >> $dir/../locks/storage-lock

for contract in ${contracts[@]}
do
  echo -e "\n=======================" >> $dir/../locks/storage-lock
  echo "➡ $contract">> $dir/../locks/storage-lock
  echo -e "=======================\n" >> $dir/../locks/storage-lock
  forge inspect --pretty $contract storageLayout >> $dir/../locks/storage-lock
done
echo "Storage layout lock stored at $dir/../locks/storage-lock"
