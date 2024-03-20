# Engineering Update #11

March 9, 2022

Greetings from your engineering team. Last week saw the deployment of the final version of Turing (on Rinkeby), a regenesis of Rinkeby, and the deployment of Turing on Mainnet. This is a major milestone, since it marks our first big step towards the Boba we envision. As you know, Boba is not just an L2 scaling solution, but a hybrid computer which allows a distributed computer, namely Ethereum, to interact seamlessly with all non-distributed computers (the Web 2 world). As part of this milestone, the online presence of Boba network will become more differentiated and more focused on the needs of Web2 and Metaverse developers looking to build compelling Web3 experiences. 

## 1. Boba Development will move to github.com/bobanetwork; Hackathons 

As of next week, the main Boba repo and associated repos will migrate to `github.com/bobanetwork` to provide a less confusing developer experience. The timing of this transition is in part due to *three* upcoming hackathons focusing on distributed compute and Web2/Web3 interoperability. As we engage hundreds of recent graduates and developers, we want them to have an easy and straightforward onboarding experience and clear documentation. Relatedly, as we prepare for the hackathons, please contribute to our ideas list, which will form the basis of the various topic areas and challenges. The document is [here](https://github.com/bobanetwork/hackathons/blob/main/README.md) - just open an issue or add your ideas. 

## 2. Mainnet Turing Monster mint next week

We have been refining the Turing Monster NFT example into production quality NFT with integration with the [ShibuDAO NFT marketplace](https://shibuinft.com), so that when we mint, all the required infrastructure will be in place. The Turing Monster mint will be the first to use random numbers from Geth and the first L2->L1 bridgeable NFT with on-chain data. Make sure to get your part of history - and let's see how the sequencer performs under heavy load!

## 3. Developer Documentation Overhaul

By popular demand, we have created a better [developer onboarding document](https://github.com/bobanetwork/boba/blob/develop/boba_documentation/Developer_Start.md). The Turing documentation has been [updated as well](https://github.com/bobanetwork/boba/blob/develop/packages/boba/turing/README.md).

## 4. EIP-4844

Please help out Ethereum and all L2s by putting your support behind [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844). Ethereum is still too expensive for normal people; EIP-4844 will make all rollups such as @Optimism, @Arbitrum, and @Boba cheaper while maintaining current functionality and security. EIP-4844 introduces `shard blob transactions` which are perfect for depositing L2 transaction inputs into L1, which is essential for transparency, security, and fraud-proving. EIP-4844 is also a great way to start building, testing, and using key features of the future Ethereum with data sharding.

## 5. Etherscan for Boba Development Start

Etherscan is now working on Bobascan for both Mainnet and Rinkeby. This means that soon, Boba will have a powerful, fully featured block explorer that virtually everyone is familiar with.  

## 6. Gateway updates per MitchellUnknown

We are trying our best to keep up with MitchellUnknown, who has provided >75 suggestions for improving the gateway and the main website - we are working through all those one by one. For example, today, we got through 8 MUIs (Mitchell Unknown Issues) and pushed those improvements out to the Mainnet gateway.