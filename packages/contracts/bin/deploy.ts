#!/usr/bin/env -S node --require ts-node/register

const contracts = require('../src/index.ts');
const { providers, Wallet } = require('ethers');
const { JsonRpcProvider } = providers;

const env = process.env;
const key = env.DEPLOYER_PRIVATE_KEY;
const sequencerKey = env.SEQUENCER_PRIVATE_KEY;
const web3Url = env.L1_NODE_WEB3_URL || 'http://127.0.0.1:8545';
const MIN_TRANSACTION_GAS_LIMIT = env.MIN_TRANSACTION_GAS_LIMIT || 0;
const MAX_TRANSACTION_GAS_LIMIT = env.MAX_TRANSACTION_GAS_LIMIT || 1000000000;
const MAX_GAS_PER_QUEUE_PER_EPOCH = env.MAX_GAS_PER_QUEUE_PER_EPOCH || 250000000;
const SECONDS_PER_EPOCH = env.SECONDS_PER_EPOCH || 600;
let WHITELIST_OWNER = env.WHITELIST_OWNER;
const WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT = env.WHITELIST_ALLOW_ARBITRARY_CONTRACT_DEPLOYMENT || true;
const FORCE_INCLUSION_PERIOD_SECONDS = env.FORCE_INCLUSION_PERIOD_SECONDS || (30 * 60);

(async () => {
  if (typeof key === 'undefined')
    throw new Error('Must pass deployer key as DEPLOYER_PRIVATE_KEY');

  if (typeof sequencerKey === 'undefined')
    throw new Error('Must pass sequencer key as SEQUENCER_PRIVATE_KEY');

  const provider = new JsonRpcProvider(web3Url);
  const signer = new Wallet(key, provider);
  const sequencer = new Wallet(sequencerKey, provider);

  const chainid = await provider.send('eth_chainId', []);

  if (typeof WHITELIST_OWNER === 'undefined')
    WHITELIST_OWNER = signer;

  const result = await contracts.deploy({
    deploymentSigner: signer,
    transactionChainConfig: {
      forceInclusionPeriodSeconds: FORCE_INCLUSION_PERIOD_SECONDS,
      sequencer,
    },
    ovmGlobalContext: {
      ovmCHAINID: chainid
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
  for (const [name, contract] of Object.entries(result.contracts)) {
    out[name] = (contract as any).address;
  }
  console.log(JSON.stringify(out, null, 2));
})().catch(err => {
  console.log(JSON.stringify({error: err.message, stack: err.stack}, null, 2));
  process.exit(1);
});
