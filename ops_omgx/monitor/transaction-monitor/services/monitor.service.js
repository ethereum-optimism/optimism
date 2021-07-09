#!/usr/bin/env node

const ethers = require('ethers');

const DatabaseService = require('./database.service');
const OptimismEnv = require('./utilities/optimismEnv');

class MonitorService extends OptimismEnv {
  constructor() {
    super(...arguments);

    this.databaseService = new DatabaseService();

    this.latestBlock = null;
    this.scannedLastBlock = null;
  }

  async initConnection() {
    this.logger.info('Trying to connect to the L1 network...');
    for (let i = 0; i < 10; i++) {
      try {
          await this.L1Provider.detectNetwork();
          this.logger.info('Successfully connected to the L1 network.');
          break;
      }
      catch (err) {
          if (i < 9) {
              this.logger.info('Unable to connect to L1 network', {
                  retryAttemptsRemaining: 10 - i,
              });
              await this.sleep(1000);
          }
          else {
              throw new Error(`Unable to connect to the L1 network, check that your L1 endpoint is correct.`);
          }
      }
    }
    this.logger.info('Trying to connect to the L2 network...');
    for (let i = 0; i < 10; i++) {
      try {
          await this.L2Provider.detectNetwork();
          this.logger.info('Successfully connected to the L2 network.');
          break;
      }
      catch (err) {
          if (i < 9) {
              this.logger.info('Unable to connect to L2 network', {
                  retryAttemptsRemaining: 10 - i,
              });
              await this.sleep(1000);
          }
          else {
              throw new Error(`Unable to connect to the L2 network, check that your L2 endpoint is correct.`);
          }
      }
    }

    await this.initOptimismEnv();
    await this.databaseService.initDatabaseService();
  }

  async initScan() {
    // Create tables
    await this.databaseService.initDatabaseService();
    await this.databaseService.initMySQL();

    // scan the chain and get the latest number of blocks
    this.latestBlock = await this.L2Provider.getBlockNumber();

    // check the latest block on MySQL
    let latestSQLBlockQuery = await this.databaseService.getNewestBlock();
    let latestSQLBlock = latestSQLBlockQuery[0]['MAX(blockNumber)'];

    // get the blocks, transactions and receipts
    this.logger.info('Fetching the block data...');
    const [blocksData, receiptsData] = await this.getChainData(latestSQLBlock, this.latestBlock);

    // write the block data into MySQL
    this.logger.info('Writing the block data...');
    for (let blockData of blocksData) {
      await this.databaseService.insertBlockData(blockData);
      // write the transaction data into MySQL
      if (blockData.transactions.length) {
        for (let transactionData of blockData.transactions) {
          transactionData.timestamp = blockData.timestamp;
          await this.databaseService.insertTransactionData(transactionData);
        }
      }
    }

    // write the receipt data into MySQL
    this.logger.info('Writing the receipt data...');
    for (let receiptData of receiptsData) {
      const correspondingBlock = blocksData.filter(i => i.hash === receiptData.blockHash);
      if (correspondingBlock.length) {
        receiptData.timestamp = correspondingBlock[0].timestamp
      } else {
        receiptData.timestamp = (new Date().getTime() / 1000).toFixed(0);
      }
      receiptData = await this.getCrossDomainMessageStatus(receiptData, blocksData);
      await this.databaseService.insertReceiptData(receiptData);
    }

    // update scannedLastBlock
    this.scannedLastBlock = this.latestBlock;
    this.databaseService.con.end();

  }

