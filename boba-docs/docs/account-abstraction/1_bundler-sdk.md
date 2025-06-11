# Bundler SDK

This section documents the usage of the Bundler SDK, that is a major component of Account Abstraction.

## Create and Send UserOperation

A UserOperation in simple terms is a pseudo-transaction object that expresses an user's intent.

This package provides 2 APIs for using UserOperations:

* Low-level "walletAPI"
* High-level Provider

Make sure you understand both of them, to use the one that is suited best for your use case.

## Low level API

### BaseAccountAPI

An abstract [base-class](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/bundler\_sdk/src/BaseAccountAPI.ts) to create UserOperations for a contract wallet.

### SimpleAccountAPI

An implementation of the `BaseAccountAPI`, for the SimpleAccount sample of account-abstraction.

**constructor()**

```ts
interface SimpleAccountApiParams {
    factoryAddress?: string;
    owner: Signer;
    index: number; // default: 0

    // inherited from BaseAccountApiParams
    provider: Provider; // @ethersproject/providers
    entryPointAddress: string;
    entryPointWrapperAddress?: string;
    accountAddress?: string;
    overheads?: Partial<GasOverheads>;
    paymasterAPI?: PaymasterAPI;
}
```

**Usage**

Note that SimpleAccountAPI either needs the `accountAddress` or the `factoryAddress` to be supplied. If the `factoryAddress` is supplied, also supply a `entryPointWrapperAddress`. To address the lack of support for 'Custom reverts' in v2 of the network, the sdk would route the call through the entryPointWrapperAddress in order to compute the account address that will be deployed.

If `accountAddress` is passed, the account is used as a sender when generating the userOp If `factoryAddress` is passed, the account will be generated on the fly. The userOp will include initCode and the precomputed address of the account and include it in the userOp.

The low-level approach above can be used as follows:

```typescript
owner = provider.getSigner()
const walletAPI = new SimpleAccountAPI({
    provider,
    entryPointAddress,
    owner,
    accountAddress
})
const op = await walletAPI.createSignedUserOp({
  target: recipient.address,
  data: recipient.interface.encodeFunctionData('something', ['hello'])
})
```

or with a SimpleAccountFactory-

```typescript
owner = provider.getSigner()
const walletAPI = new SimpleAccountAPI({
    provider,
    entryPointAddress,
    owner,
    entryPointWrapperAddress,
    factoryAddress
})
const op = await walletAPI.createSignedUserOp({
  target: recipient.address,
  data: recipient.interface.encodeFunctionData('something', ['hello'])
})
```

**PaymasterAPI**

Add `paymasterAndData` to UserOp.

```ts
  accountAPI.paymasterAPI = new PaymasterAPI({
    paymasterAndData: null // your value
})
```

Exemplary `paymasterAndData` value:

```ts
paymasterAndData = hexConcat([
      BobaDepositPaymaster.address,
      hexZeroPad(L2BOBAToken.address, 20),
    ])
```

After adding the PaymasterAPI you can sign your user operation as usual.

**PaymasterAPI:getPaymasterAndData(Partial\<UserOperationStruct>)**

Returns `paymasterAndData` of given UserOp. Returns `0x` if empty.

**getAccountInitCode()**

Return the value to put into the "initCode" field, if the contract is not yet deployed. This value holds the "factory" address, followed by this account's information.

```ts
getAccountInitCode(): Promise<string>
```

**getNonce()**

Return current account's nonce.

```ts
getNonce(): Promise<BigNumber>
```

**encodeExecute()**

Encode the call from entryPoint through our account to the target contract.

```ts
encodeExecute (target: string, value: BigNumberish, data: string): Promise<string>
```

**signUserOpHash()**

Sign a userOp's hash (userOpHash).

```ts
signUserOpHash (userOpHash: string): Promise<string>
```

**checkAccountPhantom()**

Check if the contract is already deployed.

```ts
checkAccountPhantom(): Promise<boolean>
```

**getCounterFactualAddress()**

Calculate the account address even before it is deployed.

```ts
getCounterFactualAddress (): Promise<string>
```

**getInitCode()**

Return initCode value to add into the UserOp. (either deployment code, or empty hex if contract already deployed)

```ts
getInitCode(): Promise<string>
```

**getVerificationGasLimit()**

