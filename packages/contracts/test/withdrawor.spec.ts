import { expect } from 'chai'
import { Signer, BigNumber, Wallet, providers, ethers, utils } from 'ethers'
import {
  Withdrawor,
  Withdrawor__factory,
  TestLibSecureMerkleTrie,
  TestLibSecureMerkleTrie__factory,
  WithdrawalVerifier,
  WithdrawalVerifier__factory,
  L2OutputOracle,
  L2OutputOracle__factory,
} from '../typechain'
import { toRpcHexString, toHexString, Address } from '@eth-optimism/core-utils'
import * as rlp from 'rlp'

const l2GethProvider = new providers.JsonRpcProvider('http://localhost:9545')
const l1GethProvider = new providers.JsonRpcProvider('http://localhost:8545')

const withdraworAddress = '0x4200000000000000000000000000000000000015'

const NON_ZERO_ADDRESS = '0x' + '11'.repeat(20)
const NON_ZERO_GASLIMIT = BigNumber.from(50_000)
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
  let signerAddress: string
  let l1Signer: Signer
  let l2Signer: Signer
  let withdrawor: Withdrawor
  let testLibSecureMerkleTrie: TestLibSecureMerkleTrie
  let proof: any
  let nonceBefore: BigNumber

  before('Setup L2 withdrawor contract', async () => {
    wallet = new Wallet(process.env.PRIVATE_KEY!)
    signerAddress = await wallet.getAddress()
    l1Signer = wallet.connect(l1GethProvider)
    l2Signer = wallet.connect(l2GethProvider)

    withdrawor = await new Withdrawor__factory(l2Signer).attach(
      withdraworAddress
    )

    testLibSecureMerkleTrie = await (
      await new TestLibSecureMerkleTrie__factory(l2Signer)
    ).deploy()
  })

  describe('Creating a withdrawal on L2', () => {
    let withdrawalHash: string
    let storageKey: string
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
          sender: await l2Signer.getAddress(),
          target: NON_ZERO_ADDRESS,
          value: 0,
          gasLimit: NON_ZERO_GASLIMIT,
          data: NON_ZERO_DATA,
        })
      )
    })

    it('Should add an entry to the withdrawals mapping', async () => {
      const nonceAfter = await withdrawor.nonce()
      expect(await withdrawor.withdrawals(withdrawalHash)).to.be.true
      expect(nonceAfter.sub(nonceBefore).toNumber()).to.eq(1)
    })

    // Test to ensure we're correctly calculating the storageKey. Per the solidity docs:
    // "The value corresponding to a mapping key k is located at keccak256(h(k) . p) where . is
    //   concatenation and h is a function that is applied to the key..."
    it('Should return bytes32(1) when querying the calculated storage key', async () => {
      const storageSlot = '00'.repeat(31) + '01' // i.e the second variable declared in the contract
      storageKey = ethers.utils.keccak256(withdrawalHash + storageSlot)

      expect(
        await l2GethProvider.getStorageAt(withdraworAddress, storageKey)
      ).to.equal(utils.hexZeroPad('0x01', 32))
    })

    it('should generate a valid proof', async () => {
      // Get the proof
      proof = await l2GethProvider.send('eth_getProof', [
        withdraworAddress,
        [storageKey],
        toRpcHexString((await l2GethProvider.getBlock('latest')).number),
      ])

      // Sanity check expected values of the proof
      expect(proof.storageProof[0].key).to.eq(storageKey)
      expect(proof.storageProof[0].value).to.eq('0x1')

      // Check the proof directly against the SecureMerkleTrie lib
      expect(
        await testLibSecureMerkleTrie.verifyInclusionProof(
          proof.storageProof[0].key,
          '0x01',
          toHexString(rlp.encode(proof.storageProof[0].proof)),
          proof.storageHash
        )
      ).to.be.true
    })
  })

  describe('Finalizing a withdrawal on L1', () => {
    let withdrawalVerifier: WithdrawalVerifier
    let l2OutputOracle: L2OutputOracle
    let l2Timestamp: BigNumber
    before('Setup L1 system contracts', async () => {
      // Deploy the L2 Output Oracle with a 1 second submission interval so that we can immediately
      // test the withdrawal.
      l2OutputOracle = await (
        await new L2OutputOracle__factory(l1Signer)
      ).deploy(
        1, // submissionInterval
        1, // l2BlockTime
        ethers.utils.keccak256('0x00'), // genesisL2Output
        100, // historicalTotalBlocks
        signerAddress
      )

      // Deploy the WithdrawalVerifier with a 0 second finalization delay.
      withdrawalVerifier = await (
        await new WithdrawalVerifier__factory(l1Signer)
      ).deploy(l2OutputOracle.address, withdraworAddress, 0)

      // create an output root that we can prove against only the storage root matters
      // for our purposes.
      const outputRoot = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            proof.storageHash,
            ethers.constants.HashZero,
          ]
        )
      )

      l2Timestamp = await l2OutputOracle.nextTimestamp()
      await l2OutputOracle.appendL2Output(
        outputRoot,
        l2Timestamp,
        ethers.constants.HashZero,
        0
      )
    })
    it('should successfully verify the withdrawal on L1', async () => {
      const tx = await withdrawalVerifier.verifyWithdrawal(
        nonceBefore,
        signerAddress,
        NON_ZERO_ADDRESS,
        0,
        NON_ZERO_GASLIMIT,
        NON_ZERO_DATA,
        {
          timestamp: l2Timestamp,
          version: ethers.constants.HashZero,
          stateRoot: ethers.constants.HashZero,
          withdrawerStorageRoot: proof.storageHash,
          latestBlockhash: ethers.constants.HashZero,
        },
        toHexString(rlp.encode(proof.storageProof[0].proof)),
        {
          gasLimit: 1_000_000,
        }
      )
      const events: any = (await tx.wait()).events
      expect(events[0].event).to.equal('WithdrawalVerified')
    })
  }).timeout(30000)
})
