import assert from 'assert'

import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { Contract, providers, Wallet, Signer, BigNumber } from 'ethers'

import { predeploys } from '../src'

// expectedSemver is the semver version of the contracts
// deployed at bedrock deployment
const expectedSemver = '1.0.0'
const implSlot =
  '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'
const adminSlot =
  '0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103'
const prefix = '0x420000000000000000000000000000000000'

const logLoud = () => {
  console.log('   !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!')
}

const yell = (msg: string) => {
  logLoud()
  console.log(msg)
  logLoud()
}

// checkPredeploys will ensure that all of the predeploys are set
const checkPredeploys = async (
  hre: HardhatRuntimeEnvironment,
  provider: providers.Provider
) => {
  console.log('Checking predeploys are configured correctly')
  for (let i = 0; i < 2048; i++) {
    const num = hre.ethers.utils.hexZeroPad('0x' + i.toString(16), 2)
    const addr = hre.ethers.utils.getAddress(
      hre.ethers.utils.hexConcat([prefix, num])
    )

    const code = await provider.getCode(addr)
    if (code === '0x') {
      throw new Error(`no code found at ${addr}`)
    }

    if (
      addr === predeploys.GovernanceToken ||
      addr === predeploys.ProxyAdmin ||
      addr === predeploys.WETH9
    ) {
      continue
    }

    const slot = await provider.getStorageAt(addr, adminSlot)
    const admin = hre.ethers.utils.hexConcat([
      '0x000000000000000000000000',
      predeploys.ProxyAdmin,
    ])

    if (admin !== slot) {
      throw new Error(`incorrect admin slot in ${addr}`)
    }

    if (i % 200 === 0) {
      console.log(`Checked through ${addr}`)
    }
  }
}

