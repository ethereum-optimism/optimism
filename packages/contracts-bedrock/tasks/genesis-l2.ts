import fs from 'fs'
import path from 'path'
import assert from 'assert'

import { OptimismGenesis, State } from '@eth-optimism/core-utils'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import { predeploys } from '../src'

const prefix = '0x420000000000000000000000000000000000'
const implementationSlot =
  '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'
const adminSlot =
  '0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103'

const toCodeAddr = (addr: string) => {
  const address = ethers.utils.hexConcat([
    '0xc0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3',
    '0x' + addr.slice(prefix.length),
  ])
  return ethers.utils.getAddress(address)
}

const assertEvenLength = (str: string) => {
  assert(str.length % 2 === 0, str)
}

// TODO: fix this to be compatible with smock's version
const getStorageLayout = async (
  hre: HardhatRuntimeEnvironment,
  name: string
) => {
  const buildInfo = await hre.artifacts.getBuildInfo(name)
  const keys = Object.keys(buildInfo.output.contracts)
  for (const key of keys) {
    if (name === path.basename(key, '.sol')) {
      const contract = buildInfo.output.contracts[key]
      const storageLayout = (contract[name] as any).storageLayout
      return storageLayout || { storage: [], types: {} }
    }
  }
  throw new Error(`Cannot locate storageLayout for ${name}`)
}

