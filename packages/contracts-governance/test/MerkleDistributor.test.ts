import chai, { expect } from 'chai'
import { solidity, MockProvider } from 'ethereum-waffle'
import { Contract, BigNumber, constants, Wallet } from 'ethers'
import { ethers } from 'hardhat'

import BalanceTree from '../src/balance-tree'
import { parseBalanceMap } from '../src/parse-balance-map'

chai.use(solidity)

const overrides = {
  gasLimit: 9999999,
}

const ZERO_BYTES32 =
  '0x0000000000000000000000000000000000000000000000000000000000000000'
const UNISWAP_MNEMONIC =
  'horn horn horn horn horn horn horn horn horn horn horn horn'

const isCoverage = process.env.IS_COVERAGE === 'true'

describe('MerkleDistributor', () => {
  let wallet0: Wallet
  let wallet1: Wallet
  let wallets: Wallet[]

  const deployContract = async (
    wallet: Wallet,
    name: string,
    args: any[],
    override: any
  ) => {
    const factory = await ethers.getContractFactory(name)
    const contract = await factory.deploy(...args, override)
    await contract.deployed()
    return contract
  }

  let token: Contract

  beforeEach('deploy token', async () => {
    wallets = []

    // Have to do this strange dance because the Unsiwap mnemonic is technically invalid.
    // Waffle ignores this, so to keep the tests the same we have to "import" the Waffle
    // wallets into Ethers.
    const mockProviders = new MockProvider({
      ganacheOptions: {
        hardfork: 'istanbul',
        mnemonic: UNISWAP_MNEMONIC,
        gasLimit: 9999999,
      },
    })
    const mockWallets = mockProviders.getWallets()

    const signer1 = (await ethers.getSigners())[0]
    for (let i = 0; i < 10; i++) {
      const wallet = new Wallet(mockWallets[i].privateKey)
      await signer1.sendTransaction({
        to: wallet.address,
        value: ethers.utils.parseEther('0.1'),
      })
      wallets.push(wallet)
    }

    wallet0 = wallets[0]
    wallet1 = wallets[1]
    token = await deployContract(
      wallet0,
      'TestERC20',
      ['Token', 'TKN', 0],
      overrides
    )
  })

  describe('#token', () => {
    it('returns the token address', async () => {
      const distributor = await deployContract(
        wallet0,
        'MerkleDistributor',
        [token.address, ZERO_BYTES32, wallet0.address],
        overrides
      )
      expect(await distributor.token()).to.eq(token.address)
    })
  })

  describe('#merkleRoot', () => {
    it('returns the zero merkle root', async () => {
      const distributor = await deployContract(
        wallet0,
        'MerkleDistributor',
        [token.address, ZERO_BYTES32, wallet0.address],
        overrides
      )
      expect(await distributor.merkleRoot()).to.eq(ZERO_BYTES32)
    })
  })

  describe('#claim', () => {
    it('fails for empty proof', async () => {
      const distributor = await deployContract(
        wallet0,
        'MerkleDistributor',
        [token.address, ZERO_BYTES32, wallet0.address],
        overrides
      )
      await expect(
        distributor.claim(0, wallet0.address, 10, [])
      ).to.be.revertedWith('MerkleDistributor: Invalid proof.')
    })

    it('fails for invalid index', async () => {
      const distributor = await deployContract(
        wallet0,
        'MerkleDistributor',
        [token.address, ZERO_BYTES32, wallet0.address],
        overrides
      )
      await expect(
        distributor.claim(0, wallet0.address, 10, [])
      ).to.be.revertedWith('MerkleDistributor: Invalid proof.')
    })

    describe('two account tree', () => {
      let distributor: Contract
      let tree: BalanceTree
      beforeEach('deploy', async () => {
        tree = new BalanceTree([
          { account: wallet0.address, amount: BigNumber.from(100) },
          { account: wallet1.address, amount: BigNumber.from(101) },
        ])
        distributor = await deployContract(
          wallet0,
          'MerkleDistributor',
          [token.address, tree.getHexRoot(), wallet0.address],
          overrides
        )
        await token.setBalance(distributor.address, 201)
      })

      it('successful claim', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        await expect(
          distributor.claim(0, wallet0.address, 100, proof0, overrides)
        )
          .to.emit(distributor, 'Claimed')
          .withArgs(0, wallet0.address, 100)
        const proof1 = tree.getProof(1, wallet1.address, BigNumber.from(101))
        await expect(
          distributor.claim(1, wallet1.address, 101, proof1, overrides)
        )
          .to.emit(distributor, 'Claimed')
          .withArgs(1, wallet1.address, 101)
      })

      it('transfers the token', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        expect(await token.balanceOf(wallet0.address)).to.eq(0)
        await distributor.claim(0, wallet0.address, 100, proof0, overrides)
        expect(await token.balanceOf(wallet0.address)).to.eq(100)
      })

      it('must have enough to transfer', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        await token.setBalance(distributor.address, 99)
        await expect(
          distributor.claim(0, wallet0.address, 100, proof0, overrides)
        ).to.be.revertedWith('ERC20: transfer amount exceeds balance')
      })

      it('sets #isClaimed', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        expect(await distributor.isClaimed(0)).to.eq(false)
        expect(await distributor.isClaimed(1)).to.eq(false)
        await distributor.claim(0, wallet0.address, 100, proof0, overrides)
        expect(await distributor.isClaimed(0)).to.eq(true)
        expect(await distributor.isClaimed(1)).to.eq(false)
      })

      it('cannot allow two claims', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        await distributor.claim(0, wallet0.address, 100, proof0, overrides)
        await expect(
          distributor.claim(0, wallet0.address, 100, proof0, overrides)
        ).to.be.revertedWith('MerkleDistributor: Drop already claimed.')
      })

      it('cannot claim more than once: 0 and then 1', async () => {
        await distributor.claim(
          0,
          wallet0.address,
          100,
          tree.getProof(0, wallet0.address, BigNumber.from(100)),
          overrides
        )
        await distributor.claim(
          1,
          wallet1.address,
          101,
          tree.getProof(1, wallet1.address, BigNumber.from(101)),
          overrides
        )

        await expect(
          distributor.claim(
            0,
            wallet0.address,
            100,
            tree.getProof(0, wallet0.address, BigNumber.from(100)),
            overrides
          )
        ).to.be.revertedWith('MerkleDistributor: Drop already claimed.')
      })

      it('cannot claim more than once: 1 and then 0', async () => {
        await distributor.claim(
          1,
          wallet1.address,
          101,
          tree.getProof(1, wallet1.address, BigNumber.from(101)),
          overrides
        )
        await distributor.claim(
          0,
          wallet0.address,
          100,
          tree.getProof(0, wallet0.address, BigNumber.from(100)),
          overrides
        )

        await expect(
          distributor.claim(
            1,
            wallet1.address,
            101,
            tree.getProof(1, wallet1.address, BigNumber.from(101)),
            overrides
          )
        ).to.be.revertedWith('MerkleDistributor: Drop already claimed.')
      })

      it('cannot claim for address other than proof', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        await expect(
          distributor.claim(1, wallet1.address, 101, proof0, overrides)
        ).to.be.revertedWith('MerkleDistributor: Invalid proof.')
      })

      it('cannot claim more than proof', async () => {
        const proof0 = tree.getProof(0, wallet0.address, BigNumber.from(100))
        await expect(
          distributor.claim(0, wallet0.address, 101, proof0, overrides)
        ).to.be.revertedWith('MerkleDistributor: Invalid proof.')
      })

      it('gas', async function () {
        if (isCoverage) {
          this.skip()
        }
        const proof = tree.getProof(0, wallet0.address, BigNumber.from(100))
        const tx = await distributor.claim(
          0,
          wallet0.address,
          100,
          proof,
          overrides
        )
        const receipt = await tx.wait()
        expect(receipt.gasUsed).to.eq(84480) // old 78466
      })
    })
    describe('larger tree', () => {
      let distributor: Contract
      let tree: BalanceTree
      beforeEach('deploy', async () => {
        tree = new BalanceTree(
          wallets.map((wallet, ix) => {
            return { account: wallet.address, amount: BigNumber.from(ix + 1) }
          })
        )
        distributor = await deployContract(
          wallet0,
          'MerkleDistributor',
          [token.address, tree.getHexRoot(), wallet0.address],
          overrides
        )
        await token.setBalance(distributor.address, 201)
      })

      it('claim index 4', async () => {
        const proof = tree.getProof(4, wallets[4].address, BigNumber.from(5))
        await expect(
          distributor.claim(4, wallets[4].address, 5, proof, overrides)
        )
          .to.emit(distributor, 'Claimed')
          .withArgs(4, wallets[4].address, 5)
      })

      it('claim index 9', async () => {
        const proof = tree.getProof(9, wallets[9].address, BigNumber.from(10))
        await expect(
          distributor.claim(9, wallets[9].address, 10, proof, overrides)
        )
          .to.emit(distributor, 'Claimed')
          .withArgs(9, wallets[9].address, 10)
      })

      it('gas', async function () {
        if (isCoverage) {
          this.skip()
        }
        const proof = tree.getProof(9, wallets[9].address, BigNumber.from(10))
        const tx = await distributor.claim(
          9,
          wallets[9].address,
          10,
          proof,
          overrides
        )
        const receipt = await tx.wait()
        expect(receipt.gasUsed).to.eq(87237) // old 80960
      })

      it('gas second down about 15k', async function () {
        if (isCoverage) {
          this.skip()
        }
        await distributor.claim(
          0,
          wallets[0].address,
          1,
          tree.getProof(0, wallets[0].address, BigNumber.from(1)),
          overrides
        )
        const tx = await distributor.claim(
          1,
          wallets[1].address,
          2,
          tree.getProof(1, wallets[1].address, BigNumber.from(2)),
          overrides
        )
        const receipt = await tx.wait()
        expect(receipt.gasUsed).to.eq(70117) // old 65940
      })
    })

    describe('realistic size tree', () => {
      let distributor: Contract
      let tree: BalanceTree
      const NUM_LEAVES = 100_000
      const NUM_SAMPLES = 25
      const elements: { account: string; amount: BigNumber }[] = []

      before(() => {
        for (let i = 0; i < NUM_LEAVES; i++) {
          const node = { account: wallet0.address, amount: BigNumber.from(100) }
          elements.push(node)
        }
        tree = new BalanceTree(elements)
      })

      it('proof verification works', () => {
        const root = Buffer.from(tree.getHexRoot().slice(2), 'hex')
        for (let i = 0; i < NUM_LEAVES; i += NUM_LEAVES / NUM_SAMPLES) {
          const proof = tree
            .getProof(i, wallet0.address, BigNumber.from(100))
            .map((el) => Buffer.from(el.slice(2), 'hex'))
          const validProof = BalanceTree.verifyProof(
            i,
            wallet0.address,
            BigNumber.from(100),
            proof,
            root
          )
          expect(validProof).to.be.true
        }
      })

      beforeEach('deploy', async () => {
        distributor = await deployContract(
          wallet0,
          'MerkleDistributor',
          [token.address, tree.getHexRoot(), wallet0.address],
          overrides
        )
        await token.setBalance(distributor.address, constants.MaxUint256)
      })

      it('gas', async function () {
        if (isCoverage) {
          this.skip()
        }
        const proof = tree.getProof(50000, wallet0.address, BigNumber.from(100))
        const tx = await distributor.claim(
          50000,
          wallet0.address,
          100,
          proof,
          overrides
        )
        const receipt = await tx.wait()
        expect(receipt.gasUsed).to.eq(99061) // old 91650
      })
      it('gas deeper node', async function () {
        if (isCoverage) {
          this.skip()
        }
        const proof = tree.getProof(90000, wallet0.address, BigNumber.from(100))
        const tx = await distributor.claim(
          90000,
          wallet0.address,
          100,
          proof,
          overrides
        )
        const receipt = await tx.wait()
        expect(receipt.gasUsed).to.eq(98997) // old 91586
      })
      it('gas average random distribution', async function () {
        if (isCoverage) {
          this.skip()
        }
        let total: BigNumber = BigNumber.from(0)
        let count: number = 0
        for (let i = 0; i < NUM_LEAVES; i += NUM_LEAVES / NUM_SAMPLES) {
          const proof = tree.getProof(i, wallet0.address, BigNumber.from(100))
          const tx = await distributor.claim(
            i,
            wallet0.address,
            100,
            proof,
            overrides
          )
          const receipt = await tx.wait()
          total = total.add(receipt.gasUsed)
          count++
        }
        const average = total.div(count)
        expect(average).to.eq(82453) // old 77075
      })
      // this is what we gas golfed by packing the bitmap
      it('gas average first 25', async function () {
        if (isCoverage) {
          this.skip()
        }
        let total: BigNumber = BigNumber.from(0)
        let count: number = 0
        for (let i = 0; i < 25; i++) {
          const proof = tree.getProof(i, wallet0.address, BigNumber.from(100))
          const tx = await distributor.claim(
            i,
            wallet0.address,
            100,
            proof,
            overrides
          )
          const receipt = await tx.wait()
          total = total.add(receipt.gasUsed)
          count++
        }
        const average = total.div(count)
        expect(average).to.eq(66203) // old 62824
      })

      it('no double claims in random distribution', async () => {
        for (
          let i = 0;
          i < 25;
          i += Math.floor(Math.random() * (NUM_LEAVES / NUM_SAMPLES))
        ) {
          const proof = tree.getProof(i, wallet0.address, BigNumber.from(100))
          await distributor.claim(i, wallet0.address, 100, proof, overrides)
          await expect(
            distributor.claim(i, wallet0.address, 100, proof, overrides)
          ).to.be.revertedWith('MerkleDistributor: Drop already claimed.')
        }
      })
    })
  })

  describe('parseBalanceMap', () => {
    let distributor: Contract
    let claims: {
      [account: string]: {
        index: number
        amount: string
        proof: string[]
      }
    }
    beforeEach('deploy', async () => {
      const {
        claims: innerClaims,
        merkleRoot,
        tokenTotal,
      } = parseBalanceMap({
        [wallet0.address]: 200,
        [wallet1.address]: '300', // add a string one to verify that the hex cast works
        [wallets[2].address]: 250,
      })
      expect(tokenTotal).to.eq('0x02ee') // 750
      claims = innerClaims
      distributor = await deployContract(
        wallet0,
        'MerkleDistributor',
        [token.address, merkleRoot, wallet0.address],
        overrides
      )
      await token.setBalance(distributor.address, tokenTotal)
    })

    it('check the proofs is as expected', () => {
      expect(claims).to.deep.eq({
        [wallet0.address]: {
          index: 0,
          amount: '0xc8',
          proof: [
            '0x2a411ed78501edb696adca9e41e78d8256b61cfac45612fa0434d7cf87d916c6',
          ],
        },
        [wallet1.address]: {
          index: 1,
          amount: '0x012c',
          proof: [
            '0xbfeb956a3b705056020a3b64c540bff700c0f6c96c55c0a5fcab57124cb36f7b',
            '0xd31de46890d4a77baeebddbd77bf73b5c626397b73ee8c69b51efe4c9a5a72fa',
          ],
        },
        [wallets[2].address]: {
          index: 2,
          amount: '0xfa',
          proof: [
            '0xceaacce7533111e902cc548e961d77b23a4d8cd073c6b68ccf55c62bd47fc36b',
            '0xd31de46890d4a77baeebddbd77bf73b5c626397b73ee8c69b51efe4c9a5a72fa',
          ],
        },
      })
    })

    it('all claims work exactly once', async () => {
      for (const account of Object.keys(claims)) {
        const claim = claims[account]
        await expect(
          distributor.claim(
            claim.index,
            account,
            claim.amount,
            claim.proof,
            overrides
          )
        )
          .to.emit(distributor, 'Claimed')
          .withArgs(claim.index, account, claim.amount)
        await expect(
          distributor.claim(
            claim.index,
            account,
            claim.amount,
            claim.proof,
            overrides
          )
        ).to.be.revertedWith('MerkleDistributor: Drop already claimed.')
      }
      expect(await token.balanceOf(distributor.address)).to.eq(0)
    })
  })
})