Return maximum gas used for verification. NOTE: createUnsignedUserOp will add to this value the cost of creation, if the contract is not yet created.

```ts
getVerificationGasLimit(): Promise<BigNumberish>
```

**getPreVerificationGas()**

Should cover cost of putting calldata on-chain, and some overhead. Actual overhead depends on the expected bundle size.

```ts
getVerificationGasLimit(): Promise<BigNumberish>
```

**getUserOpHash()**

Return userOpHash for signing. This value matches entryPoint.getUserOpHash (calculated off-chain, to avoid a view call)

```ts
getUserOpHash(userOp: UserOperationStruct): Promise<string>
```

**getAccountAddress()**

```ts
getAccountAddress(): Promise<string>
```

Return the account's address. This value is valid even before deploying the contract.

**createUnsignedUserOp()**

Create a UserOperation, filling all details (except signature)

* if account is not yet created, add initCode to deploy it.
* if gas or nonce are missing, read them from the chain (note that we can't fill gaslimit before the account is created)

```ts
createUnsignedUserOp (info: TransactionDetailsForUserOp): Promise<UserOperationStruct>
```

**signUserOp()**

Sign the filled userOp.

```ts
signUserOp (userOp: UserOperationStruct): Promise<UserOperationStruct>
```

**createSignedUserOp()**

Helper method: create and sign a user operation.

```ts
createSignedUserOp (info: TransactionDetailsForUserOp): Promise<UserOperationStruct>
```

**getUserOpReceipt()**

Get the transaction that has this userOpHash mined, or null if not found.

```ts
getUserOpReceipt (userOpHash: string, timeout = 30000, interval = 5000): Promise<string | null>
```

***

## High-Level Provider API

A simplified mode that doesn't require a different wallet extension. Instead, the current provider's account is used as wallet owner by calling its "Sign Message" operation.

This can only work for wallets that use an EIP-191 ("Ethereum Signed Message") signatures (like our sample SimpleWallet) Also, the UX is not great (the user is asked to sign a hash, and even the wallet address is not mentioned, only the signer)

### wrapProvider

Wrap an existing provider to tunnel requests through Account Abstraction.

```ts
async function wrapProvider(
  originalProvider: JsonRpcProvider, // @ethersproject/providers
  config: ClientConfig,
  originalSigner: Signer = originalProvider.getSigner(), // @ethersproject/abstract-signer
  entryPointWrapperAddress: string, // must be passed
  wallet?: Wallet, // ethers, must be passed
): Promise<ERC4337EthersProvider>
```

### ClientConfig

```ts
interface ClientConfig {
  /**
   * the entry point to use
   */
  entryPointAddress: string
  /**
   * url to the bundler
   */
  bundlerUrl: string
  /**
   * if set, use this pre-deployed wallet.
   * (if not set, use getSigner().getAddress() to query the "counterfactual" address of wallet.
   *  you may need to fund this address so the wallet can pay for its own creation)
   */
  walletAddres?: string
  /**
   * if set, call just before signing.
   */
  paymasterAPI?: PaymasterAPI
}
```

### Usage

Since- a) using a remote signer with eth\_sendTransaction is not supported on Boba, transactions would need to be sent from an ethers.wallet (object), for the deterministic deployment of SimpleAccountFactory. This is not a requirement if the SimpleAccountFactory has already been deployed b) wrapProvider uses the low level API internally, custom reverts were not supported in the v2 of the network and the sdk relies on the entryPointWrapperAddress to compute the account address that will be deployed

wrapProvider must be passed the parameters `entryPointWrapperAddress` and `wallet` on Boba

The high-level provider api can be used as follows:

```typescript
import { wrapProvider } from '@account-abstraction/sdk'

//use this account as wallet-owner (which will be used to sign the requests)
const signer = provider.getSigner()
const config = {
  chainId: await provider.getNetwork().then(net => net.chainId),
  entryPointAddress,
  bundlerUrl: 'http://localhost:3000/rpc'
}
const aaProvider = await wrapProvider(provider, config, aasigner, entryPointWrapperAddress, wallet)
const walletAddress = await aaProvider.getSigner().getAddress()

// send some eth to the wallet Address: wallet should have some balance to pay for its own creation, and for calling methods.

const myContract = new Contract(abi, aaProvider)

// this method will get called from the wallet address, through account-abstraction EntryPoint
await myContract.someMethod()
```