  async startTransactionMonitor() {
    const latestBlock = await this.L2Provider.getBlockNumber();
    if (latestBlock > this.latestBlock) {
      this.logger.info('Finding new blocks...');
      this.latestBlock = latestBlock;

      // connect to MySQL
      await this.databaseService.initDatabaseService();

      // get the blocks, transactions and receipts
      this.logger.info('Fetching the block data...');
      const [blocksData, receiptsData] = await this.getChainData(this.scannedLastBlock, this.latestBlock);

      // write the block data into MySQL
      this.logger.info('Writing the block data...');
      for (let blockData of blocksData) {
        await this.databaseService.insertBlockData(blockData);
        // write the transaction data into MySQL
        if (blockData.transactions.length) {
          for (let transactionData of blockData.transactions) {
            transactionData.timestamp = blockData.timestamp;
            await this.databaseService.insertTransactionData(transactionData);
          }
        }
      }

      // write the receipt data into MySQL
      this.logger.info('Writing the receipt data...');
      for (let receiptData of receiptsData) {
        const correspondingBlock = blocksData.filter(i => i.hash === receiptData.blockHash);
        if (correspondingBlock.length) {
          receiptData.timestamp = correspondingBlock[0].timestamp
        } else {
          receiptData.timestamp = (new Date().getTime() / 1000).toFixed(0);
        }
        receiptData = await this.getCrossDomainMessageStatus(receiptData, blocksData);
        await this.databaseService.insertReceiptData(receiptData);
      }

      // update scannedLastBlock
      this.scannedLastBlock = this.latestBlock;
      this.databaseService.con.end();
    } else {
      // this.logger.info('No new block found.');
    }

    this.logger.info(`Found block, receipt and transaction data. Sleeping ${this.transactionMonitorInterval} ms...`);

    await this.sleep(this.transactionMonitorInterval);
  }

  async startCrossDomainMessageMonitor() {
    // connect to MySQL
    await this.databaseService.initDatabaseService();

    this.logger.info('Searching cross domain messages...');
    const crossDomainData = await this.databaseService.getCrossDomainData();

    if (crossDomainData.length) {
      this.logger.info('Found cross domain message.');
      const promisesBlock = [], promisesReceipt = [];
      for (let hashes of crossDomainData) {
        const hash = hashes.hash;
        const blockNumber = Number(hashes.blockNumber);
        promisesReceipt.push(this.L2Provider.getTransactionReceipt(hash));
        promisesBlock.push(this.L2Provider.getBlockWithTransactions(blockNumber));
      }
      const blocksData = await Promise.all(promisesBlock);
      const receiptsData = await Promise.all(promisesReceipt);
      for (let receiptData of receiptsData) {
        receiptData = await this.getCrossDomainMessageStatus(receiptData, blocksData);
        if (receiptData.crossDomainMessageFinalize) {
          await this.updateCrossDomainData(receiptData);
        }
      }
    } else {
      this.logger.info('No waiting cross domain message found.');
    }
    this.con.end();

    this.logger.info(`Found cross domain messages. Sleeping ${this.crossDomainMessageMonitorInterval} ms...`);

    await this.sleep(this.crossDomainMessageMonitorInterval);
  }

  async getChainData(startingBlock, endingBlock) {
    const promisesBlock = [], promisesReceipt = [];
    for (let i = startingBlock; i <= endingBlock; i++) {
      promisesBlock.push(this.L2Provider.getBlockWithTransactions(i));
    }
    const blocksData = await Promise.all(promisesBlock);
    for (let blockData of blocksData) {
      if (blockData.transactions.length) {
        blockData.transactions.forEach(i => {
          promisesReceipt.push(this.L2Provider.getTransactionReceipt(i.hash));
        });
      }
    }
    const receiptsData = await Promise.all(promisesReceipt);

    return [blocksData, receiptsData]
  }

  async getL1TransactionReceipt(msgHash) {
    const blockNumber = await this.L1Provider.getBlockNumber();
    const startingBlock = Math.max(blockNumber - this.numberBlockToFetch, 0);

    const filter = {
      address: this.OVM_L1CrossDomainMessenger,
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
    const filteredLogData = receiptData.logs.filter(i => i.address === this.OVM_L2CrossDomainMessenger && i.topics[0] === ethers.utils.id('SentMessage(bytes)'));
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

module.exports = MonitorService;