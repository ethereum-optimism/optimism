import { task } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import { predeploys } from '../src'

// expectedSemver is the semver version of the contracts
// deployed at bedrock deployment
const expectedSemver = '0.0.1'
const implSlot =
  '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'
const adminSlot =
  '0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103'
const prefix = '0x420000000000000000000000000000000000'

// checkPredeploys will ensure that all of the predeploys are set
const checkPredeploys = async (hre: HardhatRuntimeEnvironment) => {
  console.log('Checking predeploys are configured correctly')
  for (let i = 0; i < 2048; i++) {
    const num = hre.ethers.utils.hexZeroPad('0x' + i.toString(16), 2)
    const addr = hre.ethers.utils.getAddress(
      hre.ethers.utils.hexConcat([prefix, num])
    )

    const code = await hre.ethers.provider.getCode(addr)
    if (code === '0x') {
      throw new Error(`no code found at ${addr}`)
    }

    if (addr === predeploys.GovernanceToken || addr === predeploys.ProxyAdmin) {
      continue
    }

    const slot = await hre.ethers.provider.getStorageAt(addr, adminSlot)
    const admin = hre.ethers.utils.hexConcat([
      '0x000000000000000000000000',
      predeploys.ProxyAdmin,
    ])

    if (admin !== slot) {
      throw new Error(`incorrect admin slot in ${addr}`)
    }
  }
}

// assertSemver will ensure that the semver is the correct version
const assertSemver = (version: string, name: string, override?: string) => {
  let target = expectedSemver
  if (override) {
    target = override
  }
  if (version !== target) {
    throw new Error(
      `${name}: version mismatch. Got ${version}, expected ${target}`
    )
  }
  console.log(`  - version: ${version}`)
}

// checkProxy will print out the proxy slots
const checkProxy = async (hre: HardhatRuntimeEnvironment, name: string) => {
  const address = predeploys[name]
  if (!address) {
    throw new Error(`unknown contract name: ${name}`)
  }

  const impl = await hre.ethers.provider.getStorageAt(address, implSlot)
  const admin = await hre.ethers.provider.getStorageAt(address, adminSlot)

  console.log(`  - EIP-1967 implementation slot: ${impl}`)
  console.log(`  - EIP-1967 admin slot: ${admin}`)
}

// assertProxy will require the proxy is set
const assertProxy = async (hre: HardhatRuntimeEnvironment, name: string) => {
  const address = predeploys[name]
  if (!address) {
    throw new Error(`unknown contract name: ${name}`)
  }

  const code = await hre.ethers.provider.getCode(address)
  const deployInfo = await hre.artifacts.readArtifact('Proxy')

  if (code !== deployInfo.deployedBytecode) {
    throw new Error(`${address}: code mismatch`)
  }

  const impl = await hre.ethers.provider.getStorageAt(address, implSlot)
  const implAddress = '0x' + impl.slice(26)
  const implCode = await hre.ethers.provider.getCode(implAddress)
  if (implCode === '0x') {
    throw new Error('No code at implementation')
  }
}

