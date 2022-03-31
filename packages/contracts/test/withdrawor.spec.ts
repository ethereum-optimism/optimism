import { expect } from 'chai'
import hre from 'hardhat'
import { Signer, BigNumber, Wallet, providers, ethers, utils } from 'ethers'
import { Withdrawor, Withdrawor__factory } from '../typechain'
import { toRpcHexString } from '@eth-optimism/core-utils'

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

const encodeWithdrawal = (args: {
  nonce: BigNumber | number
  sender: string
  target: string
  value: BigNumber | number
  gasLimit: BigNumber | number
  data: string
}): string => {
  return ethers.utils.defaultAbiCoder.encode(
    ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes'],
    [
      utils.hexZeroPad(BigNumber.from(args.nonce).toHexString(), 32),
      args.sender,
      args.target,
      utils.hexZeroPad(BigNumber.from(args.value).toHexString(), 32),
      utils.hexZeroPad(BigNumber.from(args.gasLimit).toHexString(), 32),
      args.data,
    ]
  )
}

describe('Withdraw', () => {
  let wallet: Wallet
  let signer: Signer
  let signerAddress: string
  let withdrawor: Withdrawor
  before('Setup withdrawor contract', async () => {
    wallet = new Wallet(process.env.PRIVATE_KEY!)
    signer = wallet.connect(l2GethProvider)
    signerAddress = await signer.getAddress()

    withdrawor = await new Withdrawor__factory(signer).attach(withdraworAddress)
  })

  describe('Creating a withdrawal', () => {
    let withdrawalHash: string
    let nonceBefore: BigNumber
    let storageSlot: string
    before(async () => {
      nonceBefore = await withdrawor.nonce()

      await (
        await withdrawor.initiateWithdrawal(
          NON_ZERO_ADDRESS,
          NON_ZERO_GASLIMIT,
          NON_ZERO_DATA
        )
      ).wait()

      // locally calculate the expected mapping key
      withdrawalHash = ethers.utils.keccak256(
        encodeWithdrawal({
          nonce: nonceBefore,
          sender: signerAddress,
          target: NON_ZERO_ADDRESS,
          value: 0,
          gasLimit: NON_ZERO_GASLIMIT,
          data: NON_ZERO_DATA,
        })
      )

      // locally calculate the expected storage slot in the contract
      storageSlot = ethers.utils.keccak256(
        withdrawalHash + '00'.repeat(31) + '01'
      )
    })

    it('Should add an entry to the withdrawals mapping', async () => {
      const nonceAfter = await withdrawor.nonce()
      expect(await withdrawor.withdrawals(withdrawalHash)).to.be.true
      expect(nonceAfter.sub(nonceBefore).toNumber()).to.eq(1)
    })

    it('Should return bytes32(1) when querying the calculated storage slot', async () => {
      // Test to ensure we're correctly calculating the slot, per the solidity docs:
      // "The value corresponding to a mapping key k is located at keccak256(h(k) . p) where . is
      //   concatenation and h is a function that is applied to the key..."
      expect(
        await l2GethProvider.getStorageAt(withdraworAddress, storageSlot)
      ).to.equal(utils.hexZeroPad('0x01', 32))
    })

    it('should generate a proof', async () => {
      const proof = await l2GethProvider.send('eth_getProof', [
        withdraworAddress,
        [storageSlot],
        toRpcHexString((await l2GethProvider.getBlock('latest')).number),
      ])
      expect(proof.storageProof[0].key).to.eq(storageSlot)
    })
  })
})
