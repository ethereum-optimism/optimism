#!/usr/bin/env bash

set -e

if ! command -v forge &> /dev/null
then
    echo "forge could not be found. Please install forge by running:"
    echo "curl -L https://foundry.paradigm.xyz | bash"
    exit
fi

contracts=(
  contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger
  contracts/L1/L1StandardBridge.sol:L1StandardBridge
  contracts/L1/L2OutputOracle.sol:L2OutputOracle
  contracts/L1/OptimismPortal.sol:OptimismPortal
  contracts/L1/SystemConfig.sol:SystemConfig
  contracts/legacy/DeployerWhitelist.sol:DeployerWhitelist
  contracts/L2/GasPriceOracle.sol:GasPriceOracle
  contracts/L2/L1Block.sol:L1Block
  contracts/legacy/L1BlockNumber.sol:L1BlockNumber
  contracts/L2/L2CrossDomainMessenger.sol:L2CrossDomainMessenger
  contracts/L2/L2StandardBridge.sol:L2StandardBridge
  contracts/L2/L2ToL1MessagePasser.sol:L2ToL1MessagePasser
  contracts/legacy/LegacyERC20ETH.sol:LegacyERC20ETH
  contracts/L2/SequencerFeeVault.sol:SequencerFeeVault
  contracts/vendor/WETH9.sol:WETH9
  contracts/universal/ProxyAdmin.sol:ProxyAdmin
  contracts/universal/Proxy.sol:Proxy
  contracts/legacy/L1ChugSplashProxy.sol:L1ChugSplashProxy
  contracts/universal/OptimismMintableERC20.sol:OptimismMintableERC20
  contracts/universal/OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory
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
