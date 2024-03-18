---
description: Learn more about manually relaying messages from L2 to L1
---

# Custom Relayer

The boba relayer provides a simple interface for relaying messages from L2 to L1. The relayer is a simple node.js application that can be run locally. The relayer is designed to be used with the [Boba Bridge](../boba-basics/bridge-basics/standard-bridge.md) and [Boba Fast Bridge](../boba-basics/bridge-basics/fast-bridge.md) but can be used with any L2 contract that sends a message to L1.

The boba relayer can be found in the [boba repo](https://github.com/bobanetwork/boba/tree/develop/boba\_community/boba-relayer).

To use the relayer you will need to have a node.js environment setup. You can find instructions on how to do that [here](https://nodejs.org/en/download/).

Once you have node.js installed you can clone the boba repo and install the relayer dependencies.

```bash
git clone
cd boba_community/boba-relayer
npm install
```

The relayer requires a few environment variables to be set. You can find a list of the required environment variables in the [.env.example](https://github.com/bobanetwork/boba/tree/develop/boba\_community/boba-relayer/.env.example)

```bash
cp .env.example .env
vim .env
```

Once you have the environment variables set you can start the relayer.

```bash
npm start
```
