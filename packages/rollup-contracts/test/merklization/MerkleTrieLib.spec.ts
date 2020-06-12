import { expect } from '../setup';

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle';
import { Contract } from 'ethers';

import * as MerkleTrieLib from '../../build/MerkleTrieLib.json';
import { makeAllProofTests, makeRandomProofTests } from '../helpers/trie-helpers';

describe('BinaryMerkleTreeLib', () => {
  const [wallet] = getWallets(createMockProvider());
  let trie: Contract;
  beforeEach(async () => {
    trie = await deployContract(wallet, MerkleTrieLib);
  });

  describe('verifyInclusionProof', async () => {
    it(`should verify basic proofs`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '0123456789012345678901234567890123456789xx',
        },
        {
          key: 'key2bb',
          val: 'aval2',
        },
        {
          key: 'key3cc',
          val: 'aval3',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof))).to.equal(true);
      });
    });

    it(`should verify a single long key`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '0123456789012345678901234567890123456789xx',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof))).to.equal(true);
      });
    });

    it(`should verify a single short key`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '01234',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof))).to.equal(true);
      });
    });

    it(`should verify a keys in the middle`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '0123456789012345678901234567890123456789xxx',
        },
        {
          key: 'key1',
          val: '0123456789012345678901234567890123456789Very_Long',
        },
        {
          key: 'key2bb',
          val: 'aval3',
        },
        {
          key: 'key2',
          val: 'short',
        },
        {
          key: 'key3cc',
          val: 'aval3',
        },
        {
          key: 'key3',
          val: '1234567890123456789012345678901'
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof))).to.equal(true);
      });
    });
    
    it(`should verify with embedded extension nodes`, async () => {
      (await makeAllProofTests([
        {
          key: 'a',
          val: 'a',
        },
        {
          key: 'b',
          val: 'b',
        },
        {
          key: 'c',
          val: 'c',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof))).to.equal(true);
      });
    });

    it('should verify random data (128 nodes)', async () => {
      const test = await makeRandomProofTests('seed128', 128);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof)).to.equal(true);
    });

    it('should verify random data (256 nodes)', async () => {
      const test = await makeRandomProofTests('seed256', 256);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof)).to.equal(true);
    });

    it('should verify random data (512 nodes)', async () => {
      const test = await makeRandomProofTests('seed512', 512);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof)).to.equal(true);
    });

    it('should verify random data (1024 nodes)', async () => {
      const test = await makeRandomProofTests('seed1024', 1024);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof)).to.equal(true);
    });

    it('should verify random data (2048 nodes)', async () => {
      const test = await makeRandomProofTests('seed2048', 2048);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.root, test.proof)).to.equal(true);
    });
  });
});