#!/usr/bin/env node

const ethers = require('ethers');
const DatabaseService = require('./database.service');
const OptimismEnv = require('./utilities/optimismEnv');
const { sleep } = require('@eth-optimism/core-utils')

class stateRootMonitorService extends OptimismEnv {
  constructor() {
    super(...arguments);

    this.databaseService = new DatabaseService();

    this.latestL1Block = null;
    this.latestL2Block = null;

    this.startBlock = Number(this.stateRootMonitorStartBlock);
    this.endBlock = Number(this.stateRootMonitorStartBlock) + Number(this.stateRootMonitorLogInterval);
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
              await sleep(1000);
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
              await sleep(1000);
          }
          else {
              throw new Error(`Unable to connect to the L2 network, check that your L2 endpoint is correct.`);
          }
      }
    }

    await this.initOptimismEnv();
    await this.databaseService.initDatabaseService();
    await this.databaseService.initMySQL();

    // fetch the last end block
    const startBlock = (await this.databaseService.getNewestBlockFromStateRootTable())[0]['MAX(stateRootBlockNumber)'];
    if (startBlock) {
      this.startBlock = Number(startBlock);
      this.endBlock = Number(this.startBlock) + Number(this.stateRootMonitorLogInterval);
    }
  }

  async startStateRootMonitor() {
    // Create tables
    await this.startDatabaseService();

    const latestL1Block = await this.L1Provider.getBlockNumber();
    const latestL2Block = await this.L2Provider.getBlockNumber();

    const endBlock = Math.min(latestL1Block, this.endBlock);

    const SCCLog = await this.OVM_StateCommitmentChainContract.queryFilter(
      this.OVM_StateCommitmentChainContract.filters.StateBatchAppended(),
      Number(this.startBlock),
      Number(endBlock)
    );

    if (SCCLog.length) {
      for (const eachSCCLog of SCCLog) {
        const stateRootBlockNumber = eachSCCLog.blockNumber;
        const stateRootHash = eachSCCLog.transactionHash;
        const stateRootBlockHash = eachSCCLog.blockHash;
        const stateRootBlockTimestamp = (await this.L1Provider.getBlock(stateRootBlockNumber)).timestamp;
        const receipt = await this.L1Provider.getTransactionReceipt(stateRootHash);
        const gasFee = (ethers.utils.formatEther(
          (ethers.BigNumber.from(receipt.cumulativeGasUsed.toString())
            .mul(ethers.BigNumber.from(receipt.effectiveGasPrice))).toString()
        )).toString()

        const batchSize = eachSCCLog.args._batchSize.toString();
        const batchIndex = eachSCCLog.args._batchIndex.toString();
        const prevTotalElements = eachSCCLog.args._prevTotalElements.toString();

        this.logger.info(`Found state root from block ${this.startBlock} to ${endBlock}`, {
          stateRootBlockNumber: Number(stateRootBlockNumber),
          stateRootHash,
          stateRootBlockHash,
          stateRootBlockTimestamp: Number(stateRootBlockTimestamp),
          batchSize: Number(batchSize),
          batchIndex: Number(batchIndex),
          prevTotalElements: Number(prevTotalElements),
          gasFee: Number(gasFee),
        })

        let stateRootStartBlock = Number(prevTotalElements) + 1;
        let stateRootendBlock = Number(prevTotalElements) + Number(batchSize);
        while (stateRootStartBlock <= stateRootendBlock) {
          const L2BlockData = await this.L2Provider.getBlockWithTransactions(stateRootStartBlock);
          const blockNumber = stateRootStartBlock;
          const hash = L2BlockData.transactions[0].hash;
          const blockHash = L2BlockData.hash;
          await this.databaseService.insertStateRootData({
            hash, blockHash, blockNumber, stateRootHash, stateRootBlockNumber,
            stateRootBlockHash, stateRootBlockTimestamp
          });
          stateRootStartBlock++
        }
      }
    } else {
      this.logger.info(`No state root found from block ${this.startBlock} to ${endBlock}`);
    }

    this.startBlock = endBlock;
    this.endBlock = Number(endBlock) + Number(this.stateRootMonitorLogInterval);
    this.latestL1Block = latestL1Block;
    this.latestL2Block = latestL2Block;

    await this.endDatabaseService();
    await sleep(this.stateRootMonitorInterval);
  }

    // starts up connection with mysql database safely
    async startDatabaseService(){
      await this.databaseConnectedMutex.acquire().then(async (release) => {
        try {
          if(!this.databaseConnected){
            await this.databaseService.initDatabaseService();
            this.databaseConnected = true;
          }
          release();
        } catch (error) {
          release();
          throw error;
        }
      });
    }

    // ends connection with mysql database safely
    async endDatabaseService(){
      await this.databaseConnectedMutex.acquire().then(async (release) => {
          try {
            if(this.databaseConnected){
                this.databaseService.con.end();
                this.databaseConnected = false;
            }
            release();
          } catch (error) {
            release();
            throw error;
          }
      });
    }
}

module.exports = stateRootMonitorService;