#!/usr/bin/env node

const ethers = require('ethers');
const util = require('util');

class Wather {
  async getL1TransactionReceipt(msgHash) {
    const blockNumber = await this.L1Provider.getBlockNumber();
    const startingBlock = Math.max(blockNumber - this.numberBlockToFetch, 0);

    const filter = {
      address: this.L1MessengerAddress,
      topics: [ethers.utils.id(`RelayedMessage(bytes32)`)],
      fromBlock: startingBlock,
    }

    const logs = await this.L1Provider.getLogs(filter);
    const matches = logs.filter(i => i.data === msgHash);
    
    this.logger.info(`Found matches: ${JSON.stringify(matches)}`);

    if (matches.length > 0) {
      if (matches.length > 1) return false;
      return true
    } else {
      return false;
    }
  }

  async getCrossDomainMessageStatus(receiptData, blocksData) {
    
    this.logger.info(`Searching ${receiptData.transactionHash}...`);

    const filteredBlockData = blocksData.filter(i => i.hash === receiptData.blockHash);
      
    let crossDomainMessageSendTime;
    let crossDomainMessageEstimateFinalizedTime; 
    let crossDomainMessage = false;
    let crossDomainMessageFinalize = false;
    let msgHashes = [];

    if (filteredBlockData.length) {
      crossDomainMessageSendTime = filteredBlockData[0].timestamp ;
      crossDomainMessageEstimateFinalizedTime = crossDomainMessageSendTime + 60 * 60;
    }
    // Find the transaction that sends message from L2 to L1
    const filteredLogData = receiptData.logs.filter(i => i.address === this.L2MessengerAddress && i.topics[0] === ethers.utils.id('SentMessage(bytes)'));
    if (filteredLogData.length) {
      crossDomainMessage = true;
      // Get message hashes from L2 TX
      for (let logData of filteredLogData) {
        const [message] = ethers.utils.defaultAbiCoder.decode(
          ['bytes'],
          logData.data
        );
        msgHashes.push(ethers.utils.solidityKeccak256(['bytes'], [message]));
      }

      // Check if L1 get the transaction receipt
      const L1Receipt = await this.getL1TransactionReceipt(msgHashes[0]);
      if (L1Receipt) crossDomainMessageFinalize = true;
      else crossDomainMessageFinalize = false;
    }

    receiptData.crossDomainMessage = crossDomainMessage;
    receiptData.crossDomainMessageFinalize = crossDomainMessageFinalize;
    receiptData.crossDomainMessageSendTime = crossDomainMessageSendTime;
    receiptData.crossDomainMessageEstimateFinalizedTime = crossDomainMessageEstimateFinalizedTime;

    return receiptData;
  }
}

module.exports = Wather;