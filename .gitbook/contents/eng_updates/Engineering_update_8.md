# Engineering Update #8

- [Engineering update #8](#engineering-update--8)
  * [1. Roadmap and Whitepaper](#1-roadmap-and-whitepaper)
  * [2. Blockchain Trends - the L2 space and where we fit](#2-blockchain-trends---the-l2-space-and-where-we-fit)
  * [3. Security, Security, Security](#3-security--security--security)
  * [4. Turing Example Apps](#4-turing-example-apps)
  * [5. Major Gateway Revision](#5-major-gateway-revision)

February 6, 2022

Greetings from your engineering team. As our team expands rapidly (3 more engineers just last week), we have also increasing bandwidth for communications, updates, hackathons (first one starting mid-March - only 5 weeks from today), and developer support. Among other changes, these engineering updates will now be published weekly. This specific update will be more strategic (#1 and #2) but will also briefly cover day-to-day engineering activities (#3-#5). 

## 1. Roadmap and Whitepaper

A graphical roadmap for 2022 will be released this week. This will be a 'living' document, subject to major change as needed, but will give our community a general overview of some of the major deadlines and new releases covering **Boba3 (Prelude)** and **Boba4 (Concert)**. Generally, we will operate in an overlapped tick-tock cycle, so for example when **Boba3/Prelude** goes Mainnet (June 1), **Boba4/Concert** will go Rinkeby, and so forth, so that teams can start testing early. **Boba3/Prelude** _will not_ involve a regenesis but **Boba4/Concert** _will_ require a regenesis due to very substantial changes under the hood. We are also writing a technical whitepaper motivating some of the major architectural decisions made for **Boba3** and **Boba4** - the main point there will be systems for coordinating multiple sequencers and for doing so in ways that are compatible with Turing. Expect the whitepaper in April. 

## 2. Blockchain Trends - the L2 space and where we fit

It is currently fashionable to talk about 'multichains' and there is much emphasis on bridging. We have a contrarian perspective. In our view, alt-L1s and to some extent L2s only exist because Ethereum has been slow to decisively upgrade the basic infrastructure, such as the long-promised change of PoW to PoS. Since Ethereum became congested and very expensive, users naturally sought alternatives, allowing alt-L1s and L2s to get traction. Overall, this has degraded the user experience and capital, technology, users, and traffic are now thinly scattered across 10 to 20 chains. However, Ethereum's Beacon Chain shipped on December 1, 2021 and we gauge it highly likely that Ethereum will provide a much improved user experience by early 2023. In that instant, existing users and new users: (**1**) will have little need for alt-L1s (since Ethereum will "just work") and (**2**) when Ethereum gets cheaper and faster, the Ethereum-based L2s will get even faster, even cheaper, and even more attractive. 

The combination of a highly performant L1, surrounded by a cloud of highly-performant L2s, will put alt-L1s at a disadvantage as soon as this winter. With respect to the L2s, Boba is not only about cost and speed - although we certainly appreciate when users says that Boba is fast and reasonably priced - but about allowing developers to build unique services that are currently impossible to build in any other way. Specifically, we are referring to **Turing**, which gives developers the ability to _atomically_ interact with any internet-connected computer on earth. So the world we foresee, and are building towards, is a world where (*1*) Ethereum is at the center and providing critical timing data, process synchronization, and security, and (*2*) Ethereum is surrounded by a cloud of L2s with different characteristics - speed, security, privacy, low cost, or, in our case, hybrid compute. In aggregate, this system will provide a definitive blockchain ecosystem and user experience.

In this way, blockchains will recapitulate the development of 'normal' computers. The first Apple, the Apple 1, launched on April 1 1976 with one CPU - the MOS 6502, a simplified and less expensive version of the Motorola 6800. Today, 40+ years later, when you play a game on your iPhone, several hundred chips (some of them in your hand and most of them in the cloud) work together to give you a great user experience. Blockchains are recapitulating that development arc, in which a central processor is gradually enabled to work together closely with many other specialized processors to provide amazing services to end users. The role of Boba is to serve as the pipe between the blockchain world to every other (networked) computer, router, and device on earth. Need to connect your solidity smart contract to your bank API, your smartwatch API, your router, a supercomputer, or the Twitter API? That's what we are building at Boba and that's why we are an Ethereum L2. As noted above, these themes are being formalized into a roadmap and a whitepaper. The TLDR is that in 2021/22 the blockchain community has been thinking about multichain and bridging, whereas in 22/23 the focus will be on _compute_ - after all, Ethereum is a distributed computer and attention to its aggregate compute performance is overdue. 

## 3. Security, Security, Security

The main focus of the last week, and the coming week, are largely invisible changes to the production AWS deployment focusing on security. You will notice occasional 'tech update' windows as new security, threat detection, and other systems are added to our current production stack. As part of that security focus, many of the packages and services are being upgraded and updated.      

## 4. Turing Example Apps

To help developers (and the participants of the March hackathon) we are also coding a variety of simple games and tech demonstrators, all using Turing. Many of those will be launched on Mainnet for you to test (and to help you build the apps and services that you think are missing on Boba).

## 5. Major Gateway Revision

Work continues on a near-complete re-write of the gateway emphasizing simplicity, ease of use, and speed. In parallel, we are looking closely at other ways of reducing friction especially for new users. For example a **Boba faucet** of some kind would almost certainly make life easier for new-comers, in addition to the multi-bridge capability that was launched last week and which can reduce bridging fees as much as 75% depending on how many tokens are being bridged simultaneously. 

