import { vi, test, expect, beforeEach } from 'vitest'
import {
  optimistABI,
  l1CrossDomainMessengerABI,
  l1CrossDomainMessengerAddress,
  optimismPortalABI,
  optimismPortalAddress
} from '@eth-optimism/contracts-ts'
// note I'm using 0.2.0 here
import { getLegacyProofsForL2Transaction } from '@eth-optimism/message-relayer'
import { WalletClient, createPublicClient, encodeFunctionData, http, keccak256, parseAbi, parseAbiItem } from 'viem'
import { getL2Client } from 'estimateFees'
import { optimism } from 'viem/dist/types/chains'
import {CrossChainMessage, CrossChainMessenger, MessageDirection} from '@eth-optimism/sdk'
import {ethers} from 'ethers'
import {hashWithdrawal} from '@eth-optimism/core-utils'

// to do this we need the old contract interface
// note I'm using a very old version of @eth-optimism/contracts
// note that we could simply precompute everything and simply have a mapping of withdrawal to whatever information we need
const oldMessengerInterface = require('@eth-optimism/old-contracts').getContractInterface('OVM_L2CrossDomainMessenger')


// I got this from https://raw.githubusercontent.com/ethereum-optimism/gateway/develop/packages/backend/src/data/ovm1Withdrawals_version1.json?token=GHSAT0AAAAAACGMCI43WSTVIA7BZL4NQPY6ZHX4WYA
// that originally came from the old gateway
const ovm1Withdrawals_version1 =   {
    "is_unclaimed": 1,
    "amount": "100000000000",
    "amount_tokens": "174876e800",
    "_l1Token": "0x0000000000000000000000000000000000000000",
    "_l2Token": "0x4200000000000000000000000000000000000006",
    "_from": "0xc04f345d5a50fe433de3235ca88333aba11b90a7",
    "_to": "0xc04f345d5a50fe433de3235ca88333aba11b90a7",
    "tx_hash": "0xc31d3ef563a9710728a7959c98c7781aa864956048b9627bf27fed3e7fe03e60",
    "block_hash": "0xadd3cf9da68dbb47b70ebb70b414a1739e9879dbdc90c103f30af9a64995d559",
    "block_number": 1990329,
    "block_timestamp": "2021-09-19 07:56:49.000 UTC",
    "symbol": "ETH",
    "data": "0x000000000000000000000000c04f345d5a50fe433de3235ca88333aba11b90a7000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000",
    "target": "0xc04f345d5a50fe433de3235ca88333aba11b90a7",
    "message_nonce": 8885
  } as const
// I got this from  https://raw.githubusercontent.com/ethereum-optimism/gateway/develop/packages/backend/src/data/ovm1Withdrawals_version1.json?token=GHSAT0AAAAAACGMCI43WSTVIA7BZL4NQPY6ZHX4WYA
// that originally came from the old gateway
// I don't think this is actually necessary they only both exist because I was copy pasting exactly without modifying the legacy code
const ovmWithdrawals = {
        "version": 1,
        "message_nonce": 8885,
        "days_since_initiated": 957,
        "symbol": "ETH",
        "amount_tokens": 1e-7,
        "amount_usd": 0.0001356,
        "l2_tx_hash": "0xc31d3ef563a9710728a7959c98c7781aa864956048b9627bf27fed3e7fe03e60",
        "l2_block_time": "2021-09-19T07:56:49Z",
        "target": "0xc04f345d5a50fe433de3235ca88333aba11b90a7",
        "_l1Token": "0x0000000000000000000000000000000000000000",
        "_l2Token": "0x4200000000000000000000000000000000000006",
        "all_withdrawals": 29914,
        "unclaimed_withdrawals_gt7": 13305,
        "usd_unclaimed_gt7": 17520072.364946086,
        "unclaimed_withdrawals_lte7": 84,
        "usd_unclaimed_lte7": 11313457.16728247
} as const
// finally this one came from https://raw.githubusercontent.com/ethereum-optimism/gateway/develop/packages/backend/src/data/ovmhistorical.json?token=GHSAT0AAAAAACGMCI43K27J2HZCDXTN4QDKZHX5SSA
const ovmHistorical = {
    // this is the from address
    "0xc04f345d5a50fe433de3235ca88333aba11b90a7": [
    {
        // this is the tx hash
      "0xc31d3ef563a9710728a7959c98c7781aa864956048b9627bf27fed3e7fe03e60": [
        {
          "messageNonce": 8885,
          // we really only care about this rawdata
          "rawData": "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000164cbd4ece900000000000000000000000099c9fc46f92e8a1c0dec1b1747d010903e884be10000000000000000000000004200000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000022b500000000000000000000000000000000000000000000000000000000000000a41532ec34000000000000000000000000c04f345d5a50fe433de3235ca88333aba11b90a7000000000000000000000000c04f345d5a50fe433de3235ca88333aba11b90a7000000000000000000000000000000000000000000000000000000174876e800000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
          "timestamp": "2021-09-19T07:56:49+00:00"
        }
      ]
    }
  ],
} as const

