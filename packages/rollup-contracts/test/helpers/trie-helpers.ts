import * as rlp from 'rlp';
import * as seedbytes from 'random-bytes-seed';
import * as seedfloat from 'seedrandom';
import { BaseTrie } from 'merkle-patricia-tree';

interface ProofTest {
  proof: string;
  root: string;
  key: string;
  val: string;
}

interface TrieNode {
  key: string;
  val: string;
}

const randomInt = (seed: string, min: number, max: number): number => {
  const randomFloat = seedfloat(seed);
  min = Math.ceil(min);
  max = Math.floor(max);
  return Math.floor(randomFloat() * (max - min + 1)) + min;
};

const makeTrie = async (nodes: TrieNode[]): Promise<BaseTrie> => {
  const trie = new BaseTrie();

  for (const node of nodes) {
    await trie.put(Buffer.from(node.key), Buffer.from(node.val));
  }

  return trie;
};

const makeProofTest = async (nodes: TrieNode[], key: string, trie?: BaseTrie): Promise<ProofTest> => {
  trie = trie || await makeTrie(nodes);

  const proof = await BaseTrie.prove(trie, Buffer.from(key));
  const val = await BaseTrie.verifyProof(trie.root, Buffer.from(key), proof);

  return {
    proof: '0x' + rlp.encode(proof).toString('hex'),
    root: '0x' + trie.root.toString('hex'),
    key: '0x' + Buffer.from(key).toString('hex'),
    val: '0x' + val.toString('hex'),
  };
};

export const makeAllProofTests = async (nodes: TrieNode[]): Promise<ProofTest[]> => {
  const trie = await makeTrie(nodes);
  const tests: ProofTest[] = [];

  for (const node of nodes) {
    tests.push(await makeProofTest(nodes, node.key, trie));
  }

  return tests;
}

export const makeRandomProofTests = async (germ: string, count: number, keySize: number = 32, valSize: number = 32): Promise<ProofTest> => {
  const randomBytes = seedbytes(germ);
  const nodes: TrieNode[] = Array(count).fill({}).map(() => {
    return {
      key: randomBytes(keySize).toString('hex'),
      val: randomBytes(valSize).toString('hex'),
    };
  });
  const r = randomInt(germ, 0, count);
  return makeProofTest(nodes, nodes[randomInt(germ, 0, count)].key);
}