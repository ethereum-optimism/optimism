import { task } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import { predeploys } from '../src'

/**
 * Check that the predeploy contracts have been correctly configured
 *
 * LegacyMessagePasser
 * DeployerWhitelist
 * LegacyERC20ETH
 * WETH9
 * L2CrossDomainMessenger
 * L2StandardBridge
 * SequencerFeeVault
 * OptimismMintableERC20Factory
 * L1BlockNumber
 * GasPriceOracle
 * GovernanceToken
 * L1Block
 * L2ToL1MessagePasser
 * L2ERC721Bridge
 * OptimismMintableERC721Factory
 * ProxyAdmin
 * BaseFeeVault
 * L1FeeVault
 */

// expectedSemver is the semver version of the contracts
// deployed at bedrock deployment
const expectedSemver = '0.0.1'

const assertSemver = (version: string, name: string) => {
  if (version !== expectedSemver) {
    throw new Error(`${name}: version mismatch`)
  }
  console.log(`${name}: semver ${expectedSemver}`)
}

const check = {
  // LegacyMessagePasser
  // - check version
  LegacyMessagePasser: async (hre: HardhatRuntimeEnvironment) => {
    const LegacyMessagePasser = await hre.ethers.getContractAt(
      'LegacyMessagePasser',
      predeploys.LegacyMessagePasser
    )

    const version = await LegacyMessagePasser.version()
    assertSemver(version, 'LegacyMessagePasser')
  },
  // DeployerWhitelist
  // - check version
  DeployerWhitelist: async (hre: HardhatRuntimeEnvironment) => {
    const DeployerWhitelist = await hre.ethers.getContractAt(
      'DeployerWhitelist',
      predeploys.DeployerWhitelist
    )

    const version = await DeployerWhitelist.version()
    assertSemver(version, 'DeployerWhitelist')
  },
  L2CrossDomainMessenger: async (hre: HardhatRuntimeEnvironment) => {
    const L2CrossDomainMessenger = await hre.ethers.getContractAt(
      'L2CrossDomainMessenger',
      predeploys.L2CrossDomainMessenger
    )

    const version = await L2CrossDomainMessenger.version()
    assertSemver(version, 'L2CrossDomainMessenger')
  },
  GasPriceOracle: async (hre: HardhatRuntimeEnvironment) => {
    const GasPriceOracle = await hre.ethers.getContractAt(
      'GasPriceOracle',
      predeploys.GasPriceOracle
    )

    const version = await GasPriceOracle.version()
    assertSemver(version, 'GasPriceOracle')
  },
  L2StandardBridge: async (hre: HardhatRuntimeEnvironment) => {
    const L2StandardBridge = await hre.ethers.getContractAt(
      'L2StandardBridge',
      predeploys.L2StandardBridge
    )

    const version = await L2StandardBridge.version()
    assertSemver(version, 'L2StandardBridge')
  },
  SequencerFeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const SequencerFeeVault = await hre.ethers.getContractAt(
      'SequencerFeeVault',
      predeploys.SequencerFeeVault
    )

    const version = await SequencerFeeVault.version()
    assertSemver(version, 'SequencerFeeVault')
  },
  OptimismMintableERC20Factory: async (hre: HardhatRuntimeEnvironment) => {
    const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC20Factory',
      predeploys.OptimismMintableERC20Factory
    )

    const version = await OptimismMintableERC20Factory.version()
    assertSemver(version, 'OptimismMintableERC20Factory')
  },
  L1BlockNumber: async (hre: HardhatRuntimeEnvironment) => {
    const L1BlockNumber = await hre.ethers.getContractAt(
      'L1BlockNumber',
      predeploys.L1BlockNumber
    )

    const version = await L1BlockNumber.version()
    assertSemver(version, 'L1BlockNumber')
  },
  L1Block: async (hre: HardhatRuntimeEnvironment) => {
    const L1Block = await hre.ethers.getContractAt(
      'L1Block',
      predeploys.L1Block
    )

    const version = await L1Block.version()
    assertSemver(version, 'L1Block')
  },
  LegacyERC20ETH: async (hre: HardhatRuntimeEnvironment) => {
    const LegacyERC20ETH = await hre.ethers.getContractAt(
      'LegacyERC20ETH',
      predeploys.LegacyERC20ETH
    )

    const version = await LegacyERC20ETH.version()
    assertSemver(version, 'LegacyERC20ETH')
  },
  WETH9: async (/*hre: HardhatRuntimeEnvironment*/) => {
    // TODO
  },
  GovernanceToken: async (hre: HardhatRuntimeEnvironment) => {
    const GovernanceToken = await hre.ethers.getContractAt(
      'GovernanceToken',
      predeploys.GovernanceToken
    )

    const version = await GovernanceToken.version()
    assertSemver(version, 'GovernanceToken')
  },
  L2ERC721Bridge: async (hre: HardhatRuntimeEnvironment) => {
    const L2ERC721Bridge = await hre.ethers.getContractAt(
      'L2ERC721Bridge',
      predeploys.L2ERC721Bridge
    )

    const version = await L2ERC721Bridge.version()
    assertSemver(version, 'L2ERC721Bridge')
  },
  OptimismMintableERC721Factory: async (hre: HardhatRuntimeEnvironment) => {
    const OptimismMintableERC721Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC721Factory',
      predeploys.OptimismMintableERC721Factory
    )

    const version = await OptimismMintableERC721Factory.version()
    assertSemver(version, 'OptimismMintableERC721Factory')
  },
  ProxyAdmin: async (/*hre: HardhatRuntimeEnvironment*/) => {
    // TODO
  },
  BaseFeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const BaseFeeVault = await hre.ethers.getContractAt(
      'BaseFeeVault',
      predeploys.BaseFeeVault
    )

    const version = await BaseFeeVault.version()
    assertSemver(version, 'BaseFeeVault')
  },
  L1FeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const L1FeeVault = await hre.ethers.getContractAt(
      'L1FeeVault',
      predeploys.L1FeeVault
    )

    const version = await L1FeeVault.version()
    assertSemver(version, 'L1FeeVault')
  },
}

task('check-l2', 'Checks a freshly migrated L2 system for correct migration')
  //.addParam('')
  .setAction(async (_, hre: HardhatRuntimeEnvironment) => {
    for (const [name, fn] of Object.entries(check)) {
      console.log(`Checking ${name}`)
      await fn(hre)
    }
  })
