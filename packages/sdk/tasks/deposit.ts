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
// deposits and withdraws ETH through the bridge,
// deploys a WETH9 contract, mints some WETH9 and then
// deposits that into L2 through the StandardBridge.
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

    const Artifact__L2ToL1MessagePasser = await hre.deployments.getArtifact(
      'L2ToL1MessagePasser'
    )

    const Artifact__L2CrossDomainMessenger = await hre.deployments.getArtifact(
      'L2CrossDomainMessenger'
    )

    const Artifact__L2StandardBridge = await hre.deployments.getArtifact(
      'L2StandardBridge'
    )

    const Deployment__OptimismMintableERC20TokenFactory =
      await hre.deployments.get('OptimismMintableERC20Factory')

    const Deployment__OptimismPortal = await hre.deployments.get(
      'OptimismPortal'
    )

    const Deployment__OptimismPortalProxy = await hre.deployments.get(
      'OptimismPortalProxy'
    )

    const Deployment__L1StandardBridgeProxy = await hre.deployments.get(
      'L1StandardBridgeProxy'
    )

    const Deployment__L1CrossDomainMessenger = await hre.deployments.get(
      'L1CrossDomainMessenger'
    )

    const Deployment__L1CrossDomainMessengerProxy = await hre.deployments.get(
      'L1CrossDomainMessengerProxy'
    )

    const Deployment__L1StandardBridge = await hre.deployments.get(
      'L1StandardBridge'
    )

    const OptimismPortal = new hre.ethers.Contract(
      Deployment__OptimismPortalProxy.address,
      Deployment__OptimismPortal.abi,
      signer
    )

    const L1CrossDomainMessenger = new hre.ethers.Contract(
      Deployment__L1CrossDomainMessengerProxy.address,
      Deployment__L1CrossDomainMessenger.abi,
      signer
    )

    const L1StandardBridge = new hre.ethers.Contract(
      Deployment__L1StandardBridgeProxy.address,
      Deployment__L1StandardBridge.abi,
      signer
    )

    const L2ToL1MessagePasser = new hre.ethers.Contract(
      predeploys.L2ToL1MessagePasser,
      Artifact__L2ToL1MessagePasser.abi
    )

    const L2CrossDomainMessenger = new hre.ethers.Contract(
      predeploys.L2CrossDomainMessenger,
      Artifact__L2CrossDomainMessenger.abi
    )

    const L2StandardBridge = new hre.ethers.Contract(
      predeploys.L2StandardBridge,
      Artifact__L2StandardBridge.abi
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

    const senderBalanceBefore = await signer.getBalance()
    console.log(
      `Sender balance before: ${utils.formatEther(senderBalanceBefore)}`
    )

    const opBalanceBefore = await signer.provider.getBalance(
      OptimismPortal.address
    )

    // Deposit ETH
    console.log('Depositing ETH through StandardBridge')
    const ethDeposit = await messenger.depositETH(utils.parseEther('2'))
    const depositMessageReceipt = await messenger.waitForMessageReceipt(
      ethDeposit
    )
    if (depositMessageReceipt.receiptStatus !== 1) {
      throw new Error('deposit failed')
    }
    console.log('Deposit complete')

    const opBalanceAfter = await signer.provider.getBalance(
      OptimismPortal.address
    )

    if (!opBalanceBefore.add(utils.parseEther('2')).eq(opBalanceAfter)) {
      throw new Error(`OptimismPortal balance mismatch`)
    }

    console.log('Withdrawing ETH')
    const ethWithdraw = await messenger.withdrawETH(utils.parseEther('1'))
    const ethWithdrawReceipt = await ethWithdraw.wait()

    {
      // check the logs
      for (const log of ethWithdrawReceipt.logs) {
        try {
          switch (log.address) {
            case L2ToL1MessagePasser.address: {
              const parsed = L2ToL1MessagePasser.interface.parseLog(log)
              console.log(parsed.name)
              console.log(parsed.args)
              break
            }
            case L2StandardBridge.address: {
              const parsed = L2StandardBridge.interface.parseLog(log)
              console.log(parsed.name)
              console.log(parsed.args)
              break
            }
            case L2CrossDomainMessenger.address: {
              const parsed = L2CrossDomainMessenger.interface.parseLog(log)
              console.log(parsed.name)
              console.log(parsed.args)
              break
            }
          }
        } catch (e) {
          console.log('error parsing log')
          console.log(log)
          console.log()
        }
      }
    }

    console.log(
      `Withdrawal on L2 complete: ${ethWithdrawReceipt.transactionHash}`
    )
    console.log('Waiting to be able to withdraw')

    const id = setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(ethWithdrawReceipt)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
    }, 3000)

    await messenger.waitForMessageStatus(
      ethWithdrawReceipt,
      MessageStatus.READY_FOR_RELAY
    )
    clearInterval(id)

    const ethFinalize = await messenger.finalizeMessage(ethWithdrawReceipt)
    const ethFinalizeReceipt = await ethFinalize.wait()
    if (ethFinalizeReceipt.status !== 1) {
      throw new Error('Finalize withdrawal reverted')
    }

    const senderBalanceAfter = await signer.getBalance()
    console.log(
      `Sender balance after: ${utils.formatEther(senderBalanceAfter)}`
    )

    console.log(
      `ETH withdrawal complete: ${ethFinalizeReceipt.transactionHash}`
    )
    {
      // Check that the logs are correct
      const logs = ethFinalizeReceipt.logs
      for (const log of logs) {
        switch (log.address) {
          case L1StandardBridge.address: {
            const parsed = L1StandardBridge.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            if (parsed.name !== 'ETHBridgeFinalized') {
              throw new Error('Wrong event name from L1StandardBridge')
            }
            if (!parsed.args.amount.eq(utils.parseEther('1'))) {
              throw new Error('Wrong amount in event')
            }
            if (parsed.args.from !== address) {
              throw new Error('Wrong to in event')
            }
            if (parsed.args.to !== address) {
              throw new Error('Wrong from in event')
            }
            break
          }
          case L1CrossDomainMessenger.address: {
            const parsed = L1CrossDomainMessenger.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            if (parsed.name !== 'RelayedMessage') {
              throw new Error('Wrong event from L1CrossDomainMessenger')
            }
            break
          }
          case OptimismPortal.address: {
            const parsed = OptimismPortal.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            // TODO: remove extra event from contract
            if (parsed.name === 'WithdrawalFinalized') {
              if (parsed.args.success !== true) {
                throw new Error('Unsuccessful withdrawal call')
              }
            }
            break
          }
        }
      }
    }

    const opBalanceFinally = await signer.provider.getBalance(
      OptimismPortal.address
    )
    // TODO(tynes): fix this bug
    if (!opBalanceFinally.sub(utils.parseEther('1')).eq(opBalanceBefore)) {
      console.log('OptimismPortal balance mismatch')
      console.log(`Balance before deposit: ${opBalanceBefore.toString()}`)
      console.log(`Balance after deposit: ${opBalanceAfter.toString()}`)
      console.log(`Balance after withdrawal: ${opBalanceFinally.toString()}`)
      // throw new Error('OptimismPortal balance mismatch')
    }

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
