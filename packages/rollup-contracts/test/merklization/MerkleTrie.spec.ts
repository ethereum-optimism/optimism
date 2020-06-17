import { expect } from '../setup';

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle';
import { Contract } from 'ethers';

import * as MerkleTrie from '../../build/MerkleTrie.json';
import { makeAllProofTests, makeRandomProofTests, makeTrie, makeProofTest } from '../helpers/trie-helpers';
import { ExtensionNode, LeafNode, BranchNode } from 'merkle-patricia-tree/dist/trieNode';
import { BaseTrie } from 'merkle-patricia-tree';
import * as rlp from 'rlp';

describe('MerkleTrie', () => {
  const [wallet] = getWallets(createMockProvider());
  let trie: Contract;
  beforeEach(async () => {
    trie = await deployContract(wallet, MerkleTrie);
  });

  /*
  describe('updateTrieRoot', async () => {
    it(`should handle a basic leaf update`, async () => {
      const nodes = [
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
      ];
      const t = await makeTrie(nodes);
      const test = await makeProofTest(nodes, nodes[0].key, t);
      await t.put(Buffer.from(nodes[0].key), Buffer.from('supervalue'));
      expect(await trie.updateTrieRoot(test.key, '0x' + Buffer.from('supervalue').toString('hex'), test.proof)).to.equal('0x' + t.root.toString('hex'));
    });

    it(`should handle a new leaf`, async () => {
      const nodes = [
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
      ];
      const t = await makeTrie(nodes);
      const proof = await BaseTrie.prove(t, Buffer.from('key4dd'));
      await t.put(Buffer.from('key4dd'), Buffer.from('supervalue'));
      const test = {
        key: '0x' + Buffer.from('key4dd').toString('hex'),
        val: '0x' + Buffer.from('supervalue').toString('hex'),
        proof: '0x' + rlp.encode(proof).toString('hex')
      }
      expect(await trie.updateTrieRoot(test.key, test.val, test.proof)).to.equal('0x' + t.root.toString('hex'));
    });

    it(`should handle a new leaf modifying extension`, async () => {
      const nodes = [
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
      ];
      const t = await makeTrie(nodes);
      const proof = await BaseTrie.prove(t, Buffer.from('key1ab'));
      await t.put(Buffer.from('key1ab'), Buffer.from('supervalue'));
      const s = await t.findPath(Buffer.from('key1ab'));
      for (const a of s.stack) {
        console.log(a.serialize().toString('hex'));
      }
      const test = {
        key: '0x' + Buffer.from('key1ab').toString('hex'),
        val: '0x' + Buffer.from('supervalue').toString('hex'),
        proof: '0x' + rlp.encode(proof).toString('hex')
      }
      console.log(test.key + ',', test.val + ',', test.proof)
      expect(await trie.updateTrieRoot(test.key, test.val, test.proof)).to.equal('0x' + t.root.toString('hex'));
    });
  });
  */

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
        expect((await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
      });
    });

    it(`should verify a single long key`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '0123456789012345678901234567890123456789xx',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
      });
    });

    it(`should verify a single short key`, async () => {
      (await makeAllProofTests([
        {
          key: 'key1aa',
          val: '01234',
        },
      ])).forEach(async (test, idx) => {
        expect((await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
      });
    });

    it(`should verify a key in the middle`, async () => {
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
        expect((await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
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
        expect((await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
      });
    });

    it('should verify random data (128 nodes)', async () => {
      const test = await makeRandomProofTests('seed128', 128);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root)).to.equal(true);
    });

    it('should verify random data (256 nodes)', async () => {
      const test = await makeRandomProofTests('seed256', 256);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root)).to.equal(true);
    });

    it('should verify random data (512 nodes)', async () => {
      const test = await makeRandomProofTests('seed512', 512);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root)).to.equal(true);
    });

    it('should verify random data (1024 nodes)', async () => {
      const test = await makeRandomProofTests('seed1024', 1024);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root)).to.equal(true);
    });

    it('should verify random data (2048 nodes)', async () => {
      const test = await makeRandomProofTests('seed2048', 2048);
      expect(await trie.verifyInclusionProof(test.key, test.val, test.proof, test.root)).to.equal(true);
    });
  });

  describe('verifyExclusionProof', () => {
    it('should verify exclusion with an existing key and differing value', async () => {
      const test = await makeProofTest([
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
      ], 'key1aa', null, false);

      expect((await trie.verifyExclusionProof(test.key, test.val, test.proof, test.root))).to.equal(true);
    });

    it('should verify exclusion with a non-existent extension of a leaf', async () => {
      const test = await makeProofTest([
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
      ], 'key1aab', null, false);

      expect((await trie.verifyExclusionProof(Buffer.from('key1aab'), test.val, test.proof, test.root))).to.equal(true);
    });

    it('should verify exclusion with a non-existent extension of a branch', async () => {
      const test = await makeProofTest([
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
      ], 'key4dd', null, false);

      expect((await trie.verifyExclusionProof(Buffer.from('key4dd'), test.val, test.proof, test.root))).to.equal(true);
    });
  });
});