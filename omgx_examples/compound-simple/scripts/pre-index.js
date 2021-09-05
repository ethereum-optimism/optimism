const { ethers } = require('ethers')
const {web3} = require('web3')

const sleep = (timeout) => {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      resolve()
    }, timeout)
  })
}

async function main(callback) {
  console.log('starting')
  // const accounts = await web3.eth.getAccounts()
  const GovernorBravoDelegate = artifacts.require('GovernorBravoDelegate')
  const governorBravoDelegate = await GovernorBravoDelegate.deployed()
  const GovernorBravoDelegator = artifacts.require('GovernorBravoDelegator')
  const governorBravoDelegator = await GovernorBravoDelegator.deployed()
  const Comp = artifacts.require('Comp')
  const comp = await Comp.deployed()
  const Timelock = artifacts.require('Timelock')
  const timelock = await Timelock.deployed()

  const GovernorBravo = await GovernorBravoDelegate.at(
    governorBravoDelegator.address
  )

  // Set Delegator as pending admin
  var provider = new ethers.providers.JsonRpcProvider(
    'https://rinkeby.boba.network',
    { chainId: 28 }
  )
  var blockNumber = await provider.getBlockNumber()
  var block = await provider.getBlock(blockNumber)
  var eta = block.timestamp + 1000
  var data = ethers.utils.defaultAbiCoder.encode(
    ['address'],
    [governorBravoDelegator.address]
  )
  // txHash = ethers.utils.keccak256(values)
  // console.log('txHash: ', txHash)
  // console.log('eta: ', eta)
  // queuedTransaction = await timelock.queuedTransactions(txHash)

  console.log('queuedTransaction :', queuedTransaction)
  timestamp = await timelock.exGetBlockTimestamp()
  console.log('timestamp', timestamp.toString())

  // await timelock.cancelTransaction(
  //   timelock.address,
  //   0,
  //   'setPendingAdmin(address)',
  //   data,
  //   1628744989
  // )
  
  console.log(
    '\n\n\n-----------------------------------------------------------\nQueuing setPendingAdmin'
  )

  await timelock.queueTransaction(
    timelock.address,
    0,
    'setPendingAdmin(address)',
    data,
    eta
  )
  console.log('queued')

  console.log('queued setPendingAdmin')

  sleep(2000 * 1000)

  await timelock.executeTransaction(
    timelock.address,
    0,
    'setPendingAdmin(address)',
    data,
    1628547326
  )

  console.log(
    '\n\n\n---------------------------------------------------\nqueueing initiate'
  )

  blockNumber = await provider.getBlockNumber()
  block = await provider.getBlock(blockNumber)
  eta = block.timestamp + 1000

  console.log(eta)

  await timelock.queueTransaction(
    GovernorBravo.address,
    0,
    '_initiate()',
    0,
    eta
  )

  await sleep(2000 * 1000)

  console.log('execute initiate')
  await timelock.executeTransaction(
    GovernorBravo.address,
    0,
    '_initiate()',
    0,
    1628549022
  )
  console.log('Executed initiate')
}

(async ()=> {
  main();
})();

module.exports = main;