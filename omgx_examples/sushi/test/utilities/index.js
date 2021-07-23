const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { bob } = require('./wallet');

const BASE_TEN = 10
const ADDRESS_ZERO = "0x0000000000000000000000000000000000000000"

const gasOptions = {gasLimit: 800000, gasPrice: 0}

function encodeParameters(types, values) {
  const abi = new ethers.utils.AbiCoder()
  return abi.encode(types, values)
}

async function deploy(thisObject, contracts) {
  for (let i in contracts) {
    let contract = contracts[i]
    console.log(`Deploying ${contract[0]}...`)
    let Factory__contract = new ContractFactory(
      contract[1].abi,
      contract[1].bytecode,
      bob,
    )
    thisObject[contract[0]] = await Factory__contract.deploy(...(contract[2] || []), gasOptions)
    await thisObject[contract[0]].deployTransaction.wait()
  }
}

async function createSLP(thisObject, name, tokenA, tokenB, amount) {
  let transfer, mint
  // console.log(`Creating SLP ${name}...`)
  const createPairTx = await thisObject.factory.createPair(tokenA.address, tokenB.address, gasOptions)
  const pairTX = await createPairTx.wait()

  const _pair = pairTX.events[1].args.pair

  thisObject[name] = await thisObject.Factory__UniswapV2Pair.attach(_pair)

  transfer = await tokenA.transfer(thisObject[name].address, amount, gasOptions)
  await transfer.wait()
  transfer = await tokenB.transfer(thisObject[name].address, amount, gasOptions)
  await transfer.wait()

  mint = await thisObject[name].mint(bob.address, gasOptions)
  await mint.wait()
}

// Defaults to e18 using amount * 10^18
function getBigNumber(amount, decimals = 18) {
  return BigNumber.from(amount).mul(BigNumber.from(BASE_TEN).pow(decimals))
}

module.exports = {
  getBigNumber,
  createSLP,
  deploy,
  gasOptions,
  encodeParameters,
  BASE_TEN,
  ADDRESS_ZERO,
}