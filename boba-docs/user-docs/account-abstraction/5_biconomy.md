# Biconomy Integration with Boba

In this tutorial, we'll walk through the process of integrating Biconomy with Boba chain to enable gasless transactions.

The tutorial covers the entire process from setting up the Biconomy paymaster to actually sending a gasless transaction on the Boba chain.

### Prerequisites

- A wallet with some testnet tokens for Boba chain

### Step 1: Create a Biconomy Paymaster

1. Go to the [Biconomy Dashboard](https://dashboard.biconomy.io/).
2. Sign up or log in to your account.
3. Click on "Create New Paymaster."
4. Select "Boba Testnet" as the network.
5. Choose a name for your paymaster and click "Create."
6. Once created, you'll see your Paymaster API Key. Save this for later use.

### Step 2: Create a Smart Account

1. Install the necessary dependencies:

```bash
npm install @biconomy/account viem
```

2. Set up your account and biconomy configuration:

```typescript
import { createWalletClient, http, parseEther } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { bobaSepolia } from "viem/chains";
import { createSmartAccountClient, PaymasterMode } from "@biconomy/account";

const config = {
    privateKey: "your-private-key",
    biconomyPaymasterApiKey: "your-paymaster-api-key",
    bundlerUrl: "https://bundler.biconomy.io/api/v2/28882/nJPK7B3ru.dd7f7861-190d-41bd-af80-6877f74b8f44",
};
```

3. Create a Biconomy Smart Account:

```typescript
const account = privateKeyToAccount(`0x${config.privateKey}`);
const client = createWalletClient({
    account,
    chain: bobaSepolia,
    transport: http(),
});

const smartWallet = await createSmartAccountClient({
    signer: client,
    biconomyPaymasterApiKey: config.biconomyPaymasterApiKey,
    bundlerUrl: config.bundlerUrl,
});
const saAddress = await smartWallet.getAccountAddress();
console.log("Smart Account Address", saAddress);
```

### Step 3: Send a Gasless Transaction

1. Prepare the transaction data:

```typescript
const tx = {
    to: '0xf5715961C550FC497832063a98eA34673ad7C816',
    value: parseEther('0.0001')
};
console.log("Transaction sent:", tx);
```
2. Send the transaction and wait for the result:

```typescript
const userOpResponse = await smartWallet.sendTransaction(tx, {
    paymasterServiceData: { mode: PaymasterMode.SPONSORED },
});
const { transactionHash } = await userOpResponse.waitForTxHash();
console.log("Transaction Hash", transactionHash);
const userOpReceipt = await userOpResponse.wait();
if (userOpReceipt.success === "true") {
    console.log("UserOp receipt", userOpReceipt);
    console.log("Transaction receipt", userOpReceipt.receipt);
}
```

### Conclusion

You have now successfully integrated Biconomy with Boba chain and sent a gasless transaction. This setup allows your users to interact with your dApp without worrying about gas fees, improving the overall user experience.

Remember to handle errors appropriately and consider implementing additional features like transaction status updates or retry mechanisms for failed transactions.

For more advanced use cases and detailed documentation, refer to the [Biconomy documentation](https://docs.biconomy.io/).