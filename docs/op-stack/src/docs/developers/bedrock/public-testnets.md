---
title: Public Testnets
lang: en-US
---

## Goerli

The Optimism Goerli testnet was migrated to Bedrock on January 12, 2023.

<table width="100%">
    <tbody>
        <tr>
            <td colspan="2"><strong>Chain Parameters</strong></td>
        </tr>
        <tr>
            <td>L1 Chain</td>
            <td>Goerli</td>
        </tr>
        <tr>
            <td>L1 Chain ID</td>
            <td>5</td>
        </tr>
        <tr>
            <td>L2 Chain ID</td>
            <td><code>420</code></td>
        </tr>
        <tr>
            <td>Rollup Config</td>
            <td>This network does not require a rollup config. Specify <code>--network=goerli</code> on the command line
                when official images are released.
            </td>
        </tr>
        <tr>
            <td>Bedrock Data Directory</td>
            <td><a href="https://storage.googleapis.com/oplabs-goerli-data/goerli-bedrock.tar">https://storage.googleapis.com/oplabs-goerli-data/goerli-bedrock.tar</a>
            </td>
        </tr>
        <tr>
            <td>Legacy Geth Data Directory</td>
            <td><a href="https://storage.googleapis.com/oplabs-goerli-data/goerli-legacy-archival.tar">https://storage.googleapis.com/oplabs-goerli-data/goerli-legacy-archival.tar</a>
            </td>
        </tr>
        <tr>
            <td>Block Explorer</td>
            <td>
                <a href="https://goerli-optimism.etherscan.io">https://goerli-optimism.etherscan.io</a>
            </td>
        </tr>
        <tr>
            <td>Public RPC Endpoint</td>
            <td>
                <a href="https://goerli.optimism.io">https://goerli.optimism.io</a>
            </td>
        </tr>
        <tr>
            <td>Sequencer Endpoint</td>
            <td>
                <a href="https://goerli-sequencer.optimism.io">https://goerli-sequencer.optimism.io</a>
            </td>
        </tr>
        <tr>
            <td>Withdrawal Period</td>
            <td>2 Seconds</td>
        </tr>
        <tr>
            <td colspan="2"><strong>Software Images</strong></td>
        </tr>
        <tr>
            <td>op-node</td>
            <td><code>us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:v0.10.9</code></td>
        </tr>
        <tr>
            <td>op-geth</td>
            <td><code>ethereumoptimism/op-geth:v1.10.26-166f27c</code></td>
        </tr>
        <tr>
            <td>Legacy Geth</td>
            <td><code>ethereumoptimism/l2geth:0.5.29</code></td>
        </tr>
        <tr>
            <td colspan="2"><strong>Contract Addresses</strong></td>
        </tr>
        <tr>
            <td><a href="https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60">SystemConfigProxy</a>
            </td>
            <td><code>0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60</code></td>
        </tr>
        <tr>
            <td><a href="https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0">L2OutputOracleProxy</a>
            </td>
            <td><code>0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0</code></td>
        </tr>
        <tr>
            <td><a href="https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383">OptimismPortalProxy</a>
            </td>
            <td><code>0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383</code></td>
        </tr>
        <tr>
            <td><a href="https://goerli.etherscan.io/address/0x883dcF8B05364083D849D8bD226bC8Cb4c42F9C5">OptimismMintableERC20FactoryProxy</a>
            </td>
            <td><code>0x883dcF8B05364083D849D8bD226bC8Cb4c42F9C5</code></td>
        </tr>
        <tr>
            <td><a href="https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971">SystemDictatorProxy</a>
            </td>
            <td><code>0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971</code></td>
        </tr>
        <tr>
            <td>
                <a href="https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111">Lib_AddressManager</a>
            </td>
            <td>
                <code>0xa6f73589243a6A7a9023b1Fa0651b1d89c177111</code>
            </td>
        </tr>
        <tr>
            <td>
                <a href="https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294">Proxy__OVM_L1CrossDomainMessenger</a>
            </td>
            <td>
                <code>0x5086d1eEF304eb5284A0f6720f79403b4e9bE294</code>
            </td>
        </tr>
        <tr>
            <td>
                <a href="https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8">Proxy__OVM_L1StandardBridge</a>
            </td>
            <td>
                <code>0x636Af16bf2f682dD3109e60102b8E1A089FedAa8</code>
            </td>
        </tr>
    </tbody>
</table>