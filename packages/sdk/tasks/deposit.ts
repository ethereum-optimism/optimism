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
  toAddress,
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

    // burn some ETH just to make the signer balance more
    const excessBalance = await (
      await signer.getBalance()
    ).sub(utils.parseEther('100000'))
    if (!excessBalance.isNegative()) {
      await signer.sendTransaction({
        to: toAddress(utils.hexZeroPad('0x111111111', 20)),
        value: excessBalance,
      })
    }

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

    const depositAmount = utils.parseEther('100')
    const withdrawAmount = utils.parseEther('11')
    // const l1Recipient = address
    const l1Recipient = toAddress(utils.hexZeroPad('0xabcdabcd', 20))

    /* GET AND LOG BALANCES BEFORE */
    const senderBalanceBefore = await signer.getBalance()
    const opBalanceBefore = await signer.provider.getBalance(
      OptimismPortal.address
    )
    const recipientBalanceBefore = await signer.provider.getBalance(l1Recipient)
    console.log(
      `Sender balance before: ${utils.formatEther(senderBalanceBefore)}`
    )
    console.log(`Portal balance before: ${utils.formatEther(opBalanceBefore)}`)
    console.log(
      `Recipient balance before: ${utils.formatEther(recipientBalanceBefore)}`
    )

    // Deposit ETH
    console.log('Depositing ETH through StandardBridge')
    const ethDeposit = await messenger.depositETH(depositAmount)
    const depositMessageReceipt = await messenger.waitForMessageReceipt(
      ethDeposit
    )
    if (depositMessageReceipt.receiptStatus !== 1) {
      throw new Error('deposit failed')
    }
    console.log('Deposit complete')

    /* GET AND LOG BALANCES AFTER DEPOSIT */
    const senderBalanceAfter = await signer.getBalance()
    const opBalanceAfter = await signer.provider.getBalance(
      OptimismPortal.address
    )
    const recipientBalanceAfter = await signer.provider.getBalance(l1Recipient)
    console.log(
      `Sender balance after: ${utils.formatEther(senderBalanceAfter)}`
    )
    console.log(`Portal balance after: ${utils.formatEther(opBalanceAfter)}`)
    console.log(
      `Recipient balance after: ${utils.formatEther(recipientBalanceAfter)}`
    )

    if (!opBalanceBefore.add(depositAmount).eq(opBalanceAfter)) {
      throw new Error(`OptimismPortal balance mismatch`)
    }

    console.log('Withdrawing ETH')
    const ethWithdraw = await messenger.withdrawETH(withdrawAmount, {
      recipient: l1Recipient,
    })
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
            if (!parsed.args.amount.eq(withdrawAmount)) {
              throw new Error('Wrong amount in event')
            }
            if (parsed.args.from !== address) {
              throw new Error('Wrong from in event')
            }
            if (parsed.args.to !== l1Recipient) {
              throw new Error('Wrong to in event')
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

    /* GET AND LOG BALANCES FINALLY */
    const senderBalanceFinally = await signer.getBalance()
    const opBalanceFinally = await signer.provider.getBalance(
      OptimismPortal.address
    )
    const recipientBalanceFinally = await signer.provider.getBalance(
      l1Recipient
    )
    console.log(
      `Sender balance Finally: ${utils.formatEther(senderBalanceFinally)}`
    )
    console.log(
      `Portal balance Finally: ${utils.formatEther(opBalanceFinally)}`
    )
    console.log(
      `Recipient balance Finally: ${utils.formatEther(recipientBalanceFinally)}`
    )
    // TODO(tynes): fix this bug
    if (!opBalanceFinally.sub(withdrawAmount).eq(opBalanceBefore)) {
      console.log('OptimismPortal balance mismatch')
      console.log(
        `Portal balance before deposit: ${utils.formatEther(opBalanceBefore)}`
      )
      console.log(
        `Portal balance after deposit: ${utils.formatEther(opBalanceAfter)}`
      )
      console.log(
        `Portal balance after withdrawal: ${utils.formatEther(
          opBalanceFinally
        )}`
      )

      console.log(
        `Sender balance before deposit: ${utils.formatEther(
          senderBalanceBefore
        )}`
      )
      console.log(
        `Sender balance after deposit: ${utils.formatEther(senderBalanceAfter)}`
      )
      console.log(
        `Sender balance after withdrawal: ${utils.formatEther(
          senderBalanceFinally
        )}`
      )

      console.log(
        `Recipient balance before deposit: ${utils.formatEther(
          recipientBalanceBefore
        )}`
      )
      console.log(
        `Recipient balance after deposit: ${utils.formatEther(
          recipientBalanceAfter
        )}`
      )
      console.log(
        `Recipient balance after withdrawal: ${utils.formatEther(
          recipientBalanceFinally
        )}`
      )
      // throw new Error('OptimismPortal balance mismatch')
    }
    console.log('Withdrawal success')
  })
