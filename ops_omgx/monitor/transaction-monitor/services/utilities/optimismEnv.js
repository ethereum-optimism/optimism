#!/usr/bin/env node

const ethers = require('ethers');
const core_utils_1 = require('@eth-optimism/core-utils');
const Mutex = require('async-mutex').Mutex;
const { loadContractFromManager, getContractFactory } = require('@eth-optimism/contracts')
const fetch = require("node-fetch");
const { Watcher } = require('@eth-optimism/watcher');

const addressManagerJSON = require('../../artifacts/contracts/optimistic-ethereum/libraries/resolver/Lib_AddressManager.sol/Lib_AddressManager.json');
const L1LiquidityPoolJson = require('../../artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json');
const L2LiquidityPoolJson = require('../../artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json');
const OVM_L1StandardBridgeJson = require('../../artifacts/contracts/optimistic-ethereum/OVM/bridge/tokens/OVM_L1StandardBridge.sol/OVM_L1StandardBridge.json');
const OVM_L2StandardBridgeJson = require('../../artifacts-ovm/contracts/optimistic-ethereum/OVM/bridge/tokens/OVM_L2StandardBridge.sol/OVM_L2StandardBridge.json');

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

const TRANSACTION_MONITOR_INTERVAL = env.TRANSACTION_MONITOR_INTERVAL || 3 * 1000;
const CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL = env.CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL || 5 * 1000;
const STATE_ROOT_MONITOR_INTERVAL = env.STATE_ROOT_MONITOR_INTERVAL || 5 * 1000;
const EXIT_MONITOR_INTERVAL = env.EXIT_MONITOR_INTERVAL || 3 * 1000;
const L1_BRIDGE_MONITOR_INTERVAL = env.L1_BRIDGE_MONITOR_INTERVAL || 3 * 1000;

const SQL_DISCONNECTED = "disconnected";

const WHITELIST_SLEEP = 60; // in seconds
const NON_WHITELIST_SLEEP = 3 * 60 * 60; // in seconds
const WHITELIST = "whitelist";
const NON_WHITELIST = "non_whitelist";

const L2_RATE_LIMIT = 500;
const L2_SLEEP_THRESH = 100;

const FILTER_ENDPOINT = env.FILTER_ENDPOINT;

const STATE_ROOT_MONITOR_START_BLOCK = env.STATE_ROOT_MONITOR_START_BLOCK || 0;
const STATE_ROOT_MONITOR_LOG_INTERVAL = env.STATE_ROOT_MONITOR_LOG_INTERVAL || 2000;

const BOBA_DEPLOYER_URL = env.BOBA_DEPLOYER_URL;
const L1LiquidityPoolAddress = env.PROXY__L1_LIQUIDITY_POOL_ADDRESS;
const L2LiquidityPoolAddress = env.PROXY__L2_LIQUIDITY_POOL_ADDRESS;

const EXIT_MONITOR_LOG_INTERVAL = env.STATE_ROOT_MONITOR_LOG_INTERVAL || 2000;

const L1_BRIDGE_MONITOR_START_BLOCK = env.L1_BRIDGE_MONITOR_START_BLOCK || 0;
const L1_BRIDGE_MONITOR_LOG_INTERVAL = env.L1_BRIDGE_MONITOR_LOG_INTERVAL || 2000;

// seconds
const L1_CROSS_DOMAIN_MESSAGE_WAITING_TIME = env.L1_CROSS_DOMAIN_MESSAGE_WAITING_TIME || 30;
const L2_CROSS_DOMAIN_MESSAGE_WAITING_TIME = env.L2_CROSS_DOMAIN_MESSAGE_WAITING_TIME || 60;

const OVM_L2_STANDARD_BRIDGE_ADDRESS = env.OVM_L2_STANDARD_BRIDGE_ADDRESS || "0x4200000000000000000000000000000000000010"
const OVM_L2_CROSS_DOMAIN_MESSENGER = "0x4200000000000000000000000000000000000007";

