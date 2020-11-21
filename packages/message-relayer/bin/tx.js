const ethers = require('ethers')
const { getContractFactory } = require('@eth-optimism/contracts')

const main = async () => {
  const provider = new ethers.providers.JsonRpcProvider('http://18.222.50.191:8545')
  const wallet = new ethers.Wallet('0x' + '67'.repeat(64), provider)
  const factory = getContractFactory('OVM_L2CrossDomainMessenger',)
  const messenger = factory.attach('0x4200000000000000000000000000000000000007').connect(wallet)

  const transaction = await messenger.populateTransaction.sendMessage(
    '0x0000000000000000000000000000000000000004',
    '0x1234123412341234',
    2000000,
  )

  console.log(wallet.address)

  transaction.gasLimit = ethers.BigNumber.from(1000000)
  transaction.gasPrice = ethers.BigNumber.from(0)
  transaction.nonce = await provider.getTransactionCount(wallet.address)
  transaction.chainId = 420

  const signed = await wallet.signTransaction(transaction)
  const result = await provider.sendTransaction(signed)

  console.log(result, 'RESULT')

  const receipt = await result.wait()

  console.log(receipt, 'RECEIPT')
}

main()