// get the event data
// this is copy pasted code from https://github.com/ethereum-optimism/gateway/blob/develop/packages/backend/src/routes/OvmWithdrawalsRoute.ts
// which itself was copy pasted from the old gateway bridge ui
// note we could precompute this for all legacy withdrawals ahead of time
const eventData = oldMessengerInterface.decodeEventLog(
    'SentMessage',
    ovmHistorical['0xc04f345d5a50fe433de3235ca88333aba11b90a7'][0]['0xc31d3ef563a9710728a7959c98c7781aa864956048b9627bf27fed3e7fe03e60'][0].rawData,
    [
      '0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326', // topic address
    ],
  )
  const messageHash = keccak256(eventData.message)
  console.log({ messageHash, eventData})

// use the event data to get the relay data
// note we could precompute this for all legacy withdrawals ahead of time
  const relayData = oldMessengerInterface.decodeFunctionData(
    'relayMessage',
    eventData.message,
  )

  // console logging everything because typescript atm is of type any so showing what data looks like
  console.log('relayData', {
  _message : relayData._message,
  _messageNonce : relayData._messageNonce,
  _sender : relayData._sender,
  _target : relayData._target,
  })

const l2PublicClient = createPublicClient({
    chain: optimism,
    transport: http('https://mainnet.optimism.io')
})

test('should be able to test if transaction has been fully relayed', () => {
    // to test that a message hash is successful just check the mapping(bytes32 => bool) public successfulMessages on CrossDomainMessenger
    // we will do a more robust test later though this is just instructive of how the contracts/messages work
    // this will also make the get message status test make more sense
    // the way we figure out if a message is claimed is also how we check if a message is proven (though proven is way more steps but same idea)
    const isSuccessful = l2PublicClient.readContract({
        abi: l1CrossDomainMessengerABI,
        address: l1CrossDomainMessengerAddress[1],
        functionName: 'successfulMessages',
        args: [messageHash],
    })
    expect(isSuccessful).toBe(false)
})