class OptimismEnv {
  constructor() {
    this.L1Provider = new ethers.providers.JsonRpcProvider(L1_NODE_WEB3_URL);
    this.L2Provider = new ethers.providers.JsonRpcProvider(L2_NODE_WEB3_URL);

    this.MySQLHostURL = MYSQL_HOST_URL;
    this.MySQLPort = MYSQL_PORT;
    this.MySQLUsername = MYSQL_USERNAME;
    this.MySQLPassword = MYSQL_PASSWORD;
    this.MySQLDatabaseName = MYSQL_DATABASE_NAME;

    this.addressManagerAddress = ADDRESS_MANAGER_ADDRESS;
    this.OVM_L1CrossDomainMessenger = null;
    this.OVM_L1CrossDomainMessengerFast = null;
    this.OVM_L2CrossDomainMessenger = L2_MESSENGER_ADDRESS;
    this.OVM_L2StandardBridge = OVM_L2_STANDARD_BRIDGE_ADDRESS;

    this.numberBlockToFetch = 10000000;
    this.transactionMonitorInterval = TRANSACTION_MONITOR_INTERVAL;
    this.crossDomainMessageMonitorInterval = CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL;
    this.stateRootMonitorInterval = STATE_ROOT_MONITOR_INTERVAL;
    this.exitMonitorInterval = EXIT_MONITOR_INTERVAL;
    this.l1BridgeMonitorInterval = L1_BRIDGE_MONITOR_INTERVAL;

    this.whitelistSleep = WHITELIST_SLEEP;
    this.nonWhitelistSleep = NON_WHITELIST_SLEEP;
    this.whitelistString = WHITELIST;
    this.nonWhitelistString = NON_WHITELIST;

    this.L2rateLimit = L2_RATE_LIMIT;
    this.L2sleepThresh = L2_SLEEP_THRESH;

    this.logger = new core_utils_1.Logger({ name: this.name });

    this.sqlDisconnected = SQL_DISCONNECTED;

    this.transactionMonitorSQL = false;
    this.crossDomainMessageMonitorSQL = false;

    this.databaseConnected = false;
    this.databaseConnectedMutex = new Mutex();

    this.OVM_L2CrossDomainMessengerContract = null;
    this.OVM_StateCommitmentChainContract = null;
    this.L1LiquidityPoolContract = null;
    this.L2LiquidityPoolContract = null;
    this.OVM_L1StandardBridgeContract = null;
    this.OVM_L2StandardBridgeContract = null;

    this.L1LiquidityPoolInterface = null;
    this.OVM_L1StandardBridgeInterface = null;

    this.filterEndpoint = FILTER_ENDPOINT;

    this.stateRootMonitorStartBlock = STATE_ROOT_MONITOR_START_BLOCK;
    this.stateRootMonitorLogInterval = STATE_ROOT_MONITOR_LOG_INTERVAL;

    this.exitMonitorLogInterval = EXIT_MONITOR_LOG_INTERVAL;

    this.l1BridgeMonitorStartBlock = L1_BRIDGE_MONITOR_START_BLOCK;
    this.l1BridgeMonitorLogInterval = L1_BRIDGE_MONITOR_LOG_INTERVAL;

    this.l1CrossDomainMessageWaitingTime = L1_CROSS_DOMAIN_MESSAGE_WAITING_TIME;
    this.l2CrossDomainMessageWaitingTime = L2_CROSS_DOMAIN_MESSAGE_WAITING_TIME;

    this.watcher = null;
  }

  async initOptimismEnv() {
    const addressManager = new ethers.Contract(
      this.addressManagerAddress,
      addressManagerJSON.abi,
      this.L1Provider
    );

    // Get addresses
    this.OVM_L1CrossDomainMessenger = await addressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger');
    this.OVM_L1CrossDomainMessengerFast = await addressManager.getAddress('Proxy__OVM_L1CrossDomainMessengerFast');
    this.Proxy__OVM_L1StandardBridge = await addressManager.getAddress('Proxy__OVM_L1StandardBridge');

    this.logger.info('Found OVM_L1CrossDomainMessenger, OVM_L1CrossDomainMessengerFast and Proxy__OVM_L1StandardBridge', {
      OVM_L1CrossDomainMessenger: this.OVM_L1CrossDomainMessenger,
      OVM_L1CrossDomainMessengerFast: this.OVM_L1CrossDomainMessengerFast,
      Proxy__OVM_L1StandardBridge: this.Proxy__OVM_L1StandardBridge
    });

    // Load L2 CDM
    this.OVM_L2CrossDomainMessengerContract = await loadContractFromManager({
      name: 'OVM_L2CrossDomainMessenger',
      Lib_AddressManager: addressManager,
      provider: this.L2Provider,
    });
    // Load SCC
    this.OVM_StateCommitmentChainContract = await loadContractFromManager({
      name: 'OVM_StateCommitmentChain',
      Lib_AddressManager: addressManager,
      provider: this.L1Provider,
    });
    // Load L1 Standard Bridge
    this.OVM_L1StandardBridgeContract = new ethers.Contract(
      this.Proxy__OVM_L1StandardBridge,
      OVM_L1StandardBridgeJson.abi,
      this.L1Provider
    )
    // Interface
    this.L1LiquidityPoolInterface = new ethers.utils.Interface(L1LiquidityPoolJson.abi);
    this.OVM_L1StandardBridgeInterface = new ethers.utils.Interface(OVM_L1StandardBridgeJson.abi);

    if (BOBA_DEPLOYER_URL) {
      const response = await fetch(BOBA_DEPLOYER_URL);
      const addresses = await response.json();

      this.L1LiquidityPoolAddress = addresses.Proxy__L1LiquidityPool;
      this.L2LiquidityPoolAddress = addresses.Proxy__L2LiquidityPool ;
    } else {
      this.L1LiquidityPoolAddress = L1LiquidityPoolAddress;
      this.L2LiquidityPoolAddress = L2LiquidityPoolAddress;
    }
    this.logger.info('Found L1LiquidityPool and L2LiquidityPool', {
      L1LiquidityPoolAddress: this.L1LiquidityPoolAddress,
      L2LiquidityPoolAddress: this.L2LiquidityPoolAddress,
    });
    // Load L1 LP
    this.L1LiquidityPoolContract = new ethers.Contract(
      this.L1LiquidityPoolAddress,
      L1LiquidityPoolJson.abi,
      this.L1Provider
    )
    // Load L2 LP
    this.L2LiquidityPoolContract = new ethers.Contract(
      this.L2LiquidityPoolAddress,
      L2LiquidityPoolJson.abi,
      this.L2Provider
    )
    // Load L2 Standard Bridge
    this.OVM_L2StandardBridgeContract = new ethers.Contract(
      this.OVM_L2StandardBridge,
      OVM_L2StandardBridgeJson.abi,
      this.L2Provider
    )

    // watcher
    this.watcher = new Watcher({
      l1: {
        provider: this.L1Provider,
        messengerAddress: this.OVM_L1CrossDomainMessenger,
      },
      l2: {
        provider: this.L2Provider,
        messengerAddress: OVM_L2_CROSS_DOMAIN_MESSENGER,
      },
    });

    this.logger.info("Set up")
  }
}

module.exports = OptimismEnv;
