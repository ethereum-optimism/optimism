#!/usr/bin/env node

const ethers = require('ethers');
const util = require('util');
const core_utils_1 = require('@eth-optimism/core-utils');
const Mutex = require('async-mutex').Mutex;
const { loadContract, loadContractFromManager } = require('@eth-optimism/contracts')

const addressManagerJSON = require('../../artifacts/contracts/optimistic-ethereum/libraries/resolver/Lib_AddressManager.sol/Lib_AddressManager.json');

require('dotenv').config();
const env = process.env;
const L1_NODE_WEB3_URL = env.L1_NODE_WEB3_URL || "http://localhost:8545";
const L2_NODE_WEB3_URL = env.L2_NODE_WEB3_URL || "http://localhost:9545";

const MYSQL_HOST_URL = env.MYSQL_HOST_URL || "127.0.0.1";
const MYSQL_PORT = env.MYSQL_PORT || 3306;
const MYSQL_USERNAME = env.MYSQL_USERNAME;
const MYSQL_PASSWORD = env.MYSQL_PASSWORD;
const MYSQL_DATABASE_NAME = env.MYSQL_DATABASE_NAME || "OMGXV1";

const ADDRESS_MANAGER_ADDRESS = env.ADDRESS_MANAGER_ADDRESS;
const L2_MESSENGER_ADDRESS = env.L2_MESSENGER_ADDRESS || "0x4200000000000000000000000000000000000007";

const DEPLOYER_PRIVATE_KEY = env.DEPLOYER_PRIVATE_KEY;

const TRANSACTION_MONITOR_INTERVAL = env.TRANSACTION_MONITOR_INTERVAL || 60000;
const CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL = env.CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL || 60 * 1000;

const SQL_DISCONNECTED = "disconnected";

const WHITELIST_SLEEP = 60; // in seconds
const NON_WHITELIST_SLEEP = 3 * 60 * 60; // in seconds
const WHITELIST = "whitelist";
const NON_WHITELIST = "non_whitelist";

const L2_RATE_LIMIT = 500;
const L2_SLEEP_THRESH = 100;

class OptimismEnv {
  constructor() {
    this.L1Provider = new ethers.providers.JsonRpcProvider(L1_NODE_WEB3_URL);
    this.L2Provider = new ethers.providers.JsonRpcProvider(L2_NODE_WEB3_URL);

    this.L1wallet = new ethers.Wallet(DEPLOYER_PRIVATE_KEY).connect(this.L1Provider);

    this.MySQLHostURL = MYSQL_HOST_URL;
    this.MySQLPort = MYSQL_PORT;
    this.MySQLUsername = MYSQL_USERNAME;
    this.MySQLPassword = MYSQL_PASSWORD;
    this.MySQLDatabaseName = MYSQL_DATABASE_NAME;

    this.addressManagerAddress = ADDRESS_MANAGER_ADDRESS;
    this.OVM_L1CrossDomainMessenger = null;
    this.OVM_L1CrossDomainMessengerFast = null;
    this.OVM_L2CrossDomainMessenger = L2_MESSENGER_ADDRESS;

    this.numberBlockToFetch = 10000000;
    this.transactionMonitorInterval = TRANSACTION_MONITOR_INTERVAL;
    this.crossDomainMessageMonitorInterval = CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL;

    this.whitelistSleep = WHITELIST_SLEEP;
    this.nonWhitelistSleep = NON_WHITELIST_SLEEP;
    this.whitelistString = WHITELIST;
    this.nonWhitelistString = NON_WHITELIST;

    this.L2rateLimit = L2_RATE_LIMIT;
    this.L2sleepThresh = L2_SLEEP_THRESH;

    this.logger = new core_utils_1.Logger({ name: this.name });
    this.sleep = util.promisify(setTimeout);

    this.sqlDisconnected = SQL_DISCONNECTED;

    this.transactionMonitorSQL = false;
    this.crossDomainMessageMonitorSQL = false;

    this.databaseConnected = false;
    this.databaseConnectedMutex = new Mutex();

    this.OVM_L2CrossDomainMessengerContract = null;
  }

  async initOptimismEnv() {
    const addressManager = new ethers.Contract(
      this.addressManagerAddress,
      addressManagerJSON.abi,
      this.L1wallet
    );
    this.OVM_L1CrossDomainMessenger = await addressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger');
    this.OVM_L1CrossDomainMessengerFast = await addressManager.getAddress('OVM_L1CrossDomainMessengerFast');
    this.logger.info('Found OVM_L1CrossDomainMessenger and OVM_L1CrossDomainMessengerFast', {
      OVM_L1CrossDomainMessenger: this.OVM_L1CrossDomainMessenger,
      OVM_L1CrossDomainMessengerFast: this.OVM_L1CrossDomainMessengerFast,
    });
    this.OVM_L2CrossDomainMessengerContract = await loadContractFromManager({
      name: 'OVM_L2CrossDomainMessenger',
      Lib_AddressManager: addressManager,
      provider: this.L2Provider,
    });
    this.logger.info("Set up")
  }
}

module.exports = OptimismEnv;