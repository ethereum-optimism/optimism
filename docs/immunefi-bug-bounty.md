# Optimism Bug Bounty Program (Immunefi)

- **Source:** <https://immunefi.com/bug-bounty/optimism/information/> (Information tab)
- **Captured:** 30 October 2026. The page shows **Last Updated: 13 October 2026**.

Amounts, percentages, dates, labels and eligibility conditions are recorded as the page
displays them. Explanatory prose is condensed. This document covers the Information tab only.
The asset list and the impact list are on the
[Scope tab](https://immunefi.com/bug-bounty/optimism/scope/#top).

## Program parameters

| Parameter | Value |
|---|---|
| Maximum bounty | $2,000,042 |
| Reward payment | USDC on Ethereum, denominated in USD |
| Live since | 14 January 2022 |
| Last updated | 13 October 2026 |
| Triage | Triaged by Immunefi |
| Proof of concept | Required for all severities |
| KYC | Required for payout processing |
| Submission fee | 50 USDC per report ("Paid Submissions" badge) |
| Responsible publication | Category 3: Approval Required |
| Program standard | "Immunefi Standard" badge: the project meets Immunefi's current Terms & Conditions |
| Total paid | 2.8M |
| Median resolution time | 2 days |
| Total assets in scope | 36 |
| Tags | ETH, Optimism, Defi, NFT, L1, L2, C/C++, Go, Solidity, Rust |
| Submit a bug | <https://bugs.immunefi.com/dashboard/new-submission> |

The "Immunefi Standard" and "Paid Submissions" badges are icons next to the project name. Their
text shows on hover.

## Rewards by threat level

### Blockchain/DLT

| Severity | Min | Max | Rule |
|---|---|---|---|
| Critical | — | Up to $2,000,042 | Primacy of Impact |
| High | $15,000 | $50,000 | Primacy of Impact |
| Medium | $1,000 | $15,000 | Primacy of Impact |

**Critical reward calculation:**

- The reward is 10% of the funds directly affected.
- The reward is capped at the maximum critical reward of $2,000,042.
- The reward depends on the ratio of the funds at risk to the market cap.
  - The funds at risk include all affected projects on top of the blockchain/DLT.
  - The market cap is the average of CoinMarketCap.com and CoinGecko.com.
  - Both values are calculated at the time the bug report is submitted.

### Smart Contract

| Severity | Min | Max | Rule |
|---|---|---|---|
| Critical | — | Up to $2,000,042 | Primacy of Impact |
| High | $15,000 | $50,000 | Primacy of Impact |
| Medium | $1,000 | $15,000 | Primacy of Impact |

**Critical reward calculation (mainnet assets):** the reward is 10% of the funds directly
affected, up to a maximum of $2,000,042.

### Websites and Applications

| Severity | Min | Max | Rule |
|---|---|---|---|
| Critical | $5,000 | $50,000 | Primacy of Rules |
| High | $500 | $5,000 | Primacy of Rules |
| Medium | $50 | $500 | Primacy of Rules |

### Additional reward terms

- **Proof of concept:** a PoC that demonstrates the bug's impact is required. The PoC must comply
  with the
  [Immunefi PoC Guidelines and Rules](https://immunefisupport.zendesk.com/hc/en-us/articles/9946217628561-Proof-of-Concept-PoC-Guidelines-and-Rules).
- **KYC:** OptimismPBC requests a W-9 from US residents and a W-8 from residents outside the US.
- **Critical cap:** critical vulnerabilities are further capped at 10% of economic damage. The main
  consideration is the funds affected. PR and brand considerations also apply, at the team's
  discretion.
- **Web application reports:** the sensitivity of the data involved sets the position within the
  reward range. Disclosure of personal information or of confidential customer organization data
  is rewarded towards the top of the range.
- **Cross-domain testing:** for exploits that involve cross-domain transactions, the program
  recommends the
  [dockerized services](https://github.com/ethereum-optimism/optimism/blob/6f8e432506a5f4ba094f091b22a0bb6acc53fdac/ops/README.md)
  and modified
  [integration tests](https://github.com/ethereum-optimism/optimism/blob/6f8e432506a5f4ba094f091b22a0bb6acc53fdac/integration-tests/test/bridged-tokens.spec.ts).
  Both links point to a pinned commit.

### Governance proposals

In addition to the listed assets, the calldata and code of the latest **Protocol Upgrade
Proposals** are in scope.

- Not in scope: Governor Upgrade proposals and proposal previews.
- The proposal must be an official governance proposal. A proposal is official when at least one of
  these conditions is true:
  1. The proposal has moved to an onchain vote on vote.optimism.io.
  2. Someone from OP Labs or the Optimism Foundation posted the proposal.
  3. Someone from OP Labs or the Optimism Foundation commented that the proposal is eligible for
     the bounty.
- The latest in-flight proposal is at
  <https://gov.optimism.io/c/88-category/technical-proposals/47>.

## Program focus

The program is focused on preventing:

- Theft of assets held in Optimism's smart contracts.
- Theft, freezing or other loss of funds due to vulnerabilities in the smart contracts or in the
  critical blockchain client services (op-node and op-geth).

The Program Overview also describes the OP Stack and its first release, Bedrock, and links to
<https://www.optimism.io/> and <https://stack.optimism.io/>.

## Primacy of Impact vs Primacy of Rules

**Primacy of Impact** applies to these impacts:

- Smart Contract — Critical, High, Medium
- Blockchain/DLT — Critical, High, Medium

Under Primacy of Impact, the impact takes priority over the specific asset. A bug with an in-scope
impact is eligible even if the affected asset is not in scope. See
[Best Practices: Primacy of Impact](https://immunefisupport.zendesk.com/hc/en-us/articles/12340245635089-Best-Practices-Primacy-of-Impact).

- When submitting such a report, select the **Primacy of Impact asset placeholder**.
- Other Immunefi programs of the same team are not covered by this program's Primacy of Impact.
- Testnet files and mock files are not covered by Primacy of Impact.

**Primacy of Rules** applies to all other impacts (on this page, Websites and Applications). These
impacts are bound by the terms and conditions of this program.

## Known issues

These issues are known and are not eligible for a reward.

| # | Category | Issue | Link | Last updated |
|---|---|---|---|---|
| 1 | Blockchain/DLT | An attacker who can predict an L1 reorg can reorder dispute game transactions so that the honest challenger responds incorrectly and loses its bond. The dispute game still resolves correctly. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 2 | Blockchain/DLT | op-challenger bugs that result in an "Incorrectly resolved dispute game, detected by op-dispute-mon". These are known because op-dispute-mon can already detect and resolve them. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 3 | Blockchain/DLT | All currently known devp2p issues in the upstream list. | [devp2p RLPx known issues](https://github.com/ethereum/devp2p/blob/master/rlpx.md#known-issues-in-the-current-version) | 21 June 2026 |
| 4 | Smart Contract | Bridge "foot guns" from token misconfiguration that the bridge intentionally does not prevent. Examples: both (or neither) of the local and remote tokens are OptimismMintable; tokens that change balances dynamically, such as fee-on-transfer and rebasing tokens. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 5 | Smart Contract | An attacker can grief deposit transactions, at a cost, by filling up `MAX_RESOURCE_LIMIT`. PR 5064 mitigates this by increasing the cost of a sustained attack, but does not fully resolve it. A complete fix requires architectural changes. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 6 | Smart Contract | Cross-chain messages with very large amounts of data, or with very specific amounts of gas, can enable griefing attacks. The sender's funds become stuck and need an upgrade to release them. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 7 | Smart Contract | ETH that a contract deposits to the OptimismPortal can be stranded. A failed deposit transaction still credits the mint value to the sender. If the L1 sender is a contract, the L2 `tx.origin` is aliased, and the minted ETH goes to the aliased address, which generally cannot recover it. This is documented; users are encouraged to use the CrossDomainMessenger contracts. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 8 | Blockchain/DLT | A bug in `ResolvedDelegateProxy.sol` can cause a storage-slot key collision that overwrites the implementation address. The bug depends on the storage layout of the implementation contract. Optimism is not affected. | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 9 | Smart Contract | Fake ERC-20 withdrawal via L2StandardBridge. No check in L2StandardBridge stops an attacker from withdrawing a fake ERC-20 on L2 in exchange for a real ERC-20 (for example WBTC) on L1. A check in L1StandardBridge currently blocks the withdrawal from finalizing. **A way around the L1 check is eligible for a reward.** | [Scope: out of scope](https://immunefi.com/bug-bounty/optimism/scope/#top:~:text=The%20following%20are%20considered%20out%2Dof%2Dscope%20for%20the%20bug%20bounty%3A) | 21 June 2026 |
| 10 | Blockchain/DLT | Missing zlib error/header validation in kona's channel decompressor leads to cross-client fault-proof divergence. | [Immunefi report 84201](https://bugs.immunefi.com/magnus/855/projects/355/reports/84201) | 9 September 2026 |

## Prohibited activities

The page lists Immunefi's default prohibited activities:

- Any testing on mainnet or public testnet deployed code. Do all testing on local forks of a public
  testnet or of mainnet.
- Any testing with pricing oracles or third-party smart contracts.
- Phishing or other social engineering attacks against employees or customers.
- Any testing with third-party systems and applications (for example browser extensions) or
  third-party websites (for example SSO providers, advertising networks).
- Denial of service attacks against project assets.
- Automated testing of services that generates significant amounts of traffic.
- Public disclosure of an unpatched vulnerability in an embargoed bounty.
- Any other action prohibited by the [Immunefi Rules](https://immunefi.com/rules/).

## Feasibility limitations

The program uses Immunefi's default feasibility limitation standards. These standards define what
researchers and projects can cite when a valid, in-scope attack has real-world obstacles to
execution. They also say that mitigations which are not feasible, or which need unconventional
action, are not reasons to downgrade a bug's severity. The page links these standards:

- [Chain Rollbacks](https://immunefisupport.zendesk.com/hc/en-us/articles/16913153448721-Chain-Rollbacks)
- [Pre-Impact Bug Monitoring](https://immunefisupport.zendesk.com/hc/en-us/articles/19430444320401-Pre-Impact-Bug-Monitoring)
- [Attack Investment Amount](https://immunefisupport.zendesk.com/hc/en-us/articles/17243068885265-Attack-Investment-Amount)
- [Attacks With A Financial Risk To The Attacker](https://immunefisupport.zendesk.com/hc/en-us/articles/17454897136401-Attacks-With-A-Financial-Risk-To-The-Attacker)
- [When Is An Impactful Attack Downgraded To Griefing?](https://immunefisupport.zendesk.com/hc/en-us/articles/17455102268305-When-Is-An-Impactful-Attack-Downgraded-To-Griefing)
