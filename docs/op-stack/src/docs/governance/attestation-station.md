---
title: What is the AttestationStation?
lang: en-US
---
![](../../assets/docs/governance/attestationstation/attestationstation.png)

The AttestationStation is an **attestation smart contract** deployed on Optimism.  

The goal of the AttestationStation is to provide a permissionless and accessible data source for builders creating reputation-based applications. By enabling anyone to make arbitrary attestations about other addresses, we can create a rich library of qualitative and quantitative data that can be used across the ecosystem.


<!-- TODO: Add source code link when we have an authoritative source -->

## General FAQ

#### What are attestations?

Attestations are statements by a creator (who attested this) about a subject (who is being attested about). Attestations could present any qualitative or quantitative statement. To paint a picture — actors might submit attestations that are contextual to their brand, ecosystem, and governance structure.

![](../../assets/docs/governance/attestationstation/attestations.png)



#### What can attestations be used for?

We imagine the first use case for attestations is to create sybil resistant identity that can power [non-plutocratic governance](https://vitalik.ca/general/2021/08/16/voting3.html).

Longer term, this open-source primitive can be used for a variety of sybil-resistant applications including on-chain credit scoring / under collateralized loans.

#### How can you go from attestations to sybil-resistant identity?

Attestations in the AttestationStation are on-chain and can be used by other smart contracts in a variety of applications. Instead of having a single entity owning user data and identity, the AttestationStation is a graph of peer-to-peer (p2p) attestations. 

The first step to get from attestations to sybil-resistant identity is to grow the number of attestations in the AttestationStation. To do that, we are taking a two pronged approach by growing the number of:

* **Trusted attestations**: These attestations are made by organizations like Gitcoin, DegenScore, Otterspace, etc. attest about individual community members.
* **Social attestations**: These are attestations from one address about another. Eg zain.eth says kathy.eth is a colleague, kathy.eth says will.eth is a friend, etc.

![](../../assets/docs/governance/attestationstation/network.png)

Anyone can then take the graph of p2p attestations from the AttestationStation and run computations like EigenTrust over the set of data to derive identity sets on top of a purely subjective web of trust.

![](../../assets/docs/governance/attestationstation/eigan.png)

To build a robust, trustworthy identity network, these computations will be run iteratively. We can start with a purely subjective web of trust, and use that starting point to derive a larger web of trust, and so on — we can begin to establish a credibly neutral reputation that is entirely peer-to-peer. 

#### How is the AttestationStation different from other attestation products?

The AttestationStation is deliberately dead simple and serves as an invite to ecosystem contributors to come build an open-source and permissionless attestation graph together.

Creating this system in a decentralized and open-source manner is important because it allows for greater inclusion and representation of different perspectives. This can help to ensure that the system is fair and accessible to all, and that it accurately reflects the diversity of the communities it serves.

#### How do I use the AttestationStation?

See [the tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/ecosystem/attestation-station).

#### What are the contract addresses for the AttestationStation?

| Network | Address |
| - | - |
| Optimism Goerli | [`0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77`](https://goerli-explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)  |
| Optimism Mainnet | [`0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77`](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77) |

#### What products are built on the AttestationStation? 
If your product is using the AttestationStation, make a PR including how you're using attestations to be added to the list 😊
* [AttestationStation Interface by sbvegan](https://attestationstation.xyz/)
* [Optimist Score by Flipside](https://science.flipsidecrypto.xyz/optimist/)
* [Optimism Attestor by Clique](https://provenance.clique.social/attestor/opattestor)

#### I am building on the AttestationStation but have some questions, where can I discuss these?

The best place to ask any dev related questions is the #dev-support channel on [the Optimism Discord](https://discord-gateway.optimism.io/). If you need additional support check out this [Help Article](https://help.optimism.io/hc/en-us/articles/9762044018843-How-do-I-get-project-support-marketing-integrations-etc-).

#### I want to apply for a grant to build on the AttestationStation, how can I do this?

You can learn more about the variety of grants program available at Optimism [here](allocations/#ecosystem-fund). As a reminder, your work should be published to a public GitHub repo.

#### What are some things I should build with the AttestationStation?

It will take a huge community effort to realize the potential that reputation has to transform web3. That’s why we started small with the AttestationStation and an open invite to come experiment with us. We can already think of a bunch of fun projects to build today like:

* **EiganTrust**: Aggregate attestations from various communities and use techniques like [EigenTrust](https://en.wikipedia.org/wiki/EigenTrust) to derive reputation
* **SybilRank**: Create a [SybilRank](https://users.cs.duke.edu/~qiangcao/sybilrank_project/index.html) calculator! (h/t Barry Whitehat for the suggestion)
* **Data visualization**: Create data visualizations representing the different types of attestations in the AttestationStation
* **Predictive attestations**: Instead of attesting “I trust XYZ”, try fun attestations like, “I believe XYZ will be considered trusted by a majority of node in the future”. Plus, what if we add a slashing condition to the predictive attestation?
* **Attestation delegation**: Build a system which manages attestations automatically for users. This system should enable users to delegate some of their attestation assignment to a third party. For instance, users may opt-in to delegating their trust scores to a sybil detection court system. Another project is to build that sybil detection court system! 
* **Attestation import**: Write proxy contracts which import attestations of various formats into the standardized AttestationStation format so that they can be consumed by the standard AttestationStation tooling.
* **Viral attestations**: Create systems which make it fun and easy for users to attest useful information about each other.
* **Composable NFT allowlists**: Create a way for creators to easily build, manage, and share mint allowlists for upcoming NFT drops!


## Technical specifications

The following is the breakdown of Optimism's AttestationStation smart contract.

### State

#### attestations

The following is the nested mapping that stores all the attestations made.

```
mapping(address => mapping(address => mapping(bytes32 => bytes))) public attestations;
```

The following is a struct that represents a properly formatted attestation.

#### AttestationData

```
struct AttestationData {
    address about;
    bytes32 key;
    bytes val;
}
```

### Events

#### AttestationCreated

This event is emitted when an attestation is successfully made.

```
event AttestationCreated(
    address indexed creator,
    address indexed about,
    bytes32 indexed key,
    bytes val
);
```

### Functions

#### attest

```
function attest(AttestationData[] memory _attestations) public
```

Records attestations to the AttestationStation's state and emits an `AttestationCreated` event with the address of the message sender, address the attestation is about, the bytes32 key, and bytes value.

Parameters:

| Name           | Type              | Description                         |
| -------------- | ----------------- | ----------------------------------- |
| \_attestations | AttestationData[] | Array of `AttestationData` structs. |