// assertSemver will ensure that the semver is the correct version
const assertSemver = async (
  contract: Contract,
  name: string,
  override?: string
) => {
  const version = await contract.version()
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
const checkProxy = async (
  _hre: HardhatRuntimeEnvironment,
  name: string,
  provider: providers.Provider
) => {
  const address = predeploys[name]
  if (!address) {
    throw new Error(`unknown contract name: ${name}`)
  }

  const impl = await provider.getStorageAt(address, implSlot)
  const admin = await provider.getStorageAt(address, adminSlot)

  console.log(`  - EIP-1967 implementation slot: ${impl}`)
  console.log(`  - EIP-1967 admin slot: ${admin}`)
}

// assertProxy will require the proxy is set
const assertProxy = async (
  hre: HardhatRuntimeEnvironment,
  name: string,
  provider: providers.Provider
) => {
  const address = predeploys[name]
  if (!address) {
    throw new Error(`unknown contract name: ${name}`)
  }

  const code = await provider.getCode(address)
  const deployInfo = await hre.artifacts.readArtifact('Proxy')

  if (code !== deployInfo.deployedBytecode) {
    throw new Error(`${address}: code mismatch`)
  }

  const impl = await provider.getStorageAt(address, implSlot)
  const implAddress = '0x' + impl.slice(26)
  const implCode = await provider.getCode(implAddress)
  if (implCode === '0x') {
    throw new Error('No code at implementation')
  }
}

// checks to make sure that the genesis magic value
// was set correctly
const checkGenesisMagic = async (
  hre: HardhatRuntimeEnvironment,
  l2Provider: providers.Provider,
  args
) => {
  const magic = '0x' + Buffer.from('BEDROCK').toString('hex')
  let startingBlockNumber: number

  // We have a connection to the L1 chain, fetch the remote value
  if (args.l1RpcUrl !== '') {
    const l1Provider = new hre.ethers.providers.StaticJsonRpcProvider(
      args.l1RpcUrl
    )

    // Get the address if it was passed in via CLI
    // Otherwise get the address from a hardhat deployment file
    let address: string
    if (args.l2OutputOracleAddress !== '') {
      address = args.l2OutputOracleAddress
    } else {
      const Deployment__L2OutputOracle = await hre.deployments.get(
        'L2OutputOracleProxy'
      )
      address = Deployment__L2OutputOracle.address
    }

    const abi = {
      inputs: [],
      name: 'startingBlockNumber',
      outputs: [{ internalType: 'uint256', name: '', type: 'uint256' }],
      stateMutability: 'view',
      type: 'function',
    }

    const L2OutputOracle = new hre.ethers.Contract(address, [abi], l1Provider)

    startingBlockNumber = await L2OutputOracle.startingBlockNumber()
  } else {
    // We do not have a connection to the L1 chain, use the local config
    // The `--network` flag must be set to the L1 network
    startingBlockNumber = hre.deployConfig.l2OutputOracleStartingBlockNumber
  }

  // ensure that the starting block number is a number
  startingBlockNumber = BigNumber.from(startingBlockNumber).toNumber()

  const block = await l2Provider.getBlock(startingBlockNumber)
  const extradata = block.extraData

  if (extradata !== magic) {
    throw new Error('magic value in extradata does not match')
  }
}

const check = {
  // LegacyMessagePasser
  // - check version
  // - is behind a proxy
  LegacyMessagePasser: async (
    hre: HardhatRuntimeEnvironment,
    signer: Signer
  ) => {
    const LegacyMessagePasser = await hre.ethers.getContractAt(
      'LegacyMessagePasser',
      predeploys.LegacyMessagePasser,
      signer
    )

    await assertSemver(LegacyMessagePasser, 'LegacyMessagePasser')
    await checkProxy(hre, 'LegacyMessagePasser', signer.provider)
    await assertProxy(hre, 'LegacyMessagePasser', signer.provider)
  },
  // DeployerWhitelist
  // - check version
  // - is behind a proxy
  // - owner is `address(0)`
  DeployerWhitelist: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const DeployerWhitelist = await hre.ethers.getContractAt(
      'DeployerWhitelist',
      predeploys.DeployerWhitelist,
      signer
    )

    await assertSemver(DeployerWhitelist, 'DeployerWhitelist')

    const owner = await DeployerWhitelist.owner()
    assert(owner === hre.ethers.constants.AddressZero)
    console.log(`  - owner: ${owner}`)

    await checkProxy(hre, 'DeployerWhitelist', signer.provider)
    await assertProxy(hre, 'DeployerWhitelist', signer.provider)
  },
  // L2CrossDomainMessenger
  // - check version
  // - check OTHER_MESSENGER
  // - check l1CrossDomainMessenger (legacy)
  // - is behind a proxy
  // - check owner
  // - check initialized
  L2CrossDomainMessenger: async (
    hre: HardhatRuntimeEnvironment,
    signer: Signer
  ) => {
    const L2CrossDomainMessenger = await hre.ethers.getContractAt(
      'L2CrossDomainMessenger',
      predeploys.L2CrossDomainMessenger,
      signer
    )

    await assertSemver(
      L2CrossDomainMessenger,
      'L2CrossDomainMessenger',
      '1.1.0'
    )

    const xDomainMessageSenderSlot = await signer.provider.getStorageAt(
      predeploys.L2CrossDomainMessenger,
      204
    )

    const xDomainMessageSender = '0x' + xDomainMessageSenderSlot.slice(26)
    assert(
      xDomainMessageSender === '0x000000000000000000000000000000000000dead'
    )

    const otherMessenger = await L2CrossDomainMessenger.OTHER_MESSENGER()
    assert(otherMessenger !== hre.ethers.constants.AddressZero)
    yell(`  - OTHER_MESSENGER: ${otherMessenger}`)

    const l1CrossDomainMessenger =
      await L2CrossDomainMessenger.l1CrossDomainMessenger()
    yell(`  - l1CrossDomainMessenger: ${l1CrossDomainMessenger}`)

    await checkProxy(hre, 'L2CrossDomainMessenger', signer.provider)
    await assertProxy(hre, 'L2CrossDomainMessenger', signer.provider)

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

    const slot = await signer.provider.getStorageAt(
      predeploys.L2CrossDomainMessenger,
      0
    )

    const spacer = '0x' + slot.slice(26)
    console.log(`  - legacy spacer: ${spacer}`)

    const initialized = '0x' + slot.slice(24, 26)
    assert(initialized === '0x01')
    console.log(`  - initialized: ${initialized}`)
  },
  // GasPriceOracle
  // - check version
  // - check decimals
  GasPriceOracle: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const GasPriceOracle = await hre.ethers.getContractAt(
      'GasPriceOracle',
      predeploys.GasPriceOracle,
      signer
    )

    await assertSemver(GasPriceOracle, 'GasPriceOracle')

    const decimals = await GasPriceOracle.decimals()
    assert(decimals.eq(6))
    console.log(`  - decimals: ${decimals.toNumber()}`)

    await checkProxy(hre, 'GasPriceOracle', signer.provider)
    await assertProxy(hre, 'GasPriceOracle', signer.provider)
  },
  // L2StandardBridge
  // - check version
  L2StandardBridge: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const L2StandardBridge = await hre.ethers.getContractAt(
      'L2StandardBridge',
      predeploys.L2StandardBridge,
      signer
    )

    await assertSemver(L2StandardBridge, 'L2StandardBridge', '1.1.0')

    const OTHER_BRIDGE = await L2StandardBridge.OTHER_BRIDGE()
    assert(OTHER_BRIDGE !== hre.ethers.constants.AddressZero)
    yell(`  - OTHER_BRIDGE: ${OTHER_BRIDGE}`)

    const MESSENGER = await L2StandardBridge.MESSENGER()
    assert(MESSENGER === predeploys.L2CrossDomainMessenger)

    await checkProxy(hre, 'L2StandardBridge', signer.provider)
    await assertProxy(hre, 'L2StandardBridge', signer.provider)
  },
  // SequencerFeeVault
  // - check version
  // - check RECIPIENT
  // - check l1FeeWallet (legacy)
  SequencerFeeVault: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const SequencerFeeVault = await hre.ethers.getContractAt(
      'SequencerFeeVault',
      predeploys.SequencerFeeVault,
      signer
    )

    await assertSemver(SequencerFeeVault, 'SequencerFeeVault')

    const RECIPIENT = await SequencerFeeVault.RECIPIENT()
    assert(RECIPIENT !== hre.ethers.constants.AddressZero)
    yell(`  - RECIPIENT: ${RECIPIENT}`)

    const l1FeeWallet = await SequencerFeeVault.l1FeeWallet()
    assert(l1FeeWallet !== hre.ethers.constants.AddressZero)
    console.log(`  - l1FeeWallet: ${l1FeeWallet}`)

    const MIN_WITHDRAWAL_AMOUNT =
      await SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    await checkProxy(hre, 'SequencerFeeVault', signer.provider)
    await assertProxy(hre, 'SequencerFeeVault', signer.provider)
  },
  // OptimismMintableERC20Factory
  // - check version
  OptimismMintableERC20Factory: async (
    hre: HardhatRuntimeEnvironment,
    signer: Signer
  ) => {
    const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC20Factory',
      predeploys.OptimismMintableERC20Factory,
      signer
    )

    await assertSemver(
      OptimismMintableERC20Factory,
      'OptimismMintableERC20Factory',
      '1.1.0'
    )

    const BRIDGE = await OptimismMintableERC20Factory.BRIDGE()
    assert(BRIDGE !== hre.ethers.constants.AddressZero)

    await checkProxy(hre, 'OptimismMintableERC20Factory', signer.provider)
    await assertProxy(hre, 'OptimismMintableERC20Factory', signer.provider)
  },
  // L1BlockNumber
  // - check version
  L1BlockNumber: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const L1BlockNumber = await hre.ethers.getContractAt(
      'L1BlockNumber',
      predeploys.L1BlockNumber,
      signer
    )

    await assertSemver(L1BlockNumber, 'L1BlockNumber')

    await checkProxy(hre, 'L1BlockNumber', signer.provider)
    await assertProxy(hre, 'L1BlockNumber', signer.provider)
  },
  // L1Block
  // - check version
  L1Block: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const L1Block = await hre.ethers.getContractAt(
      'L1Block',
      predeploys.L1Block,
      signer
    )

    await assertSemver(L1Block, 'L1Block')

    await checkProxy(hre, 'L1Block', signer.provider)
    await assertProxy(hre, 'L1Block', signer.provider)
  },
  // LegacyERC20ETH
  // - not behind a proxy
  // - check name
  // - check symbol
  // - check decimals
  // - check BRIDGE
  // - check REMOTE_TOKEN
  // - totalSupply should be set to 0
  LegacyERC20ETH: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const LegacyERC20ETH = await hre.ethers.getContractAt(
      'LegacyERC20ETH',
      predeploys.LegacyERC20ETH,
      signer
    )

    const name = await LegacyERC20ETH.name()
    assert(name === 'Ether')
    console.log(`  - name: ${name}`)

    const symbol = await LegacyERC20ETH.symbol()
    assert(symbol === 'ETH')
    console.log(`  - symbol: ${symbol}`)

    const decimals = await LegacyERC20ETH.decimals()
    assert(decimals === 18)
    console.log(`  - decimals: ${decimals}`)

    const BRIDGE = await LegacyERC20ETH.BRIDGE()
    assert(BRIDGE === predeploys.L2StandardBridge)

    const REMOTE_TOKEN = await LegacyERC20ETH.REMOTE_TOKEN()
    assert(REMOTE_TOKEN === hre.ethers.constants.AddressZero)

    const totalSupply = await LegacyERC20ETH.totalSupply()
    assert(totalSupply.eq(0))
    console.log(`  - totalSupply: ${totalSupply}`)

    await checkProxy(hre, 'LegacyERC20ETH', signer.provider)
    // No proxy at this address, don't call assertProxy
  },
  // WETH9
  // - check name
  // - check symbol
  // - check decimals
  WETH9: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const WETH9 = await hre.ethers.getContractAt(
      'WETH9',
      predeploys.WETH9,
      signer
    )

    const name = await WETH9.name()
    assert(name === 'Wrapped Ether')
    console.log(`  - name: ${name}`)

    const symbol = await WETH9.symbol()
    assert(symbol === 'WETH')
    console.log(`  - symbol: ${symbol}`)

    const decimals = await WETH9.decimals()
    assert(decimals === 18)
    console.log(`  - decimals: ${decimals}`)
  },
  // GovernanceToken
  // - not behind a proxy
  // - check name
  // - check symbol
  // - check owner
  GovernanceToken: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const GovernanceToken = await hre.ethers.getContractAt(
      'GovernanceToken',
      predeploys.GovernanceToken,
      signer
    )

    const name = await GovernanceToken.name()
    assert(name === 'Optimism')
    console.log(`  - name: ${name}`)

    const symbol = await GovernanceToken.symbol()
    assert(symbol === 'OP')
    console.log(`  - symbol: ${symbol}`)

    const owner = await GovernanceToken.owner()
    yell(`  - owner: ${owner}`)

    const totalSupply = await GovernanceToken.totalSupply()
    console.log(`  - totalSupply: ${totalSupply}`)

    await checkProxy(hre, 'GovernanceToken', signer.provider)
    // No proxy at this address, don't call assertProxy
  },
  // L2ERC721Bridge
  // - check version
  L2ERC721Bridge: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const L2ERC721Bridge = await hre.ethers.getContractAt(
      'L2ERC721Bridge',
      predeploys.L2ERC721Bridge,
      signer
    )

    await assertSemver(L2ERC721Bridge, 'L2ERC721Bridge', '1.1.0')

    const MESSENGER = await L2ERC721Bridge.MESSENGER()
    assert(MESSENGER !== hre.ethers.constants.AddressZero)
    console.log(`  - MESSENGER: ${MESSENGER}`)

    const OTHER_BRIDGE = await L2ERC721Bridge.OTHER_BRIDGE()
    assert(OTHER_BRIDGE !== hre.ethers.constants.AddressZero)
    yell(`  - OTHER_BRIDGE: ${OTHER_BRIDGE}`)

    await checkProxy(hre, 'L2ERC721Bridge', signer.provider)
    await assertProxy(hre, 'L2ERC721Bridge', signer.provider)
  },
  // OptimismMintableERC721Factory
  // - check version
  OptimismMintableERC721Factory: async (
    hre: HardhatRuntimeEnvironment,
    signer: Signer
  ) => {
    const OptimismMintableERC721Factory = await hre.ethers.getContractAt(
      'OptimismMintableERC721Factory',
      predeploys.OptimismMintableERC721Factory,
      signer
    )

    await assertSemver(
      OptimismMintableERC721Factory,
      'OptimismMintableERC721Factory',
      '1.1.0'
    )

    const BRIDGE = await OptimismMintableERC721Factory.BRIDGE()
    assert(BRIDGE !== hre.ethers.constants.AddressZero)
    console.log(`  - BRIDGE: ${BRIDGE}`)

    const REMOTE_CHAIN_ID =
      await OptimismMintableERC721Factory.REMOTE_CHAIN_ID()
    assert(REMOTE_CHAIN_ID !== 0)
    console.log(`  - REMOTE_CHAIN_ID: ${REMOTE_CHAIN_ID}`)

    await checkProxy(hre, 'OptimismMintableERC721Factory', signer.provider)
    await assertProxy(hre, 'OptimismMintableERC721Factory', signer.provider)
  },
  // ProxyAdmin
  // - check owner
  ProxyAdmin: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const ProxyAdmin = await hre.ethers.getContractAt(
      'ProxyAdmin',
      predeploys.ProxyAdmin,
      signer
    )

    const owner = await ProxyAdmin.owner()
    assert(owner !== hre.ethers.constants.AddressZero)
    yell(`  - owner: ${owner}`)

    const addressManager = await ProxyAdmin.addressManager()
    console.log(`  - addressManager: ${addressManager}`)

    await checkProxy(hre, 'ProxyAdmin', signer.provider)
    await assertProxy(hre, 'ProxyAdmin', signer.provider)
  },
  // BaseFeeVault
  // - check version
  // - check MIN_WITHDRAWAL_AMOUNT
  // - check RECIPIENT
  BaseFeeVault: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const BaseFeeVault = await hre.ethers.getContractAt(
      'BaseFeeVault',
      predeploys.BaseFeeVault,
      signer
    )

    await assertSemver(BaseFeeVault, 'BaseFeeVault')

    const MIN_WITHDRAWAL_AMOUNT = await BaseFeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    const RECIPIENT = await BaseFeeVault.RECIPIENT()
    assert(RECIPIENT !== hre.ethers.constants.AddressZero)
    yell(`  - RECIPIENT: ${RECIPIENT}`)

    await checkProxy(hre, 'BaseFeeVault', signer.provider)
    await assertProxy(hre, 'BaseFeeVault', signer.provider)
  },
  // L1FeeVault
  // - check version
  // - check MIN_WITHDRAWAL_AMOUNT
  // - check RECIPIENT
  L1FeeVault: async (hre: HardhatRuntimeEnvironment, signer: Signer) => {
    const L1FeeVault = await hre.ethers.getContractAt(
      'L1FeeVault',
      predeploys.L1FeeVault,
      signer
    )

    await assertSemver(L1FeeVault, 'L1FeeVault')

    const MIN_WITHDRAWAL_AMOUNT = await L1FeeVault.MIN_WITHDRAWAL_AMOUNT()
    console.log(`  - MIN_WITHDRAWAL_AMOUNT: ${MIN_WITHDRAWAL_AMOUNT}`)

    const RECIPIENT = await L1FeeVault.RECIPIENT()
    assert(RECIPIENT !== hre.ethers.constants.AddressZero)
    yell(`  - RECIPIENT: ${RECIPIENT}`)

    await checkProxy(hre, 'L1FeeVault', signer.provider)
    await assertProxy(hre, 'L1FeeVault', signer.provider)
  },
  // L2ToL1MessagePasser
  // - check version
  L2ToL1MessagePasser: async (
    hre: HardhatRuntimeEnvironment,
    signer: Signer
  ) => {
    const L2ToL1MessagePasser = await hre.ethers.getContractAt(
      'L2ToL1MessagePasser',
      predeploys.L2ToL1MessagePasser,
      signer
    )

    await assertSemver(L2ToL1MessagePasser, 'L2ToL1MessagePasser')

    const MESSAGE_VERSION = await L2ToL1MessagePasser.MESSAGE_VERSION()
    console.log(`  - MESSAGE_VERSION: ${MESSAGE_VERSION}`)

    const messageNonce = await L2ToL1MessagePasser.messageNonce()
    console.log(`  - messageNonce: ${messageNonce}`)

    await checkProxy(hre, 'L2ToL1MessagePasser', signer.provider)
    await assertProxy(hre, 'L2ToL1MessagePasser', signer.provider)
  },
}