task('genesis-l2', 'create a genesis config')
  .addOptionalParam(
    'outfile',
    'The file to write the output JSON to',
    'genesis.json'
  )
  .setAction(async (args, hre) => {
    const {
      computeStorageSlots,
      // eslint-disable-next-line @typescript-eslint/no-var-requires
    } = require('@defi-wonderland/smock/dist/src/utils')

    const { deployConfig } = hre

    // Use the addresses of the proxies here instead of the implementations
    // Be backwards compatible
    let ProxyL1CrossDomainMessenger = await hre.deployments.getOrNull(
      'Proxy__OVM_L1CrossDomainMessenger'
    )
    if (ProxyL1CrossDomainMessenger === undefined) {
      ProxyL1CrossDomainMessenger = await hre.deployments.get(
        'L1CrossDomainMessengerProxy'
      )
    }
    // Be backwards compatible
    let ProxyL1StandardBridge = await hre.deployments.getOrNull(
      'Proxy__OVM_L1StandardBridge'
    )
    if (ProxyL1StandardBridge === undefined) {
      ProxyL1StandardBridge = await hre.deployments.get('L1StandardBridgeProxy')
    }

    const variables = {
      L2ToL1MessagePasser: {
        nonce: 0,
      },
      L2CrossDomainMessenger: {
        _initialized: 1,
        _owner: deployConfig.l2CrossDomainMessengerOwner,
        xDomainMsgSender: '0x000000000000000000000000000000000000dEaD',
        msgNonce: 0,
        otherMessenger: ProxyL1CrossDomainMessenger.address,
        // TODO: handle blockedSystemAddresses mapping
        // blockedSystemAddresses: [{key: '', value: ''}],
      },
      GasPriceOracle: {
        _owner: deployConfig.gasPriceOracleOwner,
        overhead: deployConfig.gasPriceOracleOverhead,
        scalar: deployConfig.gasPriceOracleScalar,
        decimals: deployConfig.gasPriceOracleDecimals,
      },
      L2StandardBridge: {
        messenger: predeploys.L2CrossDomainMessenger,
        otherBridge: ProxyL1StandardBridge.address,
      },
      SequencerFeeVault: {
        l1FeeWallet: ethers.constants.AddressZero,
      },
      L1Block: {
        number: deployConfig.l1BlockInitialNumber,
        timestamp: deployConfig.l1BlockInitialTimestamp,
        basefee: deployConfig.l1BlockInitialBasefee,
        hash: deployConfig.l1BlockInitialHash,
        sequenceNumber: deployConfig.l1BlockInitialSequenceNumber,
      },
      LegacyERC20ETH: {
        bridge: predeploys.L2StandardBridge,
        remoteToken: ethers.constants.AddressZero,
        _name: 'Ether',
        _symbol: 'ETH',
      },
      WETH9: {
        name: 'Wrapped Ether',
        symbol: 'WETH',
        decimals: 18,
      },
      GovernanceToken: {
        name: 'Optimism',
        symbol: 'OP',
        _owner: deployConfig.proxyAdmin,
      },
    }

    assertEvenLength(implementationSlot)
    assertEvenLength(adminSlot)
    assertEvenLength(deployConfig.proxyAdmin)

    const predeployAddrs = new Set()
    for (const addr of Object.values(predeploys)) {
      predeployAddrs.add(ethers.utils.getAddress(addr))
    }

    // TODO: geth likes strings for nonce and balance now
    const alloc: State = {}

    // Set a proxy at each predeploy address
    const proxy = await hre.artifacts.readArtifact('Proxy')
    for (let i = 0; i <= 2048; i++) {
      const num = ethers.utils.hexZeroPad('0x' + i.toString(16), 2)
      const addr = ethers.utils.getAddress(
        ethers.utils.hexConcat([prefix, num])
      )

      // There is no proxy at LegacyERC20ETH or the GovernanceToken
      if (
        addr === ethers.utils.getAddress(predeploys.LegacyERC20ETH) ||
        addr === ethers.utils.getAddress(predeploys.GovernanceToken)
      ) {
        continue
      }

      alloc[addr] = {
        nonce: '0x0',
        balance: '0x0',
        code: proxy.deployedBytecode,
        storage: {
          [adminSlot]: deployConfig.proxyAdmin,
        },
      }

      if (predeployAddrs.has(ethers.utils.getAddress(addr))) {
        alloc[addr].storage[implementationSlot] = toCodeAddr(addr)
      }
    }

    // Set the GovernanceToken in the state
    // Cannot easily set storage due to no easy access to compiler
    // output
    const governanceToken = await hre.deployments.getArtifact('GovernanceToken')
    alloc[predeploys.GovernanceToken] = {
      nonce: '0x0',
      balance: '0x0',
      code: governanceToken.deployedBytecode,
    }

    // Give each predeploy a single wei
    for (let i = 0; i <= 0xff; i++) {
      const buf = Buffer.alloc(2)
      buf.writeUInt16BE(i, 0)
      const addr = ethers.utils.hexConcat([
        '0x000000000000000000000000000000000000',
        ethers.utils.hexZeroPad(buf, 2),
      ])
      alloc[addr] = {
        balance: '0x1',
      }
    }

    if (deployConfig.fundDevAccounts) {
      const accounts = [
        '0x14dC79964da2C08b23698B3D3cc7Ca32193d9955',
        '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
        '0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec',
        '0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f',
        '0x2546BcD3c84621e976D8185a91A922aE77ECEc30',
        '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
        '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
        '0x71bE63f3384f5fb98995898A86B02Fb2426c5788',
        '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199',
        '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
        '0x976EA74026E726554dB657fA54763abd0C3a0aa9',
        '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
        '0xBcd4042DE499D14e55001CcbB24a551F3b954096',
        '0xFABB0ac9d68B0B445fB7357272Ff202C5651694a',
        '0xa0Ee7A142d267C1f36714E4a8F75612F20a79720',
        '0xbDA5747bFD65F08deb54cb465eB87D40e51B197E',
        '0xcd3B766CCDd6AE721141F452C550Ca635964ce71',
        '0xdD2FD4581271e230360230F9337D5c0430Bf44C0',
        '0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097',
        '0xde3829a23df1479438622a08a116e8eb3f620bb5',
        '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
      ]

      for (const account of accounts) {
        alloc[account] = {
          balance:
            '0x200000000000000000000000000000000000000000000000000000000000000',
        }
      }
    }

    // Set the predeploys in the state
    for (const [name, addr] of Object.entries(predeploys)) {
      if (name === 'GovernanceToken') {
        continue
      }
      const artifact = await hre.artifacts.readArtifact(name)
      assertEvenLength(artifact.deployedBytecode)

      const allocAddr = name === 'LegacyERC20ETH' ? addr : toCodeAddr(addr)
      assertEvenLength(allocAddr)

      alloc[allocAddr] = {
        nonce: '0x00',
        balance: '0x00',
        code: artifact.deployedBytecode,
        storage: {},
      }

      const storageLayout = await getStorageLayout(hre, name)
      if (storageLayout === undefined) {
        throw new Error(`cannot find storage layout for ${name}`)
      }
      const slots = computeStorageSlots(storageLayout, variables[name])

      for (const slot of slots) {
        alloc[allocAddr].storage[slot.key] = slot.val
      }
    }

    const genesis: OptimismGenesis = {
      config: {
        chainId: deployConfig.genesisBlockChainid,
        homesteadBlock: 0,
        eip150Block: 0,
        eip155Block: 0,
        eip158Block: 0,
        byzantiumBlock: 0,
        constantinopleBlock: 0,
        petersburgBlock: 0,
        istanbulBlock: 0,
        muirGlacierBlock: 0,
        berlinBlock: 0,
        londonBlock: 0,
        mergeForkBlock: 0,
        terminalTotalDifficulty: 0,
        clique: {
          period: 0,
          epoch: 30000,
        },
      },
      nonce: '0x1234',
      difficulty: '0x1',
      timestamp: ethers.BigNumber.from(
        deployConfig.startingTimestamp
      ).toHexString(),
      gasLimit: deployConfig.genesisBlockGasLimit,
      extraData: deployConfig.genesisBlockExtradata,
      optimism: {
        enabled: true,
        baseFeeRecipient: deployConfig.optimismBaseFeeRecipient,
        l1FeeRecipient: deployConfig.optimismL1FeeRecipient,
      },
      alloc,
    }

    fs.writeFileSync(args.outfile, JSON.stringify(genesis, null, 2))
  })