const check = {
  // LegacyMessagePasser
  // - check version
  // - is behind a proxy
  LegacyMessagePasser: async (hre: HardhatRuntimeEnvironment) => {
    const LegacyMessagePasser = await hre.ethers.getContractAt(
      'LegacyMessagePasser',
      predeploys.LegacyMessagePasser
    )

    const version = await LegacyMessagePasser.version()
    assertSemver(version, 'LegacyMessagePasser')

    await checkProxy(hre, 'LegacyMessagePasser')
    await assertProxy(hre, 'LegacyMessagePasser')
  },
  // DeployerWhitelist
  // - check version
  // - is behind a proxy
  // - owner is `address(0)`
  DeployerWhitelist: async (hre: HardhatRuntimeEnvironment) => {
    const DeployerWhitelist = await hre.ethers.getContractAt(
      'DeployerWhitelist',
      predeploys.DeployerWhitelist
    )

    const version = await DeployerWhitelist.version()
    assertSemver(version, 'DeployerWhitelist')

    const owner = await DeployerWhitelist.owner()
    if (owner !== hre.ethers.constants.AddressZero) {
      throw new Error('owner misconfigured')
    }
    console.log(`  - owner: ${owner}`)

    await checkProxy(hre, 'DeployerWhitelist')
    await assertProxy(hre, 'DeployerWhitelist')
  },
  // L2CrossDomainMessenger
  // - check version
  // - check OTHER_MESSENGER
  // - check l1CrossDomainMessenger (legacy)
  // - is behind a proxy
  // - check owner
  // - check initialized
  L2CrossDomainMessenger: async (hre: HardhatRuntimeEnvironment) => {
    const L2CrossDomainMessenger = await hre.ethers.getContractAt(
      'L2CrossDomainMessenger',
      predeploys.L2CrossDomainMessenger
    )

    const version = await L2CrossDomainMessenger.version()
    assertSemver(version, 'L2CrossDomainMessenger')

    const xDomainMessageSenderSlot = await hre.ethers.provider.getStorageAt(
      predeploys.L2CrossDomainMessenger,
      204
    )

    const xDomainMessageSender = '0x' + xDomainMessageSenderSlot.slice(26)
    if (xDomainMessageSender !== '0x000000000000000000000000000000000000dead') {
      throw new Error('xDomainMessageSender not set')
    }

    const otherMessenger = await L2CrossDomainMessenger.OTHER_MESSENGER()
    if (otherMessenger === hre.ethers.constants.AddressZero) {
      throw new Error('otherMessenger misconfigured')
    }
    console.log(`  - otherMessenger: ${otherMessenger}`)

    const l1CrossDomainMessenger =
      await L2CrossDomainMessenger.l1CrossDomainMessenger()
    console.log(`  - l1CrossDomainMessenger: ${l1CrossDomainMessenger}`)

    await checkProxy(hre, 'L2CrossDomainMessenger')
    await assertProxy(hre, 'L2CrossDomainMessenger')

    const owner = await L2CrossDomainMessenger.owner()
    if (owner === hre.ethers.constants.AddressZero) {
      throw new Error('owner misconfigured')
    }
    console.log(`  - owner: ${owner}`)

    const MESSAGE_VERSION = await L2CrossDomainMessenger.MESSAGE_VERSION()
    console.log(`  - MESSAGE_VERSION: ${MESSAGE_VERSION}`)
    const MIN_GAS_CALLDATA_OVERHEAD =
      await L2CrossDomainMessenger.MIN_GAS_CALLDATA_OVERHEAD()
    console.log(`  - MIN_GAS_CALLDATA_OVERHEAD: ${MIN_GAS_CALLDATA_OVERHEAD}`)
    const MIN_GAS_CONSTANT_OVERHEAD =
      await L2CrossDomainMessenger.MIN_GAS_CONSTANT_OVERHEAD()
    console.log(`  - MIN_GAS_CONSTANT_OVERHEAD: ${MIN_GAS_CONSTANT_OVERHEAD}`)
    const MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR =
      await L2CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR()
    console.log(
      `  - MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR: ${MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR}`
    )
    const MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR =
      await L2CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR()
    console.log(
      `  - MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR: ${MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR}`
    )

    const slot = await hre.ethers.provider.getStorageAt(
      predeploys.L2CrossDomainMessenger,
      0
    )

    const spacer = '0x' + slot.slice(26)
    console.log(`  - legacy spacer: ${spacer}`)

    const initialized = '0x' + slot.slice(24, 26)
    if (initialized !== '0x01') {
      throw new Error('not initialized')
    }
    console.log(`  - initialized: ${initialized}`)
  },
  // GasPriceOracle
  // - check version
  // - check decimals
  GasPriceOracle: async (hre: HardhatRuntimeEnvironment) => {
    const GasPriceOracle = await hre.ethers.getContractAt(
      'GasPriceOracle',
      predeploys.GasPriceOracle
    )

    const version = await GasPriceOracle.version()
    assertSemver(version, 'GasPriceOracle')

    const decimals = await GasPriceOracle.decimals()
    if (!decimals.eq(6)) {
      throw new Error('decimals misconfigured')
    }
    console.log(`  - decimals: ${decimals.toNumber()}`)

    await checkProxy(hre, 'GasPriceOracle')
    await assertProxy(hre, 'GasPriceOracle')
  },
  // L2StandardBridge
  // - check version
  L2StandardBridge: async (hre: HardhatRuntimeEnvironment) => {
    const L2StandardBridge = await hre.ethers.getContractAt(
      'L2StandardBridge',
      predeploys.L2StandardBridge
    )

    const version = await L2StandardBridge.version()
    assertSemver(version, 'L2StandardBridge', '0.0.2')

    const OTHER_BRIDGE = await L2StandardBridge.OTHER_BRIDGE()
    if (OTHER_BRIDGE === hre.ethers.constants.AddressZero) {
      throw new Error('invalid OTHER_BRIDGE')
    }
    console.log(`  - OTHER_BRIDGE: ${OTHER_BRIDGE}`)

    const MESSENGER = await L2StandardBridge.MESSENGER()
    if (MESSENGER !== predeploys.L2CrossDomainMessenger) {
      throw new Error('misconfigured MESSENGER')
    }

    await checkProxy(hre, 'L2StandardBridge')
    await assertProxy(hre, 'L2StandardBridge')
  },
  // SequencerFeeVault
  // - check version
  // - check RECIPIENT
  // - check l1FeeWallet (legacy)
  SequencerFeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const SequencerFeeVault = await hre.ethers.getContractAt(
      'SequencerFeeVault',
      predeploys.SequencerFeeVault
    )

    const version = await SequencerFeeVault.version()
    assertSemver(version, 'SequencerFeeVault')

    const RECIPIENT = await SequencerFeeVault.RECIPIENT()
    if (RECIPIENT === hre.ethers.constants.AddressZero) {
      throw new Error('undefined RECIPIENT')
    }
    console.log(`  - RECIPIENT: ${RECIPIENT}`)

    const l1FeeWallet = await SequencerFeeVault.l1FeeWallet()
    if (l1FeeWallet === hre.ethers.constants.AddressZero) {
      throw new Error('undefined l1FeeWallet')
    }
    console.log(`  - l1FeeWallet: ${l1FeeWallet}`)

    const MIN_WITHDRAWAL_AMOUNT =
      await SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    await checkProxy(hre, 'SequencerFeeVault')
    await assertProxy(hre, 'SequencerFeeVault')
  },
  // OptimismMintableERC20Factory
  // - check version
  OptimismMintableERC20Factory: async (hre: HardhatRuntimeEnvironment) => {
    const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC20Factory',
      predeploys.OptimismMintableERC20Factory
    )

    const version = await OptimismMintableERC20Factory.version()
    assertSemver(version, 'OptimismMintableERC20Factory', '1.0.0')

    const BRIDGE = await OptimismMintableERC20Factory.BRIDGE()
    if (BRIDGE === hre.ethers.constants.AddressZero) {
      throw new Error('BRIDGE misconfigured')
    }

    await checkProxy(hre, 'OptimismMintableERC20Factory')
    await assertProxy(hre, 'OptimismMintableERC20Factory')
  },
  // L1BlockNumber
  // - check version
  L1BlockNumber: async (hre: HardhatRuntimeEnvironment) => {
    const L1BlockNumber = await hre.ethers.getContractAt(
      'L1BlockNumber',
      predeploys.L1BlockNumber
    )

    const version = await L1BlockNumber.version()
    assertSemver(version, 'L1BlockNumber')

    await checkProxy(hre, 'L1BlockNumber')
    await assertProxy(hre, 'L1BlockNumber')
  },
  // L1Block
  // - check version
  L1Block: async (hre: HardhatRuntimeEnvironment) => {
    const L1Block = await hre.ethers.getContractAt(
      'L1Block',
      predeploys.L1Block
    )

    const version = await L1Block.version()
    assertSemver(version, 'L1Block')

    await checkProxy(hre, 'L1Block')
    await assertProxy(hre, 'L1Block')
  },
  // LegacyERC20ETH
  // - not behind a proxy
  // - check name
  // - check symbol
  // - check decimals
  // - check BRIDGE
  // - check REMOTE_TOKEN
  // - totalSupply should be set to 0
  LegacyERC20ETH: async (hre: HardhatRuntimeEnvironment) => {
    const LegacyERC20ETH = await hre.ethers.getContractAt(
      'LegacyERC20ETH',
      predeploys.LegacyERC20ETH
    )

    const name = await LegacyERC20ETH.name()
    if (name !== 'Ether') {
      throw new Error('name mismatch')
    }
    console.log(`  - name: ${name}`)

    const symbol = await LegacyERC20ETH.symbol()
    if (symbol !== 'ETH') {
      throw new Error('symbol mismatch')
    }
    console.log(`  - symbol: ${symbol}`)

    const decimals = await LegacyERC20ETH.decimals()
    if (decimals !== 18) {
      throw new Error('decimals mismatch')
    }
    console.log(`  - decimals: ${decimals}`)

    const BRIDGE = await LegacyERC20ETH.BRIDGE()
    if (BRIDGE !== predeploys.L2StandardBridge) {
      throw new Error('BRIDGE misconfigured')
    }

    const REMOTE_TOKEN = await LegacyERC20ETH.REMOTE_TOKEN()
    if (REMOTE_TOKEN !== hre.ethers.constants.AddressZero) {
      throw new Error('REMOTE_TOKEN misconfigured')
    }

    const totalSupply = await LegacyERC20ETH.totalSupply()
    if (!totalSupply.eq(0)) {
      throw new Error('totalSupply not 0')
    }

    await checkProxy(hre, 'LegacyERC20ETH')
    // No proxy at this address, don't call assertProxy
  },
  // WETH9
  // - check name
  // - check symbol
  // - check decimals
  // - is behind a proxy
  WETH9: async (hre: HardhatRuntimeEnvironment) => {
    const WETH9 = await hre.ethers.getContractAt('WETH9', predeploys.WETH9)

    const name = await WETH9.name()
    if (name !== 'Wrapped Ether') {
      throw new Error('name misconfigured')
    }
    console.log(`  - name: ${name}`)

    const symbol = await WETH9.symbol()
    if (symbol !== 'WETH') {
      throw new Error('symbol misconfigured')
    }
    console.log(`  - symbol: ${symbol}`)

    const decimals = await WETH9.decimals()
    if (decimals !== 18) {
      throw new Error('decimals misconfigured')
    }
    console.log(`  - decimals: ${decimals}`)

    await checkProxy(hre, 'WETH9')
    await assertProxy(hre, 'WETH9')
  },
  // GovernanceToken
  // - not behind a proxy
  // - check name
  // - check symbol
  // - check owner
  GovernanceToken: async (hre: HardhatRuntimeEnvironment) => {
    const GovernanceToken = await hre.ethers.getContractAt(
      'GovernanceToken',
      predeploys.GovernanceToken
    )

    const name = await GovernanceToken.name()
    if (name !== 'Optimism') {
      throw new Error('name misconfigured')
    }
    console.log(`  - name: ${name}`)

    const symbol = await GovernanceToken.symbol()
    if (symbol !== 'OP') {
      throw new Error('symbol misconfigured')
    }
    console.log(`  - symbol: ${symbol}`)

    const owner = await GovernanceToken.owner()
    console.log(`  - owner: ${owner}`)

    const totalSupply = await GovernanceToken.totalSupply()
    console.log(`  - totalSupply: ${totalSupply}`)

    await checkProxy(hre, 'GovernanceToken')
    // No proxy at this address, don't call assertProxy
  },
  // L2ERC721Bridge
  // - check version
  L2ERC721Bridge: async (hre: HardhatRuntimeEnvironment) => {
    const L2ERC721Bridge = await hre.ethers.getContractAt(
      'L2ERC721Bridge',
      predeploys.L2ERC721Bridge
    )

    const version = await L2ERC721Bridge.version()
    assertSemver(version, 'L2ERC721Bridge')

    const MESSENGER = await L2ERC721Bridge.MESSENGER()
    if (MESSENGER === hre.ethers.constants.AddressZero) {
      throw new Error('MESSENGER misconfigured')
    }

    const OTHER_BRIDGE = await L2ERC721Bridge.OTHER_BRIDGE()
    if (OTHER_BRIDGE === hre.ethers.constants.AddressZero) {
      throw new Error('OTHER_BRIDGE misconfigured')
    }
  },
  // OptimismMintableERC721Factory
  // - check version
  OptimismMintableERC721Factory: async (hre: HardhatRuntimeEnvironment) => {
    const OptimismMintableERC721Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC721Factory',
      predeploys.OptimismMintableERC721Factory
    )

    const version = await OptimismMintableERC721Factory.version()
    assertSemver(version, 'OptimismMintableERC721Factory', '1.0.0')

    const BRIDGE = await OptimismMintableERC721Factory.BRIDGE()
    console.log(`  - BRIDGE: ${BRIDGE}`)

    const REMOTE_CHAIN_ID =
      await OptimismMintableERC721Factory.REMOTE_CHAIN_ID()
    console.log(`  - REMOTE_CHAIN_ID: ${REMOTE_CHAIN_ID}`)

    await checkProxy(hre, 'OptimismMintableERC721Factory')
    await assertProxy(hre, 'OptimismMintableERC721Factory')
  },
  // ProxyAdmin
  // - check owner
  ProxyAdmin: async (hre: HardhatRuntimeEnvironment) => {
    const ProxyAdmin = await hre.ethers.getContractAt(
      'ProxyAdmin',
      predeploys.ProxyAdmin
    )

    const owner = await ProxyAdmin.owner()
    if (owner === hre.ethers.constants.AddressZero) {
      throw new Error('misconfigured owner')
    }
    console.log(`  - owner: ${owner}`)

    const addressManager = await ProxyAdmin.addressManager()
    console.log(`  - addressManager: ${addressManager}`)
  },
  // BaseFeeVault
  // - check version
  // - check MIN_WITHDRAWAL_AMOUNT
  // - check RECIPIENT
  BaseFeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const BaseFeeVault = await hre.ethers.getContractAt(
      'BaseFeeVault',
      predeploys.BaseFeeVault
    )

    const version = await BaseFeeVault.version()
    assertSemver(version, 'BaseFeeVault')

    const MIN_WITHDRAWAL_AMOUNT = await BaseFeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    const RECIPIENT = await BaseFeeVault.RECIPIENT()
    if (RECIPIENT === hre.ethers.constants.AddressZero) {
      throw new Error(`RECIPIENT misconfigured`)
    }
    console.log(`  - RECIPIENT: ${RECIPIENT}`)

    assertSemver(version, 'BaseFeeVault')
    await checkProxy(hre, 'BaseFeeVault')
    await assertProxy(hre, 'BaseFeeVault')
  },
  // L1FeeVault
  // - check version
  // - check MIN_WITHDRAWAL_AMOUNT
  // - check RECIPIENT
  L1FeeVault: async (hre: HardhatRuntimeEnvironment) => {
    const L1FeeVault = await hre.ethers.getContractAt(
      'L1FeeVault',
      predeploys.L1FeeVault
    )

    const version = await L1FeeVault.version()
    assertSemver(version, 'L1FeeVault')

    const MIN_WITHDRAWAL_AMOUNT = await L1FeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    const RECIPIENT = await L1FeeVault.RECIPIENT()
    if (RECIPIENT === hre.ethers.constants.AddressZero) {
      throw new Error(`RECIPIENT misconfigured`)
    }
    console.log(`  - RECIPIENT: ${RECIPIENT}`)

    assertSemver(version, 'L1FeeVault')
    await checkProxy(hre, 'L1FeeVault')
    await assertProxy(hre, 'L1FeeVault')
  },
  // L2ToL1MessagePasser
  // - check version
  L2ToL1MessagePasser: async (hre: HardhatRuntimeEnvironment) => {
    const L2ToL1MessagePasser = await hre.ethers.getContractAt(
      'L2ToL1MessagePasser',
      predeploys.L2ToL1MessagePasser
    )

    const version = await L2ToL1MessagePasser.version()
    assertSemver(version, 'L2ToL1MessagePasser')

    const MESSAGE_VERSION = await L2ToL1MessagePasser.MESSAGE_VERSION()
    console.log(`  - MESSAGE_VERSION: ${MESSAGE_VERSION}`)

    const messageNonce = await L2ToL1MessagePasser.messageNonce()
    console.log(`  - messageNonce: ${messageNonce}`)

    await checkProxy(hre, 'L2ToL1MessagePasser')
    await assertProxy(hre, 'L2ToL1MessagePasser')
  },
}

task(
  'check-l2',
  'Checks a freshly migrated L2 system for correct migration'
).setAction(async (_, hre: HardhatRuntimeEnvironment) => {
  // Ensure that all the predeploys exist, including the not
  // currently configured ones
  await checkPredeploys(hre)
  // Check the currently configured predeploys
  for (const [name, fn] of Object.entries(check)) {
    const address = predeploys[name]
    console.log(`${name}: ${address}`)
    await fn(hre)
  }
})
