#!/usr/bin/env node

const { Watcher } = require("../build/src")
const {
	providers: { JsonRpcProvider },
} = require('ethers');

const l2Provider = new JsonRpcProvider('')
const l1Provider = new JsonRpcProvider('')
let watcher
const initWatcher = () => {
	watcher = new Watcher({
		l1: {
			provider: l1Provider,
			messengerAddress: '0x'
		},
		l2: {
			provider: l2Provider,
			messengerAddress: '0x'
		}
	})
}

;(async ()=> {
	initWatcher()
	const msgHashes = await watcher.getMessageHashesFromL2Tx('')
	console.log('got messages', msgHashes)
	const receipt = await watcher.getL1TransactionReceipt(msgHashes[0])
	console.log('got receipt:', receipt)
})()