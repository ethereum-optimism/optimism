#!/usr/bin/env node

const contracts = require('../build/src/contract-deployment/deploy');
const { providers, Wallet, utils, ethers } = require('ethers');
const { LedgerSigner } = require('@ethersproject/hardware-wallets');
const { JsonRpcProvider } = providers;

const env = process.env;
const key = env.DEPLOYER_PRIVATE_KEY;
const sequencerKey = env.SEQUENCER_PRIVATE_KEY;
let SEQUENCER_ADDRESS = env.SEQUENCER_ADDRESS;
const web3Url = env.L1_NODE_WEB3_URL || 'http://127.0.0.1:8545';
const DEPLOY_TX_GAS_LIMIT = env.DEPLOY_TX_GAS_LIMIT || 5000000;
const MIN_TRANSACTION_GAS_LIMIT = env.MIN_TRANSACTION_GAS_LIMIT || 50000;
const MAX_TRANSACTION_GAS_LIMIT = env.MAX_TRANSACTION_GAS_LIMIT || 9000000;
const MAX_GAS_PER_QUEUE_PER_EPOCH = env.MAX_GAS_PER_QUEUE_PER_EPOCH || 250000000;
const SECONDS_PER_EPOCH = env.SECONDS_PER_EPOCH || 0;
const WAIT_FOR_RECEIPTS = env.WAIT_FOR_RECEIPTS === 'true';
let WHITELIST_OWNER = env.WHITELIST_OWNER;
const WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT = env.WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT || true;
const FORCE_INCLUSION_PERIOD_SECONDS = env.FORCE_INCLUSION_PERIOD_SECONDS || 2592000; // 30 days
const FRAUD_PROOF_WINDOW_SECONDS = env.FRAUD_PROOF_WINDOW_SECONDS || (60 * 60 * 24 * 7); // 7 days
const SEQUENCER_PUBLISH_WINDOW_SECONDS = env.SEQUENCER_PUBLISH_WINDOW_SECONDS || (60 * 30); // 30 min
const CHAIN_ID = env.CHAIN_ID || 420; // layer 2 chainid
const USE_LEDGER = env.USE_LEDGER || false;
const ADDRESS_MANAGER_ADDRESS = env.ADDRESS_MANAGER_ADDRESS || undefined;
const HD_PATH = env.HD_PATH || utils.defaultPath;
const BLOCK_TIME_SECONDS = env.BLOCK_TIME_SECONDS || 15;
const L2_CROSS_DOMAIN_MESSENGER_ADDRESS =
  env.L2_CROSS_DOMAIN_MESSENGER_ADDRESS || '0x4200000000000000000000000000000000000007';
let RELAYER_ADDRESS = env.RELAYER_ADDRESS || '0x0000000000000000000000000000000000000000';
const RELAYER_PRIVATE_KEY = env.RELAYER_PRIVATE_KEY;

(async () => {
  const provider = new JsonRpcProvider(web3Url);
  let signer;

  // Use the ledger for the deployer
  if (USE_LEDGER) {
    signer = new LedgerSigner(provider, 'default', HD_PATH);
  } else  {
    if (typeof key === 'undefined')
      throw new Error('Must pass deployer key as DEPLOYER_PRIVATE_KEY');
    signer = new Wallet(key, provider);
  }

  if (SEQUENCER_ADDRESS) {
    if (!utils.isAddress(SEQUENCER_ADDRESS))
      throw new Error(`Invalid Sequencer Address: ${SEQUENCER_ADDRESS}`);
  } else {
    if (!sequencerKey)
      throw new Error('Must pass sequencer key as SEQUENCER_PRIVATE_KEY');
    const sequencer = new Wallet(sequencerKey, provider);
    SEQUENCER_ADDRESS = await sequencer.getAddress();
  }

  if (typeof WHITELIST_OWNER === 'undefined')
    WHITELIST_OWNER = signer;

  // Use the address derived from RELAYER_PRIVATE_KEY if a private key
  // is passed. Using the zero address as the relayer address will mean
  // there is no relayer authentication.
  if (RELAYER_PRIVATE_KEY) {
    if (!utils.isAddress(RELAYER_ADDRESS))
      throw new Error(`Invalid Relayer Address: ${RELAYER_ADDRESS}`);
    const relayer = new Wallet(RELAYER_PRIVATE_KEY, provider);
    RELAYER_ADDRESS = await relayer.getAddress();
  }

  const result = await contracts.deploy({
    deploymentSigner: signer,
    transactionChainConfig: {
      forceInclusionPeriodSeconds: FORCE_INCLUSION_PERIOD_SECONDS,
      sequencer: SEQUENCER_ADDRESS,
      forceInclusionPeriodBlocks: Math.ceil(FORCE_INCLUSION_PERIOD_SECONDS/BLOCK_TIME_SECONDS),
    },
    stateChainConfig: {
      fraudProofWindowSeconds: FRAUD_PROOF_WINDOW_SECONDS,
      sequencerPublishWindowSeconds: SEQUENCER_PUBLISH_WINDOW_SECONDS,
    },
    ovmGlobalContext: {
      ovmCHAINID: CHAIN_ID,
      L2CrossDomainMessengerAddress: L2_CROSS_DOMAIN_MESSENGER_ADDRESS
    },
    l1CrossDomainMessengerConfig: {
      relayerAddress: RELAYER_ADDRESS,
    },
    ovmGasMeteringConfig: {
      minTransactionGasLimit: MIN_TRANSACTION_GAS_LIMIT,
      maxTransactionGasLimit: MAX_TRANSACTION_GAS_LIMIT,
      maxGasPerQueuePerEpoch: MAX_GAS_PER_QUEUE_PER_EPOCH,
      secondsPerEpoch: SECONDS_PER_EPOCH
    },
    whitelistConfig: {
      owner: WHITELIST_OWNER,
      allowArbitraryContractDeployment: WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT
    },
    deployOverrides: {
      gasLimit: DEPLOY_TX_GAS_LIMIT
    },
    waitForReceipts: WAIT_FOR_RECEIPTS,
    addressManager: ADDRESS_MANAGER_ADDRESS,
  });

  const { failedDeployments, AddressManager } = result;
  if (failedDeployments.length !== 0)
    throw new Error(`Contract deployment failed: ${failedDeployments.join(',')}`);

  const out = {};
  out.AddressManager = AddressManager.address;
  out.OVM_Sequencer = SEQUENCER_ADDRESS;
  out.Deployer = await signer.getAddress()
  for (const [name, contract] of Object.entries(result.contracts)) {
    out[name] = contract.address;
  }
  console.log(JSON.stringify(out, null, 2));
})().catch(err => {
  console.log(JSON.stringify({error: err.message, stack: err.stack}, null, 2));
  process.exit(1);
});
