import { expect } from 'chai'
import hre from 'hardhat'
import { Signer, BigNumber, Wallet, providers, ethers } from 'ethers'
import { Withdrawor, Withdrawor__factory } from '../typechain'

// not secret. Don't use for real shit. Don't submit bug reports if you find this in git.
// 0xf3c101f4e376e7994E78CFB13a3A7e4B40910983
const l2GethProvider = new providers.JsonRpcProvider('http://localhost:9545')
const l1GethProvider = new providers.JsonRpcProvider('http://localhost:8545')
const withdraworAddress = '0x4200000000000000000000000000000000000015'
const ZERO_ADDRESS = '0x' + '00'.repeat(20)
const ZERO_BIGNUMBER = BigNumber.from(0)
const ZERO_BYTES32 = '0x' + '00'.repeat(32)
const NON_ZERO_ADDRESS = '0x' + '11'.repeat(20)
const NON_ZERO_GASLIMIT = BigNumber.from(50_000)
const NON_ZERO_VALUE = BigNumber.from(100)
const NON_ZERO_DATA = '0x' + '11'.repeat(42)

if (!process.env.PRIVATE_KEY) {
  throw new Error('You must define PRIVATE_KEY in your environment.')
}

describe('Withdraw', () => {
  let wallet: Wallet
  let signer: Signer
  let signerAddress: string
  let withdrawor: Withdrawor
  before(async () => {
    wallet = new Wallet(process.env.PRIVATE_KEY!)
    signer = wallet.connect(l2GethProvider)
    signerAddress = await signer.getAddress()

    withdrawor = await new Withdrawor__factory(signer).attach(withdraworAddress)
  })

  it('Should create a withdrawal', async () => {
    const nonceBefore = await withdrawor.nonce()

    const tx = await withdrawor.initiateWithdrawal(
      NON_ZERO_ADDRESS,
      NON_ZERO_GASLIMIT,
      NON_ZERO_DATA
    )
    await tx.wait()

    const messageHash = ethers.utils.keccak256(
      ethers.utils.defaultAbiCoder.encode(
        ['uint256', 'address', 'address', 'uint256', 'bytes'],
        [
          nonceBefore,
          signerAddress,
          NON_ZERO_ADDRESS,
          ZERO_BYTES32,
          NON_ZERO_DATA,
        ]
      )
    )

    const nonceAfter = await withdrawor.nonce()
    expect(await withdrawor.withdrawals(messageHash)).to.be.true
    expect(nonceAfter.sub(nonceBefore).toNumber()).to.eq(1)
  })
})
