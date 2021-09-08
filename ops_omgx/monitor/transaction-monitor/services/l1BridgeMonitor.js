#!/usr/bin/env node

const ethers = require('ethers');
const DatabaseService = require('./database.service');
const OptimismEnv = require('./utilities/optimismEnv');
const { sleep } = require('@eth-optimism/core-utils')

class l1BridgeMonitorService extends OptimismEnv {
  constructor() {
    super(...arguments);

    this.databaseService = new DatabaseService();

    this.latestL1Block = null;

    this.startBlock = Number(this.l1BridgeMonitorStartBlock);
    this.endBlock = Number(this.startBlock) + Number(this.l1BridgeMonitorLogInterval);
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
    const startBlock = (await this.databaseService.getNewestBlockFromL1BridgeTable())[0]['MAX(blockNumber)'];
    if (startBlock) {
      this.startBlock = Number(startBlock);
      this.endBlock = Number(this.startBlock) + Number(this.l1BridgeMonitorLogInterval);
    }
  }

  async startL1BridgeMonitor() {
    const latestL1Block = await this.L1Provider.getBlockNumber();
    const endBlock = Math.min(latestL1Block, this.endBlock);

    const [userRewardFeeRate, ownerRewardFeeRate] = await Promise.all([
      this.L2LiquidityPoolContract.userRewardFeeRate(),
      this.L2LiquidityPoolContract.ownerRewardFeeRate()
    ]);

    const totalFeeRate = userRewardFeeRate.add(ownerRewardFeeRate)

    const L1LPLog = await this.L1Provider.getLogs({
      address: this.L1LiquidityPoolAddress,
      fromBlock: Number(this.startBlock),
      toBlock: Number(endBlock)
    })

    if (L1LPLog.length) {
      for (const eachL1LPLog of L1LPLog) {
        const L1LPEvent = this.L1LiquidityPoolInterface.parseLog(eachL1LPLog);
        if (L1LPEvent.name !== "ClientPayL1Settlement" && L1LPEvent.name !== "ClientPayL1") {
          const hash = eachL1LPLog.transactionHash;
          const blockHash = eachL1LPLog.blockHash;
          const blockNumber = eachL1LPLog.blockNumber;
          const timestamp = Number((await this.L1Provider.getBlock(blockNumber)).timestamp)
          const receipt = await this.L1Provider.getTransactionReceipt(hash);
          const from = receipt.from;
          const to = receipt.to;
          const contractAddress = to;
          const contractName = "L1LiquidityPool";
          const activity = L1LPEvent.name;
          let crossDomainMessage = false;
          let crossDomainMessageFinalize = null;
          let crossDomainMessageSendTime = null;
          let crossDomainMessageEstimateFinalizedTime = null;
          let fastDeposit = null;

          // deposit L2
          if (L1LPEvent.name === "ClientDepositL1") {
            fastDeposit = true;
            crossDomainMessage = true;
            crossDomainMessageFinalize = false;
            crossDomainMessageSendTime = timestamp;
            crossDomainMessageEstimateFinalizedTime = timestamp + Number(this.l1CrossDomainMessageWaitingTime);
            const depositSender = L1LPEvent.args.sender;
            const depositTo = L1LPEvent.args.sender;
            const depositToken = L1LPEvent.args.tokenAddress;
            const depositAmount = L1LPEvent.args.receivedAmount.toString();
            const depositReceive = L1LPEvent.args.receivedAmount.sub(
              L1LPEvent.args.receivedAmount.mul(totalFeeRate).div(ethers.BigNumber.from('1000'))
            ).toString()
            const depositFeeRate = totalFeeRate.toString();
            await this.databaseService.insertDepositL2({
              hash, blockHash, blockNumber, depositSender, depositTo, depositToken,
              depositAmount, depositReceive, depositFeeRate, fastDeposit
            })
          }

          const payload = {
            hash, blockHash, blockNumber, from, to, contractAddress, contractName, activity,
            crossDomainMessage, crossDomainMessageFinalize, crossDomainMessageSendTime,
            crossDomainMessageEstimateFinalizedTime, crossDomainMessageFinalizedTime: null,
            timestamp, fastDeposit,
          }

          await this.databaseService.insertL1BridgeData(payload);
          this.logger.info(`Found L1 LP logs found from block ${this.startBlock} to ${endBlock}`);
        }
      }
    } else {
      this.logger.info(`No L1 LP logs found from block ${this.startBlock} to ${endBlock}`);
    }

    const L1StandardBridgeLog = await this.L1Provider.getLogs({
      address: this.OVM_L1StandardBridgeContract.address,
      fromBlock: Number(this.startBlock),
      toBlock: Number(endBlock)
    })

    if (L1StandardBridgeLog.length) {
      for (const eachL1StandardBridgeLog of L1StandardBridgeLog) {
        const L1StandardBridgeEvent = this.OVM_L1StandardBridgeInterface.parseLog(eachL1StandardBridgeLog);
        if (L1StandardBridgeEvent.name === "ERC20DepositInitiated" || L1StandardBridgeEvent.name === "ETHDepositInitiated") {
          const hash = eachL1StandardBridgeLog.transactionHash;
          const blockHash = eachL1StandardBridgeLog.blockHash;
          const blockNumber = eachL1StandardBridgeLog.blockNumber;
          const timestamp = Number((await this.L1Provider.getBlock(blockNumber)).timestamp)
          const receipt = await this.L1Provider.getTransactionReceipt(hash);
          const from = receipt.from;
          const to = receipt.to;
          const contractAddress = to;
          const contractName = "L1StandardBridge";
          const activity = L1StandardBridgeEvent.name;
          const crossDomainMessage = true;
          const crossDomainMessageFinalize = false;
          const crossDomainMessageSendTime = timestamp;
          const crossDomainMessageEstimateFinalizedTime = timestamp + Number(this.l1CrossDomainMessageWaitingTime);
          const fastDeposit = false;
          const depositSender = L1StandardBridgeEvent.args._from;
          const depositTo = L1StandardBridgeEvent.args._to;
          let depositToken = null;
          if (L1StandardBridgeEvent.name === "ETHDepositInitiated") {
            depositToken = "0x0000000000000000000000000000000000000000";
          } else {
            depositToken = L1StandardBridgeEvent.args._l1Token;
          }
          const depositAmount = L1StandardBridgeEvent.args._amount.toString();
          const depositReceive = L1StandardBridgeEvent.args._amount.toString()
          const depositFeeRate = "0";

          await this.databaseService.insertDepositL2({
            hash, blockHash, blockNumber, depositSender, depositTo, depositToken,
            depositAmount, depositReceive, depositFeeRate, fastDeposit
          })

          const payload = {
            hash, blockHash, blockNumber, from, to, contractAddress, contractName, activity,
            crossDomainMessage, crossDomainMessageFinalize, crossDomainMessageSendTime,
            crossDomainMessageEstimateFinalizedTime, crossDomainMessageFinalizedTime: null,
            timestamp, fastDeposit,
          }

          await this.databaseService.insertL1BridgeData(payload);
          this.logger.info(`Found standard bridge logs found from block ${this.startBlock} to ${endBlock}`);
        }
      }
    } else {
      this.logger.info(`No L1 standard bridge logs found from block ${this.startBlock} to ${endBlock}`);
    }

    this.startBlock = endBlock;
    this.endBlock = Number(endBlock) + Number(this.l1BridgeMonitorLogInterval);
    this.latestL1Block = latestL1Block;

    await sleep(this.l1BridgeMonitorInterval);
  }

