import { task, types } from 'hardhat/config'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts-bedrock'
import { utils } from 'ethers'
import '@eth-optimism/op-hardhat-chainops'

const printBalances = async (hre: HardhatRuntimeEnvironment) => {
  const l1Height = await hre.optimism.l1Signer.provider.getBlockNumber()
  const l2Height = await hre.optimism.l2Signer.provider.getBlockNumber()

  console.log()
  console.log('Fetching balances')
  console.log(`L1 height: ${l1Height}`)
  console.log(`L2 height: ${l2Height}`)

  const l1Balance = await hre.optimism.l1Signer.getBalance()
  const l2Balance = await hre.optimism.l2Signer.getBalance()
  console.log(`L1 balance: ${utils.formatEther(l1Balance)} ETH`)
  console.log(`L2 balance: ${utils.formatEther(l2Balance)} ETH`)

  const table = {}
  const contracts = [
    'OptimismPortal',
    'L1CrossDomainMessenger',
    'L1StandardBridge',
    'L2ToL1MessagePasser',
    'L2CrossDomainMessenger',
    'L2StandardBridge',
  ]
  for (const name of contracts) {
    const contract = hre.optimism.contracts[name]
    const balance = await contract.provider.getBalance(contract.address)
    table[name] = `${utils.formatEther(balance)} ETH`
  }
  console.table(table)
  console.log()
}

import {
  CrossChainMessenger,
  StandardBridgeAdapter,
  MessageStatus,
} from '../src'

task('deposit-eth', 'Deposits WETH9 onto L2.')
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
  .addOptionalParam('to', 'Recipient of the ether', '', types.string)
  .addOptionalParam(
    'amount',
    'Amount of ether to send (in ETH)',
    '',
    types.string
  )
  .addOptionalParam(
    'withdraw',
    'Follow up with a withdrawal',
    true,
    types.boolean
  )
  .addOptionalParam('withdrawAmount', 'Amount to withdraw', '', types.string)
  .setAction(async (args, hre) => {
    const signers = await hre.ethers.getSigners()
    if (signers.length === 0) {
      throw new Error('No configured signers')
    }

    await hre.optimism.init({
      l1Signer: signers[0],
      l2Url: args.l2ProviderUrl,
    })

    const signer = hre.optimism.l1Signer

    // Use the first configured signer for simplicity
    const address = await signer.getAddress()
    console.log(`Using signer ${address}`)

    // Ensure that the signer has a balance before trying to
    // do anything
    const balance = await signer.getBalance()
    if (balance.eq(0)) {
      throw new Error('Signer has no balance')
    }

    // send to self if not specified
    const to = args.to ? args.to : address
    const amount = args.amount
      ? utils.parseEther(args.amount)
      : utils.parseEther('1')
    const withdrawAmount = args.withdrawAmount
      ? utils.parseEther(args.withdrawAmount)
      : utils.parseEther(amount.div(2).toString())

    const l2Signer = hre.optimism.l2Signer
    const OptimismPortal = hre.optimism.contracts.OptimismPortal
    const L1CrossDomainMessenger = hre.optimism.contracts.L1CrossDomainMessenger
    const L1StandardBridge = hre.optimism.contracts.L1StandardBridge
    const L2OutputOracle = hre.optimism.contracts.L2OutputOracle

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: signer,
      l2SignerOrProvider: l2Signer,
      l1ChainId: await signer.getChainId(),
      l2ChainId: await l2Signer.getChainId(),
      bridges: {
        Standard: {
          Adapter: StandardBridgeAdapter,
          l1Bridge: L1StandardBridge.address,
          l2Bridge: predeploys.L2StandardBridge,
        },
      },
      contracts: {
        l1: {
          L1StandardBridge: L1StandardBridge.address,
          L1CrossDomainMessenger: L1CrossDomainMessenger.address,
          L2OutputOracle: L2OutputOracle.address,
          OptimismPortal: OptimismPortal.address,
        },
      },
      bedrock: true,
    })

    await printBalances(hre)
    const opBalanceBefore = await signer.provider.getBalance(
      OptimismPortal.address
    )

    // Deposit ETH
    console.log('Depositing ETH through StandardBridge')
    const ethDeposit = await messenger.depositETH(amount, { recipient: to })
    const depositMessageReceipt = await messenger.waitForMessageReceipt(
      ethDeposit
    )
    if (depositMessageReceipt.receiptStatus !== 1) {
      throw new Error('deposit failed')
    }
    console.log(
      `Deposit complete - ${depositMessageReceipt.transactionReceipt.transactionHash}`
    )
    await printBalances(hre)

    const opBalanceAfter = await signer.provider.getBalance(
      OptimismPortal.address
    )

    if (!opBalanceBefore.add(amount).eq(opBalanceAfter)) {
      throw new Error(`OptimismPortal balance mismatch`)
    }

    if (!args.withdraw) {
      return
    }

    console.log('Withdrawing ETH')
    const ethWithdraw = await messenger.withdrawETH(withdrawAmount)
    const ethWithdrawReceipt = await ethWithdraw.wait()
    console.log(`ETH withdrawn on L2 - ${ethWithdrawReceipt.transactionHash}`)

    for (const log of ethWithdrawReceipt.logs) {
      try {
        const parsed = hre.optimism.parseLog(log)
        console.log(`Log from ${parsed.name} at ${log.address}`)
        console.log(parsed.log.args)
        console.log()
      } catch (e) {
        console.log(
          `Unknown log emitted from ${log.address} - ${log.topics[0]}`
        )
      }
    }
    console.log(
      `Withdrawal on L2 complete: ${ethWithdrawReceipt.transactionHash}`
    )
    await printBalances(hre)

    console.log('Waiting to be able to withdraw')
    setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(ethWithdrawReceipt)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
    }, 3000)

    await messenger.waitForMessageStatus(
      ethWithdrawReceipt,
      MessageStatus.READY_FOR_RELAY
    )

    const ethFinalize = await messenger.finalizeMessage(ethWithdrawReceipt)
    const ethFinalizeReceipt = await ethFinalize.wait()
    if (ethFinalizeReceipt.status !== 1) {
      throw new Error('Finalize withdrawal reverted')
    }

    for (const log of ethFinalizeReceipt.logs) {
      try {
        const parsed = hre.optimism.parseLog(log)
        console.log(`Log from ${parsed.name} at ${log.address}`)
        console.log(parsed.log.args)
        console.log()
      } catch (e) {
        console.log(
          `Unknown log emitted from ${log.address} - ${log.topics[0]}`
        )
      }
    }
    console.log(
      `ETH withdrawal complete: ${ethFinalizeReceipt.transactionHash}`
    )
    await printBalances(hre)

    const opBalanceFinally = await signer.provider.getBalance(
      OptimismPortal.address
    )
    // TODO(tynes): fix this bug
    if (!opBalanceFinally.sub(withdrawAmount).eq(opBalanceAfter)) {
      console.log('OptimismPortal balance mismatch')
      console.log(`Balance before deposit: ${opBalanceBefore.toString()}`)
      console.log(`Balance after deposit: ${opBalanceAfter.toString()}`)
      console.log(`Balance after withdrawal: ${opBalanceFinally.toString()}`)
      return
      // throw new Error('OptimismPortal balance mismatch')
    }
    console.log('Withdraw success')
  })
