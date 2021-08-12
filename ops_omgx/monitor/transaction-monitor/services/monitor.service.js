#!/usr/bin/env node

const ethers = require('ethers');
const DatabaseService = require('./database.service');
const OptimismEnv = require('./utilities/optimismEnv');
const fetch = require('node-fetch');

class MonitorService extends OptimismEnv {
  constructor() {
    super(...arguments);

    this.databaseService = new DatabaseService();

    this.latestBlock = null;
    this.scannedLastBlock = null;
    this.lastCheckWhitelist = (new Date().getTime() / 1000).toFixed(0);
    this.lastCheckNonWhitelist = (new Date().getTime() / 1000).toFixed(0);

    this.whitelist = [];

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

    // get whitelist
    await this.getWhitelist();

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

      receiptData = await this.getCrossDomainMessageStatusL2(receiptData, blocksData);
      // if message is cross domain check if message has been finalized
      if (receiptData.crossDomainMessage){
        receiptData = await this.getCrossDomainMessageStatusL1(receiptData);
      }
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
      this.transactionMonitorSQL = true;
      await this.startDatabaseService();

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
        // check if message is cross domain
        receiptData = await this.getCrossDomainMessageStatusL2(receiptData, blocksData);

        // if message is cross domain check if message has been finalized
        if (receiptData.crossDomainMessage){
          receiptData = await this.getCrossDomainMessageStatusL1(receiptData);
        }

        await this.databaseService.insertReceiptData(receiptData);
      }

      // update scannedLastBlock
      this.scannedLastBlock = this.latestBlock;

      this.transactionMonitorSQL = false;
      await this.endDatabaseService();

