# op-dispute-mon: Optimism Dispute Game Monitor

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethereum-optimism/op-dispute-mon)](https://goreportcard.com/report/github.com/ethereum-optimism/op-dispute-mon)
[![Latest Tag](https://img.shields.io/github/v/tag/ethereum-optimism/op-dispute-mon?label=Latest%20Version)](https://github.com/ethereum-optimism/op-dispute-mon/releases)

**op-dispute-mon** is a critical, off-chain service designed to **continuously monitor the state of Optimism's Dispute Games (Fault Proofs)**. Its primary function is to track active fault challenges, alert operators to potential issues, and ensure the health and security of the decentralized verification process.

---

## 🎯 Project Goal

In the Optimism ecosystem, **Dispute Games** (or Fault Proofs) are the mechanism used to cryptographically guarantee the validity of L2 state transitions and secure withdrawals.

This service acts as an **independent watchdog** that ensures:
1. All active dispute games are correctly progressing through their stages.
2. Operators are promptly alerted if a game requires an action (e.g., submitting a next claim) to avoid losing a challenge.
3. Provides visibility into the decentralized settlement layer of the Rollup.


---

## ⚡ Quickstart: Building and Running

### Prerequisites

Ensure you have **Go (1.20+)** and **Make** installed on your system.

### Build

Clone the repository and build the binary using the `make` command:

```shell
# Clone the repository
git clone [https://github.com/ethereum-optimism/op-dispute-mon.git](https://github.com/ethereum-optimism/op-dispute-mon.git)
cd op-dispute-mon

# Build the executable
make op-dispute-mon