  async startCrossDomainMessageMonitor() {
    const crossDomainMessages = await this.databaseService.getL1CrossDomainData();
    for (const eachCrossDomainMessage of crossDomainMessages) {
      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(eachCrossDomainMessage.hash);
      const l2Message = await this.getL1ToL2TransactionReceipt(l1ToL2msgHash);
      if (l2Message !== false) {
        const l2Hash = l2Message.transactionHash;
        const l2BlockNumber = Number(l2Message.blockNumber);
        const l2BlockHash = l2Message.blockHash;
        const blockData = await this.L2Provider.getBlockWithTransactions(l2BlockNumber);
        const crossDomainMessageFinalizedTime = Number(blockData.timestamp);
        const l2From = blockData.transactions[0].from;
        const l2To = blockData.transactions[0].to;

        const payload = {
          hash: eachCrossDomainMessage.hash, blockNumber: eachCrossDomainMessage.blockNumber,
          l2Hash, l2BlockNumber, l2BlockHash, l2From, l2To, crossDomainMessageFinalizedTime,
          crossDomainMessageFinalize: true
        };

        await this.databaseService.updateL1BridgeData(payload);

        // Fast deposit
        if (eachCrossDomainMessage.fastDeposit) {
          const L2LiquidityPoolLog = await this.L2LiquidityPoolContract.queryFilter(
            this.L2LiquidityPoolContract.filters.ClientPayL2(),
            Number(l2BlockNumber),
            Number(l2BlockNumber)
          )

          const filteredL2LiquidityPoolLog = L2LiquidityPoolLog.filter(i => i.transactionHash === l2Hash)
          if (filteredL2LiquidityPoolLog.length) {
            this.logger.info("Found successful L2 deposit", { status: 'succeeded', hash: eachCrossDomainMessage.hash, blockNumber: eachCrossDomainMessage.blockNumber });
            await this.databaseService.updateDepositL2Data({
              status: 'succeeded',
              hash: eachCrossDomainMessage.hash,
              blockNumber: eachCrossDomainMessage.blockNumber
            })
          } else {
            this.logger.info("Found failure L2 deposit", { status: 'reverted', hash: eachCrossDomainMessage.hash, blockNumber: eachCrossDomainMessage.blockNumber });
            await this.databaseService.updateDepositL2Data({
              status: 'reverted',
              hash: eachCrossDomainMessage.hash,
              blockNumber: eachCrossDomainMessage.blockNumber
            })
          }
        } else {
          // Standard deposit
          this.logger.info("Found successful L2 deposit", { status: 'succeeded', hash: eachCrossDomainMessage.hash, blockNumber: eachCrossDomainMessage.blockNumber });
          await this.databaseService.updateDepositL2Data({
            status: 'succeeded',
            hash: eachCrossDomainMessage.hash,
            blockNumber: eachCrossDomainMessage.blockNumber
          })
        }
        this.logger.info(`Found L1 LP cross domain message ${eachCrossDomainMessage}`);
      } else {
        this.logger.info(`No L1 LP cross domain message found ${eachCrossDomainMessage}`);
      }
    }

    await sleep(this.crossDomainMessageMonitorInterval);
  }

  async getL1ToL2TransactionReceipt(l1ToL2msgHash) {

    const blockNumber = await this.L2Provider.getBlockNumber();
    const startBlock = Math.max(blockNumber - 50000, 0);

    const filter = {
      address: this.OVM_L2CrossDomainMessenger,
      topics: [ethers.utils.id(`RelayedMessage(bytes32)`)],
      fromBlock: startBlock,
    };

    const logs = await this.L2Provider.getLogs(filter);
    const matches = logs.filter(log => log.data === l1ToL2msgHash);

    if (matches.length > 0) {
      if (matches.length > 1) {
        this.logger.warn(`Found multiple transactions relaying the same message hash ${l1ToL2msgHash}.`);
        return false;
      }
      return matches[0];
    } else {
      return false;
    }
  }
}

module.exports = l1BridgeMonitorService;
