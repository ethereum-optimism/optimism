# Services

## L2geth

It's forked from `go-etherum` with some modifications. 

> For the purpose of keeping the L2 data, we use volume for the L2geth image.

## Data Transport Layer

It syncs data from L1.

## Deployer

It deploys contracts on L1. These contracts help the communication between L1 and L2. The `Lib_AddressManager` contract has the addresses of all these important contract addresses.

```
{
  "AddressManager": "0x0787a989eDDeb40a64BBEAB94F2fF30CEb70A184",
  "OVM_CanonicalTransactionChain": "0xa833AAF1888d257b08F634F3c6FBc751Ce1A4815",
  "OVM_ChainStorageContainer:CTC:batches": "0xDb4CAB17Ffce0eD7B0A4F44Bb793b0C469C8Caa5",
  "OVM_ChainStorageContainer:CTC:queue": "0xd96c792e72A1D9A5F3584B14e26d86F17C27CB3b",
  "OVM_ChainStorageContainer:SCC:batches": "0x65fafD8A61a303FFD1d7EF265f439Ff452742cDe",
  "OVM_ExecutionManager": "0x99b2897C450A72267bfF439484Bae250009Aaaf7",
  "OVM_FraudVerifier": "0xd7b6CA71Bc11776A383359237c3995A0F43d4316",
  "OVM_L1CrossDomainMessenger": "0x9362c3e6fD23Eb8049c82f69BD013B8d92072701",
  "OVM_L1ETHGateway": "0x909C14780026E97eA67105C73cDD3DFD55a0ECc6",
  "OVM_L1MultiMessageRelayer": "0x8867577DC13647ce8e43D61b61E639421C9ADcc7",
  "OVM_SafetyChecker": "0x638a72ec9919A24684Fc328f8dE0887A80Bf2892",
  "OVM_StateCommitmentChain": "0xE5B5814Ca76FEA28d9c389cbf86CaDEfC5f25c20",
  "OVM_StateManagerFactory": "0x125c97d9F374624391bedd5d4e7F73F735646A2A",
  "OVM_StateTransitionerFactory": "0xf1263FCfCDC1b9b41C57Df7c7bE30530F29f4C22",
  "Proxy__OVM_L1CrossDomainMessenger": "0xaD05Bc932DA6f15a30976C533d309944c18C3b57",
  "Proxy__OVM_L1ETHGateway": "0xe3B6f4b17adE8809d292437951Af9521ad96B1cE",
  "OVM_BondManager": "0x6b32a2d71f90D5645613bBc0DbAe465e4Ce4b98C",
  "OVM_Sequencer": "0xE48E5b731FAAb955d147FA954cba19d93Dc03529",
  "Deployer": "0x122816e7A7AeB40601d0aC0DCAA8402F7aa4cDfA"
}
```

When we start the deployer service, it exposes two files. One is `addresses.json`, which returns the above addresses. Another one is `state-dump.latest.json`, which is the **abi** and address of the above contracts. Other services load `state-dump.latest.json` and use the **abi** to interact with these contracts.

> Deployer starts before all other services. 

> We should comment out the deploying code **packages/contracts/bin/deploy.ts** when we build the image for Rinkeby env. Otherwise, it will redeploy some contracts when the deployer service starts.

> It uses `DEPLOYER_PRIVATE_KEY` to deploy contracts. SCC and CTC only allow certain accounts to submit the data. These accounts are defined in **packages/contracts/bin/deploy.ts**. **SCC** only allows **ovmProposerAddress** to submit data and **CTC** only allows **ovmSequencerAddress** to push data.
>
> We use different accounts for **ovmSequencerAddress**, **ovmProposerAddress**, **ovmRelayerAddress** and **ovmAddressManagerOwner**.

## Batch Submitter

It submits the TX batch and state batch to contracts deployed by the deployer. The TX is submitted to `OVM_CanonicalTransactionChain` and the state batch is submitted to `OVM_StateCommitmentChain`. For any message that is sent between L1 and L2, the batch submitter submits TX batch first, then submits the state batch next.

We use two different accounts (**ovmSequencerAddress** and **ovmProposerAddress** ) to submit tx and state batch, respectively.

## Message relayer