// Note this can be done using getMessageStatus in the sdk
// but we will show how it works at a low level so we have
//the correct intuition of what is going on
test('should be able to get message status prove and claim', async () => {
   /**
    * we could do this with the optimism sdk as follows
    * @example
    * const messageReceipt = await crossChainMessenger.getMessageReceipt(relayData._message, relayData._messageNonce)
    */
   // we will use viem to do it from scratch though

   // this is a rewrite of hashCrossDomaiNMessagev0 from @eth-optimism/core-utils
   // gonna use viem to show what is going on at a low level
   // IMPORTANT: at this point we can simply pass this
   // cross domain message into getMessageStatus in the sdk
   // and it will just work
   // we will keep going showing what to do at a lower level
   // to show how it works though
   const hashedCrossDomainMessageV0 = keccak256(
     encodeFunctionData({
         abi: parseAbi([
  'function relayMessage(address,address,bytes,uint256)',
  'function relayMessage(uint256,address,address,uint256,uint256,bytes)',
         ]),
         functionName: 'relayMessage',
         args: [relayData._target, relayData._sender, relayData._message, relayData._messageNonce]
     })
   )

   // ok now we got what the cross domain message should be based on the withdrawal information
   // we can use this to  see if there is a RelayedMessage event
   const relayedMessageEvents = await l2PublicClient.getLogs({
      address: l1CrossDomainMessengerAddress[10],
      event: parseAbiItem('event RelayedMessage(bytes32 indexed msgHash)'),
      args: {
        msgHash: hashedCrossDomainMessageV0
      }
   })
   // this figures out if the withdrawal is complete
  const isSuccessful = relayedMessageEvents.length > 0

  // we can also checkt o see if it failed
  const failedRelayedMessageEvents = await l2PublicClient.getLogs({
        address: l1CrossDomainMessengerAddress[10],
        event: parseAbiItem('event FailedRelayedMessage(bytes32 indexed msgHash)'),
        args: {
            msgHash: hashedCrossDomainMessageV0
        }
    }
  )

  // not that if it's failed we can retry so message status is retryable READY_FOR_RELAY
    const isFailed = !isSuccessful && failedRelayedMessageEvents.length > 0

    // so if we have no relay attempts we know the message is either in challenge period or ready to prove
    // for normal withdrawals we would check if state root is published yet but here we already know it has been since it's a legacy withdrawal
  // attempt to find the proven withdrawal
  // this is similar to how we checked for fully relayed on line 114
  // we check optimism portal for this
  // this is by far the most complicated part
  // gonna partially use the sdk to make my work here easier
  const sdk = new CrossChainMessenger({
    l1SignerOrProvider: new ethers.providers.JsonRpcProvider(process.env.RPC_URL_L1),
    l2SignerOrProvider: new ethers.providers.JsonRpcProvider(process.env.RPC_URL_L2),
    l1ChainId: 1,
    l2ChainId: 10,
  })

  // copying logic from optimism sdk here
  const crossChainmessage: CrossChainMessage = {
    blockNumber: ovm1Withdrawals_version1.block_number,
    direction: MessageDirection.L2_TO_L1,
    // assuming the log index is 0
    // should be 0 for all or almost all legacy withdrawals
    // but worth double checking that's true
    logIndex: 0,
    message: relayData._message,
    messageNonce: relayData._messageNonce,
    // legacy withdrawals don't have the concept of a value or gasLimit
    // see https://github.com/ethereum-optimism/optimism/blob/b25169ea7daec694d7d6d7f8b107ec66f41dd46b/op-chain-ops/crossdomain/legacy_withdrawal.go
    value: ethers.BigNumber.from(0),
    minGasLimit: ethers.BigNumber.from(0),
    sender: relayData._sender,
    target: relayData._target,
    transactionHash: ovm1Withdrawals_version1.tx_hash,
  }

  // too many lines of code in this function for me to write
  // here without messing it up and timeboxing this so just
  // gonna use the sdk here
  // internally what this will do is call toBedrockCrossChainMessage
  // to transform into a bedrock cross chain message
  // it will encode the message using encodeCrossDomainMessageV1
  // it will get the migratedWithdrawalGasLimit which is based on the data in the message
  // finally it returns the low level message
  const withdrawalLowLevelMessage = await sdk.toLowLevelMessage(
    crossChainmessage
  )

  // gonna use @eth-optimism/core-utils here to do this faster
  // under the hood this is simply abiEncoding the low level message
  // into [uint256, address, address, uint256 uint256, bytes]
  // and then keccak256ing the result
  const withdrawalMessageHash = hashWithdrawal(
    withdrawalLowLevelMessage.messageNonce,
    withdrawalLowLevelMessage.sender,
    withdrawalLowLevelMessage.target,
    withdrawalLowLevelMessage.value,
    withdrawalLowLevelMessage.minGasLimit,
    withdrawalLowLevelMessage.message
  )

  // returns [bytes32 outputRoot, uint128 timestamp, uint128 l2OutputIndex]
  const provenWithdrawal = await l2PublicClient.readContract({
    abi: optimismPortalABI,
    address: optimismPortalAddress[10],
    functionName: 'provenWithdrawals',
    args: [withdrawalMessageHash as `0x${string}`],
  })
  const timestamp = provenWithdrawal[1]
    // if timestamp is 0 then it's not proven
    const isReadyTOProve = timestamp === BigInt(0)

    const ONE_SECOND = 1000
    // this is wrong replace me with the actual challenge period in seconds
    const challengePeriodInSeconds = BigInt(420 * ONE_SECOND)
    const now = BigInt(Date.now())
    const provenTime = timestamp * BigInt(ONE_SECOND)
    const isChallengePeriodOver = now - provenTime > challengePeriodInSeconds

    // remember it's also ready to claim if a failed tx happened that can be replayed as we checked way further up
    const isReadyToClaim = isFailed || (!isSuccessful && !isReadyTOProve && isChallengePeriodOver)

    const isInChallengePeriod = !isReadyTOProve && !isChallengePeriodOver

    // this is how you do getMessageStatus at a low level
    // with a legacy withdrawal
    // This can be done via passing in a full message
    // to getMessageStatus in the sdk but hopefully
    // this shows what is going on
    // note if using the sdk you gotta pass the full message
    // not just the tx hash because no rpc exists to fetch the
    // l2 logs needed to fill in the message
    // this is where the legacy json info comes in handy
    // as it has all the information we need to construct the message
    console.log({
        isChallengePeriodOver,
        isSuccessful,
        isFailed,
        isInChallengePeriod,
        isReadyTOProve,
        isReadyToClaim,
    })

    async function howToProve() {
        // pass into proveMessage
        // under the hood this will generate a trie proof
        // using the BedrockMessagePasser
        // After it generates the proof it will submit
        // the tx data proof data and low leval message
        // to the optimism portal
        // note the sdk needs to have a l1Signer for this to work
        await sdk.proveMessage(crossChainmessage)
    }

    // will show how you would claim from this point
    function howToClaim() {
      // similar to proveMessage we can call finalizeMessage to do this
      // note this should be a wallet client not a publicClient
      // just showing what the sdk is doing
      (l2PublicClient as any as WalletClient).writeContract({
        abi: l1CrossDomainMessengerABI,
        address: l1CrossDomainMessengerAddress[10],
        functionName: 'relayMessage',
        args: [withdrawalLowLevelMessage.messageNonce.toBigInt(), withdrawalLowLevelMessage.sender as `0x${string}`, withdrawalLowLevelMessage.target as `0x${string}`, withdrawalLowLevelMessage.value.toBigInt(), withdrawalLowLevelMessage.minGasLimit.toBigInt(), withdrawalLowLevelMessage.message as `0x${string}`],
        account: '0x420',
        chain: optimism,
        value: BigInt(0)
      })
    }
})