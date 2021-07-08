#!/usr/bin/env node

const DatabaseService = require('./database.service');

class ChainScannerService extends DatabaseService {
  constructor() {
    super(...arguments);
    this.running = false;
    this.latestBlock = null;
    this.scannedLastBlock = null;
  }

  async initChainScannerService() {
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

    await this.initBaseService();

    this.running = true;
  }

  async startChainScannerService() {
    // Create tables
    await this.initDatabaseService();
    await this.initMySQL();

    // scan the chain and get the latest number of blocks
    this.latestBlock = await this.L2Provider.getBlockNumber();

    // check the latest block on MySQL
    let latestSQLBlockQuery = await this.getNewestBlock();
    let latestSQLBlock = latestSQLBlockQuery[0]['MAX(blockNumber)'];

    // get the blocks, transactions and receipts
    this.logger.info('Fetching the block data...');
    const [blocksData, receiptsData] = await this.getChainData(latestSQLBlock, this.latestBlock);

    // write the block data into MySQL
    this.logger.info('Writing the block data...');
    for (let blockData of blocksData) {
      await this.insertBlockData(blockData);
      // write the transaction data into MySQL
      if (blockData.transactions.length) {
        for (let transactionData of blockData.transactions) {
          transactionData.timestamp = blockData.timestamp;
          await this.insertTransactionData(transactionData);
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
      await this.insertReceiptData(receiptData);
    }

    // update scannedLastBlock
    this.scannedLastBlock = this.latestBlock;
    this.con.end();

    while (this.running) {
      const latestBlock = await this.L2Provider.getBlockNumber();
      if (latestBlock > this.latestBlock) {
        this.logger.info('Finding new blocks...');
        this.latestBlock = latestBlock;

        // connect to MySQL
        await this.initDatabaseService();

        // get the blocks, transactions and receipts
        this.logger.info('Fetching the block data...');
        const [blocksData, receiptsData] = await this.getChainData(this.scannedLastBlock, this.latestBlock);

        // write the block data into MySQL
        this.logger.info('Writing the block data...');
        for (let blockData of blocksData) {
          await this.insertBlockData(blockData);
          // write the transaction data into MySQL
          if (blockData.transactions.length) {
            for (let transactionData of blockData.transactions) {
              transactionData.timestamp = blockData.timestamp;
              await this.insertTransactionData(transactionData);
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
          await this.insertReceiptData(receiptData);
        }

        // update scannedLastBlock
        this.scannedLastBlock = this.latestBlock;
        this.con.end();
      } else {
        // this.logger.info('No new block found.');
      }

      await this.sleep(this.chainScanInterval);
    }
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
}

module.exports = ChainScannerService;