The message relayer relays the message that is sent from the L2 contract to the L1 contract. The message relayer fetches the batches from `OVM_StateCommitmentChain`, then it gets the original message from L2 according to the block number of the batches. If L1 contracts use the standard **OVM_CrossDomainEnabled**, the message that is sent from the L2 contract to the L1 contract has a 7-days proof window. Once it passes 7 days, the message relayer will relay the message.

> How is the message sent from L2 to L1?
>
> The L2 and L1 contracts need to import **OVM_CrossDomainEnabled** to enable the communication between L1 and L2. L2 contracts send the message by using:
>
> ```
> bytes memory data = abi.encodeWithSelector(
>     L1Contract.L1ContractFunction.selector,
>     L1ContractFunctionVariable_1,
>     L1ContractFunctionVariable_2,
>     L1ContractFunctionVariable_3
> );
> 
> // Send calldata into L1
> sendCrossDomainMessage(
>     address(L1ContractAddress),
>     data, // message that the message relayer is going to relay
>     getFinalizeDepositL1Gas() 
> );
> ```

> The message relayer doesn't relay the message again if the relaying transaction has the nonce error. Therefore, we have to use different accounts for the **message-relayer service** and **message-relayer-fast service**.

## Message Relayer Fast

The message that is relayed by the **message-relayer service** has a 7-days proof window time. Therefore, we create the **message-relayer-fast** to bypass the proof window time. Using the **message-relayer-fast** requires the L1 contract to import the **OVM_FastCrossDomainEnabled**. The fast message relayer fetches the batches from `OVM_StateCommitmentChain` , just like the standard message relayer, but we add the whitelist system to restrict the messages that can be relayed to L1. Only messages that  target certain L1 contracts can be relayed by **message-relayer-fast service**. 

Since **message-relayer service** can get the messages that are supposed to be relayed by **message-relayer-fast service**, we add the blacklist system. The standard message relayer doesn't relay messages that target certain L1 contracts. It saves us gas.

# Understand Log

When we start the integration or production services, the most important test is sending the message between L1 and L2. Normally, the message can be easily sent from L1 to L2 if the batch submitter works correctly. Thus, I focus on how to debug the system when the message can't be sent from L2 to L1.

## Debugging steps

1. Make sure all services connect to the right **SCC** and **CTC**. The addressManagerAddresses of all services are the same.

   * Batch submitter

     ```javascript
     Configured batch submitter address: {addressManagerAddress:  0x0787a989eDDeb40a64BBEAB94F2fF30CEb70A184}
     Initialed new CTC: 0xa833AAF1888d257b08F634F3c6FBc751Ce1A4815
     ```

   * Message relayer and message relayer fast

     ```javascript
     Connecting to OVM_StateCommitmentChain...
     Connected to OVM_StateCommitmentChain: 0xE5B5814Ca76FEA28d9c389cbf86CaDEfC5f25c20
     Connecting to OVM_L1CrossDomainMessenger...
     Connected to OVM_L1CrossDomainMessenger: 0xaD05Bc932DA6f15a30976C533d309944c18C3b57
     // Message relayer and message relayer fast connect to two differnt OVM_L1CrossDomainMessangers.
     ```

2. When the test is running, check the log of the batch submitter first. Successfully submitting the batches should have the following log:

   ```javascript
   // Submit the tx batch first
   Submitted appendSequencerBatch transaction: {txHash: , from: }
   appendSequencerBatch transaction data: {data: }
   Submitted batch!
   // Submit the state root batch next
   Submitted appendStateBatch transaction: {txHash: from: }
   appendStateBatch transaction data: {data: }
   Submitted state root batch!
   ```

3. After the batch submitter submits the data, please check the log of the message relayer or message relayer fast according to the transaction type. Both message relayers should find a batch of the finalized transaction(s). The log of successfully relaying the message is

   ```
   Found a batch of finalized transaction(s), checking for more...
   Found finalized transactions
   Found a message sent during transaction
   Message not yet relayed. Attempting to generate a proof...
   Successfully generated a proof. Attempting to relay to Layer 1...
   Relay message transaction sent
   Relay message included in block
   Message successfully relayed to Layer 1!
   ```

   

   