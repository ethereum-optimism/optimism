const { ethers } = require("ethers");


async function test(){
// Define the private key and the network
const privateKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";

// Initialize the provider and the signer
const provider = new ethers.providers.JsonRpcProvider("http://localhost:8545");
const signer = new ethers.Wallet(privateKey, provider);

// Define the amount to send (in ether) and the number of transactions to send
const amount = ethers.utils.parseEther("0.001"); // 0.001 ETH
const numTransactions = 10;

// Get the balance of the account
signer.getBalance().then((balance) => {
  // Convert the balance to ether and log it
  const balanceInEth = ethers.utils.formatEther(balance);
  console.log(`Account balance: ${balanceInEth} ETH`);
}).catch((err) => {
  console.error(`Error getting account balance: ${err.message}`);
});

// Send the transactions
for (let i = 0; i < numTransactions; i++) {
  // Generate a random address to send to
  const randomAddress = ethers.Wallet.createRandom().address;

  // Create the transaction
  const transaction = {
    to: randomAddress,
    value: amount
  };

  // Sign and send the transaction
  await signer.sendTransaction(transaction)
    .then((tx) => {
      console.log(`Transaction sent: ${tx.hash}`);
    })
    .catch((err) => {
      console.error(`Error sending transaction: ${err.message}`);
    });
}
}
await test()
