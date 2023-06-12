import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, l2OutputOracleProxy, l2OutputOracleImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'L2OutputOracleProxy',
        iface: 'L2OutputOracle',
        signerOrProvider: deployer,
      },
      {
        name: 'L2OutputOracle',
      },
    ])

  const startingBlockNumber = hre.deployConfig.l2OutputOracleStartingBlockNumber
  let startingTimestamp = hre.deployConfig.l2OutputOracleStartingTimestamp

  if (startingTimestamp < 0) {
    const l1StartingBlock = await hre.ethers.provider.getBlock(
      hre.deployConfig.l1StartingBlockTag
    )
    if (l1StartingBlock === null) {
      throw new Error(
        `Cannot fetch block tag ${hre.deployConfig.l1StartingBlockTag}`
      )
    }
    startingTimestamp = l1StartingBlock.timestamp
  }

  try {
    const tx = await proxyAdmin.upgradeAndCall(
      l2OutputOracleProxy.address,
      l2OutputOracleImpl.address,
      l2OutputOracleProxy.interface.encodeFunctionData('initialize', [
        startingBlockNumber,
        startingTimestamp,
      ])
    )
    await tx.wait()
  } catch (e) {
    console.log('L2OutputOracle already initialized')
  }

  const fetchedStartingBlockNumber =
    await l2OutputOracleProxy.callStatic.startingBlockNumber()
  const fetchedStartingTimestamp =
    await l2OutputOracleProxy.callStatic.startingTimestamp()
  assert(fetchedStartingBlockNumber.toNumber() === startingBlockNumber)
  assert(fetchedStartingTimestamp.toNumber() === startingTimestamp)

  console.log('Updgraded and initialized L2OutputOracle')
  const version = await l2OutputOracleProxy.callStatic.version()
  console.log(`L2OutputOracle version: ${version}`)
}

deployFn.tags = ['L2OutputOracleInitialize', 'l1']

export default deployFn
