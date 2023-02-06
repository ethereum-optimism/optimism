# The Optimism Community Hub

[![Discord](https://img.shields.io/discord/667044843901681675.svg?color=768AD4&label=discord&logo=https%3A%2F%2Fdiscordapp.com%2Fassets%2F8c9701b98ad4372b58f13fd9f65f966e.svg)](https://discord-gateway.optimism.io)
[![Twitter Follow](https://img.shields.io/twitter/follow/optimismPBC.svg?label=optimismPBC&style=social)](https://twitter.com/optimismPBC)

Optimism is a Layer 2 platform for Ethereum.

Optimism is, in a nutshell, an application inside of Ethereum that executes transactions more efficiently than Ethereum itself. It's based on the concept of the [Optimistic Rollup](https://research.paradigm.xyz/rollups), a construction that allows us to "optimistically" publish transaction results without actually executing those transactions on Ethereum (most of the time). Optimism makes transactions cheaper, faster, and smarter.

Please note that this repository is undergoing rapid development.

------

This is the source for the [community hub](https://community.optimism.io/).

# Usage
## Serve Locally
```shell
yarn dev
```

Then navigate to http://localhost:8080.
If that link doesn't work, double check the output of `yarn dev`. 
You might already be serving something on port 8080 and the site may be on port 8081.

## Build for Production
```shell
yarn build
```

You probably don't need to run this command, but now you know.
