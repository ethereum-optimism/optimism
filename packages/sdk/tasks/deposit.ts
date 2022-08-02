import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import {
  predeploys,
  getContractInterface,
} from '@eth-optimism/contracts-bedrock'
import { Event } from 'ethers'

import {
  CrossChainMessenger,
  StandardBridgeAdapter,
  MessageStatus,
} from '../src'

// TODO(tynes): this task could be modularized in the future
// so that it can deposit an arbitrary token. Right now it
// deploys a WETH9 contract, mints some WETH9 and then
// deposits that into L2 through the StandardBridge
task('deposit', 'Deposits WETH9 onto L2.')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .addParam(
    'opNodeProviderUrl',
    'op-node provider URL',
    'http://localhost:7545',
    types.string
  )
  .setAction(async (args, hre) => {
    const { utils } = hre.ethers

    const signers = await hre.ethers.getSigners()
    if (signers.length === 0) {
      throw new Error('No configured signers')
    }
    // Use the first configured signer for simplicity
    const signer = signers[0]
    const address = await signer.getAddress()
    console.log(`Using signer ${address}`)

    // Ensure that the signer has a balance before trying to
    // do anything
    const balance = await signer.getBalance()
    if (balance.eq(0)) {
      throw new Error('Signer has no balance')
    }

    const l2Provider = new hre.ethers.providers.StaticJsonRpcProvider(
      args.l2ProviderUrl
    )

    const Deployment__L2OutputOracleProxy = await hre.deployments.get(
      'L2OutputOracleProxy'
    )

    const l2Signer = new hre.ethers.Wallet(
      hre.network.config.accounts[0],
      l2Provider
    )

    const Artifact__WETH9 = await hre.deployments.getArtifact('WETH9')
    const Factory__WETH9 = new hre.ethers.ContractFactory(
      Artifact__WETH9.abi,
      Artifact__WETH9.bytecode,
      signer
    )

    const Deployment__OptimismMintableERC20TokenFactory =
      await hre.deployments.get('OptimismMintableERC20Factory')

    const Deployment__OptimismPortalProxy = await hre.deployments.get(
      'OptimismPortalProxy'
    )

    const Deployment__L1StandardBridgeProxy = await hre.deployments.get(
      'L1StandardBridgeProxy'
    )

    const Deployment__L1CrossDomainMessengerProxy = await hre.deployments.get(
      'L1CrossDomainMessengerProxy'
    )

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: signer,
      l2SignerOrProvider: l2Signer,
      l1ChainId: await signer.getChainId(),
      l2ChainId: await l2Signer.getChainId(),
      bridges: {
        Standard: {
          Adapter: StandardBridgeAdapter,
          l1Bridge: Deployment__L1StandardBridgeProxy.address,
          l2Bridge: predeploys.L2StandardBridge,
        },
      },
      contracts: {
        l1: {
          L1StandardBridge: Deployment__L1StandardBridgeProxy.address,
          L1CrossDomainMessenger:
            Deployment__L1CrossDomainMessengerProxy.address,
          L2OutputOracle: Deployment__L2OutputOracleProxy.address,
          OptimismPortal: Deployment__OptimismPortalProxy.address,
        },
      },
      bedrock: true,
    })

    const OptimismMintableERC20TokenFactory = await hre.ethers.getContractAt(
      Deployment__OptimismMintableERC20TokenFactory.abi,
      predeploys.OptimismMintableERC20Factory,
      l2Signer
    )

    console.log('Deploying WETH9 to L1')
    const WETH9 = await Factory__WETH9.deploy()
    await WETH9.deployTransaction.wait()
    console.log(`Deployed to ${WETH9.address}`)

    console.log('Creating L2 WETH9')
    const deployTx =
      await OptimismMintableERC20TokenFactory.createOptimismMintableERC20(
        WETH9.address,
        'L2 Wrapped Ether',
        'L2-WETH9'
      )
    const receipt = await deployTx.wait()
    const event = receipt.events.find(
      (e: Event) => e.event === 'OptimismMintableERC20Created'
    )
    if (!event) {
      throw new Error('Unable to find OptimismMintableERC20Created event')
    }
    // TODO(tynes): may need to be updated based on
    // https://github.com/ethereum-optimism/optimism/pull/3104
    const l2WethAddress = event.args.remoteToken
    console.log(`Deployed to ${l2WethAddress}`)

    console.log('Wrapping ETH')
    const deposit = await signer.sendTransaction({
      value: utils.parseEther('1'),
      to: WETH9.address,
    })
    await deposit.wait()
    console.log('ETH wrapped')

    console.log(`Approving WETH9 for deposit`)
    const approvalTx = await messenger.approveERC20(
      WETH9.address,
      l2WethAddress,
      hre.ethers.constants.MaxUint256
    )
    await approvalTx.wait()
    console.log('WETH9 approved')

    console.log('Depositing WETH9 to L2')
    const depositTx = await messenger.depositERC20(
      WETH9.address,
      l2WethAddress,
      utils.parseEther('1')
    )
    await depositTx.wait()
    console.log('ERC20 deposited')

    const messageReceipt = await messenger.waitForMessageReceipt(depositTx)
    if (messageReceipt.receiptStatus !== 1) {
      throw new Error('deposit failed')
    }

    const L2WETH9 = new hre.ethers.Contract(
      l2WethAddress,
      getContractInterface('OptimismMintableERC20'),
      l2Signer
    )

    const l2Balance = await L2WETH9.balanceOf(await signer.getAddress())
    if (l2Balance.lt(utils.parseEther('1'))) {
      throw new Error('bad deposit')
    }
    console.log('Deposit success')

    console.log('Starting withdrawal')
    const preBalance = await WETH9.balanceOf(signer.address)
    const tx = await messenger.withdrawERC20(
      WETH9.address,
      l2WethAddress,
      utils.parseEther('1')
    )
    await tx.wait()

    setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(tx)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
    }, 3000)

    const now = Math.floor(Date.now() / 1000)

    console.log('Waiting for message to be able to be relayed')
    await messenger.waitForMessageStatus(tx, MessageStatus.READY_FOR_RELAY)

    const finalize = await messenger.finalizeMessage(tx)
    await finalize.wait()
    console.log(`Took ${Math.floor(Date.now() / 1000) - now} seconds`)

    const postBalance = await WETH9.balanceOf(signer.address)

    const expectedBalance = preBalance.add(utils.parseEther('1'))
    if (!expectedBalance.eq(postBalance)) {
      throw new Error('Balance mismatch')
    }
    console.log('Withdrawal success')
  })
