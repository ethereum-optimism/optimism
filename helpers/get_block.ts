import { ethers } from 'ethers'
import assert from 'assert'
import fs from 'fs'

const keccak256 = ethers.utils.keccak256
const rlp = require('rlp')

// local geth
const provider = new ethers.providers.JsonRpcProvider('http://192.168.1.213:8545')

/*
type extblock struct {
	Header *Header
	Txs    []*Transaction
	Uncles []*Header
}
*/

function getBlockRlp(block : any) {
  let dat = [
    block['parentHash'],
    block['sha3Uncles'],
    block['miner'],
    block['stateRoot'],
    block['transactionsRoot'],
    block['receiptsRoot'],
    block['logsBloom'],
    block['difficulty'],
    block['number'],
    block['gasLimit'],
    block['gasUsed'],
    block['timestamp'],
    block['extraData'],
    block['mixHash'],
    block['nonce'],
    block['baseFeePerGas']
  ];
  // special case for 0
  dat = dat.map((x) => (x == "0x0") ? "0x" : x)
  return rlp.encode(dat);
}

async function main() {
  const blockNumber = 13247502;
  let blockData = await provider.send("eth_getBlockByNumber", ["0x"+(blockNumber).toString(16), true])
  const blockHeaderRlp = getBlockRlp(blockData)
  assert(keccak256(blockHeaderRlp) == blockData['hash'])

  fs.writeFileSync(`data/block_${blockNumber}`, blockHeaderRlp)
}

main().then(() => process.exit(0))
