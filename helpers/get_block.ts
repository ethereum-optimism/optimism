import { ethers } from 'ethers'
import assert from 'assert'
import fs from 'fs'
import { exit } from 'process'

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
function getTransactionRlp(tx : any): Buffer {
  let dat: any
  // TODO: there are also type 1 transactions
  if (tx.type == "0x2") {
    let accesslist = tx.accessList.map((x : any) => [x.address, x.storageKeys])
    // london
    dat = [
      tx.chainId,
      tx.nonce,
      tx.maxPriorityFeePerGas,
      tx.maxFeePerGas,
      tx.gas,
      tx.to,
      tx.value,
      tx.input,
      accesslist,
      tx.v,
      tx.r,
      tx.s,
    ]
    dat = dat.map((x : any) => (x == "0x0") ? "0x" : x)
    dat = Buffer.concat([Buffer.from([parseInt(tx.type)]), rlp.encode(dat)])
    assert(keccak256(dat) == tx.hash)
  } else {
    // pre london
    dat = [
      tx.nonce,
      tx.gasPrice,
      tx.gas,
      tx.to,
      tx.value,
      tx.input,
      tx.v,
      tx.r,
      tx.s,
    ];
    dat = dat.map((x : any) => (x == "0x0") ? "0x" : x)
    dat = rlp.encode(dat)
    assert(keccak256(dat) == tx.hash)
    dat = rlp.decode(dat)
  }
  //console.log(tx)
  return dat
}

function getBlockRlp(block : any): Buffer {
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
  dat = dat.map((x : any) => (x == "0x0") ? "0x" : x)
  //console.log(dat)
  let rdat = rlp.encode(dat)
  assert(keccak256(rdat) == block['hash'])
  return rdat
}

async function getBlock(blockNumber: Number) {
  let blockData = await provider.send("eth_getBlockByNumber", ["0x"+(blockNumber).toString(16), true])
  const blockHeaderRlp = getBlockRlp(blockData)
  //console.log(blockData)
  const txsRlp = blockData.transactions.map(getTransactionRlp)

  fs.writeFileSync(`data/block_${blockNumber}`, blockHeaderRlp)
  fs.writeFileSync(`data/txs_${blockNumber}`, rlp.encode(txsRlp))
}


async function main() {
  await getBlock(13247501)
  await getBlock(13247502)
}

main().then(() => process.exit(0))
