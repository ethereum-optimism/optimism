/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { BigNumber } from 'ethers'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

import { predeploys } from '../src/constants'

const upgradeABIs = {
  L2OutputOracleProxy: async (deployConfig) => [
    'initialize(bytes32,uint256,address,address)',
    [
      deployConfig.l2OutputOracleGenesisL2Output,
      deployConfig.l2OutputOracleStartingBlockNumber,
      deployConfig.l2OutputOracleProposer,
      deployConfig.l2OutputOracleOwner,
    ],
  ],
  OptimismPortalProxy: async () => ['initialize', []],
  L1CrossDomainMessengerProxy: async () => ['initialize', []],
}

const deployFn: DeployFunction = async (hre) => {
  const { deploy, get } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre
  const l1 = hre.ethers.provider

  let deployL2StartingTimestamp = deployConfig.l2OutputOracleStartingTimestamp
  if (deployL2StartingTimestamp < 0) {
    const l1StartingBlock = await l1.getBlock(deployConfig.l1StartingBlockTag)
    if (l1StartingBlock === null) {
      throw new Error(
        `Cannot fetch block tag ${deployConfig.l1StartingBlockTag}`
      )
    }
    deployL2StartingTimestamp = l1StartingBlock.timestamp
  }

  const oracleProxy = await get('L2OutputOracleProxy')
  const portalProxy = await get('OptimismPortalProxy')
  const messengerProxy = await get('L1CrossDomainMessengerProxy')
  const bridgeProxy = await get('L1StandardBridgeProxy')
  const erc721BridgeProxy = await get('L1ERC721BridgeProxy')

  let nonce = await l1.getTransactionCount(deployer)
  const implTxs = [
    deploy('L2OutputOracle', {
      from: deployer,
      args: [
        deployConfig.l2OutputOracleSubmissionInterval,
        deployConfig.l2OutputOracleGenesisL2Output,
        deployConfig.l2OutputOracleHistoricalTotalBlocks,
        deployConfig.l2OutputOracleStartingBlockNumber,
        deployL2StartingTimestamp,
        deployConfig.l2BlockTime,
        deployConfig.l2OutputOracleProposer,
        deployConfig.l2OutputOracleOwner,
      ],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce,
    }),
    deploy('OptimismPortal', {
      from: deployer,
      args: [oracleProxy.address, 2],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('L1CrossDomainMessenger', {
      from: deployer,
      args: [portalProxy.address],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('L1StandardBridge', {
      from: deployer,
      args: [messengerProxy.address],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('OptimismMintableERC20Factory', {
      from: deployer,
      args: [bridgeProxy.address],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('AddressManager', {
      from: deployer,
      args: [],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('ProxyAdmin', {
      from: deployer,
      args: [deployer],
      log: true,
      waitConfirmations: deployConfig.deploymentWaitConfirmations,
      nonce: ++nonce,
    }),
    deploy('L1ERC721Bridge', {
      from: deployer,
      args: [messengerProxy.address, predeploys.L2ERC721Bridge],
      log: true,
      waitConfirmations: deployConfig.waitConfirmations,
      nonce: ++nonce,
    }),
  ]
  await Promise.all(implTxs)

  let tx
  // Reset the nonce for the next set of transactions
  for (const [proxy, upgrader] of Object.entries(upgradeABIs)) {
    const upgraderOut = await upgrader(deployConfig)
    const implName = proxy.replace('Proxy', '')
    const implDeployment = await get(implName)
    const implContract = await hre.ethers.getContractAt(
      implName,
      implDeployment.address
    )
    const proxyDeployment = await get(proxy)
    const proxyContract = await hre.ethers.getContractAt(
      'Proxy',
      proxyDeployment.address
    )
    console.log(`Upgrading contract impl ${implName}.`)
    tx = await proxyContract.upgradeToAndCall(
      implContract.address,
      implContract.interface.encodeFunctionData(
        upgraderOut[0] as string,
        upgraderOut[1] as any[]
      )
    )
    console.log(`Awaiting TX hash ${tx.hash}.`)
    await tx.wait()
    console.log('Done.')
  }

  const bridge = await get('L1StandardBridge')
  const bridgeProxyContract = await hre.ethers.getContractAt(
    'Proxy',
    bridgeProxy.address
  )
  console.log(`Upgrading L1StandardBridge at ${bridge.address}.`)
  tx = await bridgeProxyContract.upgradeTo(bridge.address)
  console.log(`Awaiting TX hash ${tx.hash}.`)
  await tx.wait()
  console.log('Done')

  const factory = await get('OptimismMintableERC20Factory')
  const factoryProxy = await get('OptimismMintableERC20FactoryProxy')
  const factoryProxyContract = await hre.ethers.getContractAt(
    'Proxy',
    factoryProxy.address
  )
  console.log(`Upgrading OptimismMintableERC20Factory at ${factory.address}.`)
  tx = await factoryProxyContract.upgradeTo(factory.address)
  console.log(`Awaiting TX hash ${tx.hash}.`)
  await tx.wait()
  console.log('Done')

  const erc721Bridge = await get('L1ERC721Bridge')
  const erc721BridgeProxyContract = await hre.ethers.getContractAt(
    'Proxy',
    erc721BridgeProxy.address
  )
  console.log(`Upgrading L1ERC721Bridge at ${erc721Bridge.address}`)
  tx = await erc721BridgeProxyContract.upgradeTo(erc721Bridge.address)
  console.log(`Awaiting TX hash ${tx.hash}.`)
  await tx.wait()
  console.log('Done')

  await validateOracle(hre, deployConfig, deployL2StartingTimestamp)
  await validatePortal(hre)
  await validateMessenger(hre)
  await validateBridge(hre)
  await validateTokenFactory(hre)
  await validateERC721Bridge(hre)
}

const validateOracle = async (hre, deployConfig, deployL2StartingTimestamp) => {
  const proxy = await hre.deployments.get('L2OutputOracleProxy')
  const L2OutputOracle = await hre.ethers.getContractAt(
    'L2OutputOracle',
    proxy.address
  )

  const submissionInterval = await L2OutputOracle.SUBMISSION_INTERVAL()
  if (
    !submissionInterval.eq(
      BigNumber.from(deployConfig.l2OutputOracleSubmissionInterval)
    )
  ) {
    throw new Error('submission internal misconfigured')
  }

  const historicalBlocks = await L2OutputOracle.HISTORICAL_TOTAL_BLOCKS()
  if (
    !historicalBlocks.eq(
      BigNumber.from(deployConfig.l2OutputOracleHistoricalTotalBlocks)
    )
  ) {
    throw new Error('historical total blocks misconfigured')
  }

  const startingBlockNumber = await L2OutputOracle.STARTING_BLOCK_NUMBER()
  if (
    !startingBlockNumber.eq(
      BigNumber.from(deployConfig.l2OutputOracleStartingBlockNumber)
    )
  ) {
    throw new Error('starting block number misconfigured')
  }

  const startingTimestamp = await L2OutputOracle.STARTING_TIMESTAMP()
  if (!startingTimestamp.eq(BigNumber.from(deployL2StartingTimestamp))) {
    throw new Error('starting timestamp misconfigured')
  }
  const l2BlockTime = await L2OutputOracle.L2_BLOCK_TIME()
  if (!l2BlockTime.eq(BigNumber.from(deployConfig.l2BlockTime))) {
    throw new Error('L2 block time misconfigured')
  }
}

const validatePortal = async (hre) => {
  const oracle = await hre.deployments.get('L2OutputOracleProxy')
  const proxy = await hre.deployments.get('OptimismPortalProxy')

  const OptimismPortal = await hre.ethers.getContractAt(
    'OptimismPortal',
    proxy.address
  )
  const l2Oracle = await OptimismPortal.L2_ORACLE()
  if (l2Oracle !== oracle.address) {
    throw new Error('L2 Oracle mismatch')
  }
}

const validateMessenger = async (hre) => {
  const portal = await hre.deployments.get('OptimismPortalProxy')
  const proxy = await hre.deployments.get('L1CrossDomainMessengerProxy')
  const L1CrossDomainMessenger = await hre.ethers.getContractAt(
    'L1CrossDomainMessenger',
    proxy.address
  )
  const portalAddress = await L1CrossDomainMessenger.portal()
  if (portalAddress !== portal.address) {
    throw new Error('portal misconfigured')
  }
}

const validateBridge = async (hre) => {
  const messenger = await hre.deployments.get('L1CrossDomainMessengerProxy')
  const proxy = await hre.deployments.get('L1StandardBridgeProxy')
  const L1StandardBridge = await hre.ethers.getContractAt(
    'L1StandardBridge',
    proxy.address
  )
  if (messenger.address !== (await L1StandardBridge.messenger())) {
    throw new Error('misconfigured messenger')
  }
}

// The messenger address should be set to the proxy messenger
// The other bridge should be set to the predeploy on L2
const validateERC721Bridge = async (hre) => {
  const messenger = await hre.deployments.get('L1CrossDomainMessengerProxy')
  const proxy = await hre.deployments.get('L1ERC721BridgeProxy')
  const L1ERC721Bridge = await hre.ethers.getContractAt(
    'L1ERC721Bridge',
    proxy.address
  )
  if (messenger.address !== (await L1ERC721Bridge.messenger())) {
    throw new Error('misconfigured messenger')
  }
  if ((await L1ERC721Bridge.otherBridge()) !== predeploys.L2ERC721Bridge) {
    throw new Error('misconfigured otherBridge')
  }
}

const validateTokenFactory = async (hre) => {
  const bridge = await hre.deployments.get('L1StandardBridgeProxy')
  const proxy = await hre.deployments.get('OptimismMintableERC20FactoryProxy')
  const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
    'OptimismMintableERC20Factory',
    proxy.address
  )
  if (bridge.address !== (await OptimismMintableERC20Factory.bridge())) {
    throw new Error('bridge misconfigured')
  }
}

deployFn.tags = ['InitImplementations']

export default deployFn
