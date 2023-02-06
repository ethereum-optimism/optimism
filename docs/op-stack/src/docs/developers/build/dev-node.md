---
title: Running a local development environment
lang: en-US
---

## What is this?

A development environment is a local installation of the entire Optimism system.
Our default development environment includes both L1 and L2 development nodes.
Running the Optimism environment locally is a great way to test your code and see how your contracts will behave on Optimism before you graduate to a testnet deployment (and eventually a mainnet deployment).

Alternatively, you can get a hosted development node from [Alchemy](https://www.alchemy.com/optimism) or [any of these providers](../../useful-tools/providers.md).


## Do I need this?

We generally recommend using the local development environment if your application falls into one of the following categories:

1. **You're building contracts on both Optimism and Ethereum that need to interact with one another.** The local development environment is a great way to quickly test interactions between L1 and L2. The Optimism testnet and mainnet environments both have a communication delay between L1 and L2 that can make testing slow during the early stages of development.

2. **You're building an application that might be subject to one of the few [differences between Ethereum and Optimism](./differences.md).** Although Optimism is [EVM equivalent](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306), it's not exactly the same as Ethereum. If you're building an application that might be subject to one of these differences, you should use the local development environment to double check that everything is running as expected. You might otherwise have unexpected issues when you move to testnet. We strongly recommend reviewing these differences carefully to see if you might fall into this category.

However, not everyone will need to use the local development environment.
Optimism is [EVM equivalent](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306), which means that Optimism looks almost exactly like Ethereum under the hood.
If you don't fall into one of the above categories, you can probably get away with simply relying on existing testing tools like Truffle or Hardhat.
If you don't know whether or not you should be using the development environment, feel free to hop into the [Optimism discord](https://discord-gateway.optimism.io).
Someone nice will help you out!

## What does it include?

Everything you need to test your Optimistic application:

1. An L1 (Ethereum) node available at [http://localhost:9545](http://localhost:9545).
1. An L2 (Optimism) node available at [http://localhost:8545](http://localhost:8545).
1. All of the Optimism contracts and services that make L1 ⇔ L2 communication possible.
1. Accounts with lots of ETH on both L1 and L2.

## Prerequisites

You'll need to have the following installed:

1. [Docker](https://www.docker.com/). these directions were verified with version 20.10.17

To compile the software on your own you also need:

1. [Node.js](https://nodejs.org/en/), version 12 or later
1. [Classic Yarn](https://classic.yarnpkg.com/lang/en/)

## Setting up the environment

We use [Docker](https://www.docker.com) to run our development environment.

On a Linux system you can get the appropriate versions using these steps:


1. Install Docker. 
   If you prefer not to use the convenience script shown below, there are other installation methods.

   ```sh
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   ```

1. Configure Docker permissions.
   Note that these permissions do not take effect until you log in again, so you need to open a new command line window.

   ```sh
   sudo usermod -a -G docker `whoami`
   ```

::: tip There is no need to install docker-compose anymore
It is now available on Docker itself as `docker compose`
:::



## Getting the software

You can set up your development environment either by downloading the required software from [Docker Hub](https://hub.docker.com/u/ethereumoptimism) or by building the software from the [source code](https://github.com/ethereum-optimism/optimism).
Downloading images from Docker Hub is easier and more reliable and is the recommended solution.

### Downloading from Docker Hub

1. Clone and enter the [Optimism monorepo](https://github.com/ethereum-optimism/optimism):

   ```sh
   git clone https://github.com/ethereum-optimism/optimism.git
   cd optimism
   ```

2. Move into the `ops` directory:

   ```sh
   cd ops
   ```

3. Download the images:

   ```sh
   docker compose pull
   ``` 

4. Wait for the download to complete. This can take a while.

Depending on your machine, this startup process may take some time and it can be unclear when the system is fully ready.


## Accessing the environment

The local development environment consists of both an L1 node and an L2 node.
You can interact with these nodes at the following ports:

- L1 (Ethereum) node: [http://localhost:9545](http://localhost:9545)
- L2 (Optimism) node: [http://localhost:8545](http://localhost:8545)

## Getting ETH for transactions

::: warning WARNING
The private keys for the accounts used within the local development environment are **PUBLICLY KNOWN**.
Any funds sent to these accounts on a live network (Ethereum, Optimism, or any other public network) **WILL BE LOST**.
:::

All accounts that are funded by default within [Hardhat](https://hardhat.org) are funded with 5000 ETH on both L1 and L2.
These accounts are derived from the following mnemonic:

```
test test test test test test test test test test test junk
```

Here's the full list of accounts and their corresponding private keys:

```
Account #0: 0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
Private Key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

Account #1: 0x70997970c51812dc3a010c7d01b50e0d17dc79c8
Private Key: 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d

Account #2: 0x3c44cdddb6a900fa2b585dd299e03d12fa4293bc
Private Key: 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a

Account #3: 0x90f79bf6eb2c4f870365e785982e1f101e93b906
Private Key: 0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6

Account #4: 0x15d34aaf54267db7d7c367839aaf71a00a2c6a65
Private Key: 0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a

Account #5: 0x9965507d1a55bcc2695c58ba16fb37d819b0a4dc
Private Key: 0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba

Account #6: 0x976ea74026e726554db657fa54763abd0c3a0aa9
Private Key: 0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e

Account #7: 0x14dc79964da2c08b23698b3d3cc7ca32193d9955
Private Key: 0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356

Account #8: 0x23618e81e3f5cdf7f54c3d65f7fbc0abf5b21e8f
Private Key: 0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97

Account #9: 0xa0ee7a142d267c1f36714e4a8f75612f20a79720
Private Key: 0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6

Account #10: 0xbcd4042de499d14e55001ccbb24a551f3b954096
Private Key: 0xf214f2b2cd398c806f84e317254e0f0b801d0643303237d97a22a48e01628897

Account #11: 0x71be63f3384f5fb98995898a86b02fb2426c5788
Private Key: 0x701b615bbdfb9de65240bc28bd21bbc0d996645a3dd57e7b12bc2bdf6f192c82

Account #12: 0xfabb0ac9d68b0b445fb7357272ff202c5651694a
Private Key: 0xa267530f49f8280200edf313ee7af6b827f2a8bce2897751d06a843f644967b1

Account #13: 0x1cbd3b2770909d4e10f157cabc84c7264073c9ec
Private Key: 0x47c99abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd

Account #14: 0xdf3e18d64bc6a983f673ab319ccae4f1a57c7097
Private Key: 0xc526ee95bf44d8fc405a158bb884d9d1238d99f0612e9f33d006bb0789009aaa

Account #15: 0xcd3b766ccdd6ae721141f452c550ca635964ce71
Private Key: 0x8166f546bab6da521a8369cab06c5d2b9e46670292d85c875ee9ec20e84ffb61

Account #16: 0x2546bcd3c84621e976d8185a91a922ae77ecec30
Private Key: 0xea6c44ac03bff858b476bba40716402b03e41b8e97e276d1baec7c37d42484a0

Account #17: 0xbda5747bfd65f08deb54cb465eb87d40e51b197e
Private Key: 0x689af8efa8c651a91ad287602527f3af2fe9f6501a7ac4b061667b5a93e037fd

Account #18: 0xdd2fd4581271e230360230f9337d5c0430bf44c0
Private Key: 0xde9be858da4a475276426320d5e9262ecfc3ba460bfac56360bfa6c4c28b4ee0

Account #19: 0x8626f6940e2eb28930efb4cef49b2d1f2c9c1199
Private Key: 0xdf57089febbacf7ba0bc227dafbffa9fc08a93fdc68e1e42411a14efcf23656e
```

## Accessing logs

The logs produced by the L1 and L2 nodes can sometimes be useful for debugging particularly hairy bugs.
Logs will appear in the terminal you used to start the development environment, but they often scroll too quickly to be of much use.
If you'd like to look at the logs for a specific container, you'll first need to know the name of the container you want to inspect.

Run the following command to get the name of all running containers:

```sh
docker ps -a --format '{{.Image}}\t\t\t{{.Names}}' | grep ethereumoptimism
```

The output includes two columns.
The first is the name of the **image**, and the second the name of the **container** based on that image.
Each container is essentially an instance of a particular image and you're looking for the name of the *container* that you want to inspect.

| Image | Container |
| - | - |
| ethereumoptimism/l2geth:latest | ops_verifier_1
| ethereumoptimism/l2geth:latest | ops_replica_1
| ethereumoptimism/data-transport-layer:latest | ops_dtl_1
| ethereumoptimism/batch-submitter-service:latest | ops_batch_submitter_1
| ethereumoptimism/l2geth:latest | ops_l2geth_1
| ethereumoptimism/deployer:latest | ops_deployer_1
| ethereumoptimism/hardhat:latest | ops_l1_chain_1

You can then dump the logs for a given container as follows:

```sh
docker logs <container name>
```

For example, to see the logs produced by the L1 node:

```sh
docker logs ops_l1_chain_1
```

If you'd like to follow these logs as they're being generated, run:

```sh
docker logs --follow <name of container>
```

## Getting Optimism system contract addresses

If you want to [interact with Optimism system contracts](./system-contracts.md), you'll need to know the addresses of the contracts that are deployed on the network.

### Getting L2 contract addresses

L2 contracts are always deployed to the same addresses on every Optimism network.
You can simply look at [the L2 contract addresses for the mainnet Optimism network](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/deployments/mainnet#layer-2-contracts) and you'll have the addresses for your local environment.

### Getting L1 contract addresses

Optimism's L1 contracts are deployed to different addresses on different networks.
However, the addresses for your local environment will always be the same, even if you reset the environment.
You can get the addresses for your environment with the following command:

```sh
curl http://localhost:8080/addresses.json
```

You should get back a JSON object that contains a mapping of contract names to contract addresses.
These addresses should not change, even if you restart your environment.
