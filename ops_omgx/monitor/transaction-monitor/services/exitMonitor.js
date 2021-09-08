#!/usr/bin/env node

const ethers = require('ethers');
const DatabaseService = require('./database.service');
const OptimismEnv = require('./utilities/optimismEnv');
const { sleep } = require('@eth-optimism/core-utils')

class exitMonitorService extends OptimismEnv {
  constructor() {
    super(...arguments);

    this.databaseService = new DatabaseService();

    this.latestL2Block = null;

    this.startBlock = Number(0);
    this.endBlock = Number(this.startBlock) + Number(this.exitMonitorLogInterval);
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
    await this.databaseService.initMySQL();

    // fetch the last end block
    const startBlock = (await this.databaseService.getNewestBlockFromExitTable())[0]['MAX(blockNumber)'];
    if (startBlock) {
      this.startBlock = Number(startBlock);
      this.endBlock = Number(this.startBlock) + Number(this.exitMonitorLogInterval);
    }
  }

  async startExitMonitor() {
    const latestL2Block = await this.L2Provider.getBlockNumber();

    const endBlock = Math.min(latestL2Block, this.endBlock);

    const [userRewardFeeRate, ownerRewardFeeRate] = await Promise.all([
      this.L1LiquidityPoolContract.userRewardFeeRate(),
      this.L1LiquidityPoolContract.ownerRewardFeeRate()
    ]);

    const totalFeeRate = userRewardFeeRate.add(ownerRewardFeeRate)

    const L2LPLog = await this.L2LiquidityPoolContract.queryFilter(
      this.L2LiquidityPoolContract.filters.ClientDepositL2(),
      Number(this.startBlock),
      Number(endBlock)
    )

    const OVM_L2StandardBridgeLog = await this.OVM_L2StandardBridgeContract.queryFilter(
      this.OVM_L2StandardBridgeContract.filters.WithdrawalInitiated(),
      Number(this.startBlock),
      Number(endBlock)
    )

    if (L2LPLog.length) {
      for (const eachL2LPLog of L2LPLog) {
        const hash = eachL2LPLog.transactionHash;
        const blockHash = eachL2LPLog.blockHash;
        const blockNumber = Number(eachL2LPLog.blockNumber);
        const exitSender = eachL2LPLog.args.sender;
        const exitTo = eachL2LPLog.args.sender;
        const exitToken = eachL2LPLog.args.tokenAddress;
        const exitAmount = eachL2LPLog.args.receivedAmount.toString();
        const exitReceive = eachL2LPLog.args.receivedAmount.sub(
          eachL2LPLog.args.receivedAmount.mul(totalFeeRate).div(ethers.BigNumber.from('1000'))
        ).toString()
        const exitFeeRate = totalFeeRate.toString();
        const fastRelay = true;
        const payload = { hash, blockHash, blockNumber, exitSender, exitTo, exitToken,
          exitAmount, exitReceive, exitFeeRate, fastRelay };
        this.logger.info(`Found LP fast exit found from block ${this.startBlock} to ${endBlock}`, payload);
        await this.databaseService.insertExitData(payload);
      }
    } else {
      this.logger.info(`No LP fast exit found from block ${this.startBlock} to ${endBlock}`);
    }


    if (OVM_L2StandardBridgeLog.length) {
      for (const eachOVM_L2StandardBridgeLog of OVM_L2StandardBridgeLog) {
        const hash = eachOVM_L2StandardBridgeLog.transactionHash;
        const blockHash = eachOVM_L2StandardBridgeLog.blockHash;
        const blockNumber = Number(eachOVM_L2StandardBridgeLog.blockNumber);
        const exitSender = eachOVM_L2StandardBridgeLog.args._from;
        const exitTo = eachOVM_L2StandardBridgeLog.args._to;
        const exitToken = eachOVM_L2StandardBridgeLog.args._l2Token;
        const exitAmount = eachOVM_L2StandardBridgeLog.args._amount.toString();
        const exitReceive = eachOVM_L2StandardBridgeLog.args._amount.toString();
        const exitFeeRate = '0';
        const fastRelay = false;
        const payload = { hash, blockHash, blockNumber, exitSender, exitTo, exitToken,
          exitAmount, exitReceive, exitFeeRate, fastRelay };
        this.logger.info(`Found standard bridge exit found from block ${this.startBlock} to ${endBlock}`, payload);
        await this.databaseService.insertExitData(payload);
      }
    } else {
      this.logger.info(`No standard bridge exit found from block ${this.startBlock} to ${endBlock}`);
    }

    this.startBlock = endBlock;
    this.endBlock = Number(endBlock) + Number(this.exitMonitorLogInterval);
    this.latestL2Block = latestL2Block;

    await sleep(this.exitMonitorInterval);
  }
}

module.exports = exitMonitorService;
