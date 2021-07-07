#!/usr/bin/env node

const ethers = require('ethers');
const util = require('util');
const DatabaseService = require('./database.service');

class L2ToL1MessageScanner extends DatabaseService {
  constructor() {
    super(...arguments);
    this.running = false;
    this.latestBlock = null;
    this.scannedLastBlock = null;
  }

  async initL2ToL1MessageScanner() {
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

    await this.initDatabaseService();
    await this.initMySQL();
    this.con.end();

    this.running = true;
  }

  async startL2ToL1MessageScanner() {
    while (this.running) {
      await this.initDatabaseService();
      this.logger.info('Searching cross domain messages...');
      const crossDomainData = await this.getCrossDomainData();

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

      await this.sleep(this.messageScanInterval);
    }
  }
}

module.exports = L2ToL1MessageScanner;