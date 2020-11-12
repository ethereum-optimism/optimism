#!/usr/bin/env node

const contracts = require('../build/src/contract-deployment/deploy');
const { providers, Wallet, utils } = require('ethers');
const { LedgerSigner } = require('@ethersproject/hardware-wallets');
const { JsonRpcProvider } = providers;

const env = process.env;
const key = env.DEPLOYER_PRIVATE_KEY;
const sequencerKey = env.SEQUENCER_PRIVATE_KEY;
let SEQUENCER_ADDRESS = env.SEQUENCER_ADDRESS;
const web3Url = env.L1_NODE_WEB3_URL || 'http://127.0.0.1:8545';
const MIN_TRANSACTION_GAS_LIMIT = env.MIN_TRANSACTION_GAS_LIMIT || 0;
const MAX_TRANSACTION_GAS_LIMIT = env.MAX_TRANSACTION_GAS_LIMIT || 1000000000;
const MAX_GAS_PER_QUEUE_PER_EPOCH = env.MAX_GAS_PER_QUEUE_PER_EPOCH || 250000000;
const SECONDS_PER_EPOCH = env.SECONDS_PER_EPOCH || 600;
let WHITELIST_OWNER = env.WHITELIST_OWNER;
const WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT = env.WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT || true;
const FORCE_INCLUSION_PERIOD_SECONDS = env.FORCE_INCLUSION_PERIOD_SECONDS || (30 * 60);
const CHAIN_ID = env.CHAIN_ID || 420; // layer 2 chainid
const USE_LEDGER = env.USE_LEDGER || false;
const HD_PATH = env.HD_PATH || utils.defaultPath;
const L2_CROSS_DOMAIN_MESSENGER_ADDRESS =
  env.L2_CROSS_DOMAIN_MESSENGER_ADDRESS || '0xdeaddeaddeaddeaddeaddeaddeaddeaddead0000';

(async () => {
  const provider = new JsonRpcProvider(web3Url);
  let signer;

  if (USE_LEDGER) {
    signer = new LedgerSigner(provider, 'default', HD_PATH);
  } else  {
    if (typeof key === 'undefined')
      throw new Error('Must pass deployer key as DEPLOYER_PRIVATE_KEY');
    signer = new Wallet(key, provider);
  }

  if (SEQUENCER_ADDRESS) {
    if (!utils.isAddress(SEQUENCER_ADDRESS))
      throw new Error(`Invalid Sequencer Address: ${SEQUENCER_ADDRESS}`)
  } else {
    if (!sequencerKey)
      throw new Error('Must pass sequencer key as SEQUENCER_PRIVATE_KEY');
    const sequencer = new Wallet(sequencerKey, provider);
    SEQUENCER_ADDRESS = await sequencer.getAddress()
  }

  if (typeof WHITELIST_OWNER === 'undefined')
    WHITELIST_OWNER = signer;

  const result = await contracts.deploy({
    deploymentSigner: signer,
    transactionChainConfig: {
      forceInclusionPeriodSeconds: FORCE_INCLUSION_PERIOD_SECONDS,
      sequencer: SEQUENCER_ADDRESS,
    },
    ovmGlobalContext: {
      ovmCHAINID: CHAIN_ID,
      L2CrossDomainMessengerAddress: L2_CROSS_DOMAIN_MESSENGER_ADDRESS
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