      this.logger.info(`Found block, receipt and transaction data. Sleeping ${this.transactionMonitorInterval} ms...`);
    } else {
      // this.logger.info('No new block found.');
    }

    await this.sleep(this.transactionMonitorInterval);
  }

  async startCrossDomainMessageMonitor() {
    // connect to MySQL
    this.crossDomainMessageMonitorSQL = true;
    await this.startDatabaseService();


    this.logger.info('Searching cross domain messages...');
    const crossDomainData = await this.databaseService.getCrossDomainData();

    // counts the number of server request
    let promiseCount = 0;

    let checkWhitelist = this.checkTime(this.whitelistString);
    if(checkWhitelist) await this.getWhitelist();
    let checkNonWhitelist = this.checkTime(this.nonWhitelistString);

    if (crossDomainData.length) {
      this.logger.info('Found cross domain message.');

      for (let receiptData of crossDomainData) {
        if(promiseCount % this.L2sleepThresh === 0){
          await this.sleep(2000);
        }
        // if its time check cross domain message finalization
        if(receiptData.fastRelay){
          receiptData = await this.getCrossDomainMessageStatusL1(receiptData);
        }else if (checkNonWhitelist && !receiptData.fastRelay){
          receiptData = await this.getCrossDomainMessageStatusL1(receiptData);
        }
        promiseCount = promiseCount + 1;

        if (receiptData.crossDomainMessageFinalize) {
          await this.databaseService.updateCrossDomainData(receiptData);
        }
      }
      promiseCount = 0;
    } else {
      this.logger.info('No waiting cross domain message found.');
    }

    if(checkWhitelist) checkWhitelist = false;
    if(checkNonWhitelist) checkNonWhitelist = false;

    this.crossDomainMessageMonitorSQL = false;
    await this.endDatabaseService();

    this.logger.info(`End searching cross domain messages. Sleeping ${this.crossDomainMessageMonitorInterval} ms...`);

    await this.sleep(this.crossDomainMessageMonitorInterval);
  }

  async getChainData(startingBlock, endingBlock) {
    const promisesBlock = [], promisesReceipt = [];
    for (let i = startingBlock; i <= endingBlock; i++) {
      promisesBlock.push(this.L2Provider.getBlockWithTransactions(i));
      this.logger.info(`Pushing block`);
    }
    const blocksData = await Promise.all(promisesBlock);
    for (let blockData of blocksData) {
      if (blockData.transactions.length) {
        blockData.transactions.forEach(i => {
          promisesReceipt.push(this.L2Provider.getTransactionReceipt(i.hash));
        });
      }
      this.sleep(2000);
    }
    const receiptsData = await Promise.all(promisesReceipt);

    return [blocksData, receiptsData]
  }


  async getCrossDomainMessageStatusL2(receiptData, blocksData){
    this.logger.info(`Searching ${receiptData.transactionHash}...`);

    const filteredBlockData = blocksData.filter(i => i.hash === receiptData.blockHash);

    let crossDomainMessageSendTime;
    let crossDomainMessageEstimateFinalizedTime;
    let crossDomainMessage = false;
    let crossDomainMessageFinalize = false;
    let fastRelay = false;

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
        let decoded = this.OVM_L2CrossDomainMessengerContract.interface.decodeFunctionData(
          'relayMessage',
          message
        );
        if(this.whitelist.includes(decoded._target)) fastRelay = true;
      }
    }

    if (filteredBlockData.length) {
      crossDomainMessageSendTime = filteredBlockData[0].timestamp ;
      crossDomainMessageEstimateFinalizedTime = fastRelay ?
        crossDomainMessageSendTime + 60 : crossDomainMessageSendTime + 60 * 60 * 24 * 6;
    }

    receiptData.crossDomainMessageSendTime = crossDomainMessageSendTime;
    receiptData.crossDomainMessageEstimateFinalizedTime = crossDomainMessageEstimateFinalizedTime;
    receiptData.crossDomainMessage = crossDomainMessage;
    receiptData.crossDomainMessageFinalize = crossDomainMessageFinalize;
    receiptData.fastRelay = fastRelay;

    return receiptData;
  }


  async getCrossDomainMessageStatusL1(receiptData){
    this.logger.info("Checking if message has been finalized...");
    const receiptDataRaw = await this.L2Provider.getTransactionReceipt(receiptData.transactionHash ? receiptData.transactionHash : receiptData.hash);
    receiptData = { ...JSON.parse(JSON.stringify(receiptData)), ...receiptDataRaw };
    // Find the transaction that sends message from L2 to L1
    const filteredLogData = receiptData.logs.filter(i => i.address === this.OVM_L2CrossDomainMessenger && i.topics[0] === ethers.utils.id('SentMessage(bytes)'));
    let msgHash;
    if (filteredLogData.length) {
      const [message] = ethers.utils.defaultAbiCoder.decode(
        ['bytes'],
        filteredLogData[0].data
      );
      msgHash = ethers.utils.solidityKeccak256(['bytes'], [message])
    } else return receiptData;

    let crossDomainMessageFinalize = false;
    let crossDomainMessageFinalizedTime;

    const matches = await this.getL1TransactionReceipt(msgHash, receiptData.fastRelay)
    if (matches !== false) {
      const l1Receipt = await this.L1Provider.getTransactionReceipt(matches[0].transactionHash)
      crossDomainMessageFinalize = true;
      crossDomainMessageFinalizedTime = (new Date().getTime() / 1000).toFixed(0);
      receiptData.l1Hash = l1Receipt.transactionHash.toString();
      receiptData.l1BlockNumber = Number(l1Receipt.blockNumber.toString());
      receiptData.l1BlockHash = l1Receipt.blockHash.toString();
      receiptData.l1From = l1Receipt.from.toString();
      receiptData.l1To = l1Receipt.to.toString();
    }

    receiptData.crossDomainMessageFinalize = crossDomainMessageFinalize;
    receiptData.crossDomainMessageFinalizedTime = crossDomainMessageFinalizedTime;

    this.logger.info("Found the cross domain message status", {
      crossDomainMessageFinalize,
      crossDomainMessageFinalizedTime
    });

    return receiptData;
  }

  async getL1TransactionReceipt(msgHash, fast=false) {
    const blockNumber = await this.L1Provider.getBlockNumber();
    const startingBlock = Math.max(blockNumber - this.numberBlockToFetch, 0);

    const filter = {
      address: (fast ? this.OVM_L1CrossDomainMessengerFast : this.OVM_L1CrossDomainMessenger),
      topics: [ethers.utils.id(`RelayedMessage(bytes32)`)],
      fromBlock: startingBlock,
    }

    const logs = await this.L1Provider.getLogs(filter);
    const matches = logs.filter(i => i.data === msgHash);

    if (matches.length > 0) {
      if (matches.length > 1) return false;
      return matches
    } else {
      return false;
    }
  }

  // gets list of addresses whose messages may finalize fast
  async getWhitelist(){
    let response = await fetch(this.filterEndpoint);
    const filter = await response.json();
    const filterSelect = [ filter.Proxy__L1LiquidityPool, filter.L1Message ]
    this.whitelist = filterSelect
    this.logger.info('Found the filter', { filterSelect })
  }

  // checks to see if its time to look for L1 finalization
  checkTime(list){
    let currentTime = (new Date().getTime() / 1000).toFixed(0);
    if(list === this.whitelistString){
      if((currentTime - this.lastCheckWhitelist) >= this.whitelistSleep){
        this.lastCheckWhitelist = currentTime;
        return true;
      }
    } else if (list === this.nonWhitelistString){
      if((currentTime - this.lastCheckNonWhitelist) >= this.nonWhitelistSleep){
        this.lastCheckNonWhitelist = currentTime;
        return true;
      }
    }
    return false;
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
          if(this.databaseConnected && !this.transactionMonitorSQL && !this.crossDomainMessageMonitorSQL){
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

  errorCatcher(func, param){
    return (async () =>{
      for(let i=0; i < 2; i++){
        try {
            let result = await func(param);
            return result;
        } catch (error) {
          console.log(`${func}returned an error!`, error);
          await this.sleep(1000);
        }
      }
    })();
  }
}



module.exports = MonitorService;