task('check-l2', 'Checks a freshly migrated L2 system for correct migration')
  .addOptionalParam('l1RpcUrl', 'L1 RPC URL of node', '', types.string)
  .addOptionalParam('l2RpcUrl', 'L2 RPC URL of node', '', types.string)
  .addOptionalParam('chainId', 'Expected chain id', 0, types.int)
  .addOptionalParam(
    'l2OutputOracleAddress',
    'Address of the L2OutputOracle oracle',
    '',
    types.string
  )
  .addOptionalParam(
    'skipPredeployCheck',
    'Skip long check',
    false,
    types.boolean
  )
  .setAction(async (args, hre: HardhatRuntimeEnvironment) => {
    yell('Manually check values wrapped in !!!!')
    console.log()

    let signer: Signer = hre.ethers.provider.getSigner()

    if (args.l2RpcUrl !== '') {
      console.log('Using CLI URL for provider instead of hardhat network')
      const provider = new hre.ethers.providers.JsonRpcProvider(args.l2RpcUrl)
      signer = Wallet.createRandom().connect(provider)
    }

    if (args.chainId !== 0) {
      const chainId = await signer.getChainId()
      if (chainId !== args.chainId) {
        throw new Error(
          `Unexpected Chain ID. Got ${chainId}, expected ${args.chainId}`
        )
      }
      console.log(`Verified Chain ID: ${chainId}`)
    } else {
      console.log(`Skipping Chain ID validation...`)
    }

    // Ensure that all the predeploys exist, including the not
    // currently configured ones
    if (!args.skipPredeployCheck) {
      await checkPredeploys(hre, signer.provider)
    }

    await checkGenesisMagic(hre, signer.provider, args)

    console.log()
    // Check the currently configured predeploys
    for (const [name, fn] of Object.entries(check)) {
      const address = predeploys[name]
      console.log(`${name}: ${address}`)
      await fn(hre, signer)
    }
  })
