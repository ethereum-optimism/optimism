# libp2ptool

A small utility that reads public/private keys from STDIN, and returns their corresponding libp2p peer ID. Used in our automation scripts to create pre-populated groups of peers.

## Installation

Run `make`. To install globally, run `make install`.

## Usage

For a private key:

```bash
cat key.txt | libp2ptool -private-key
```

For a public key:

```bash
cat key.txt | libp2ptool
```

Spaces are stripped from the input before parsing. Any leading `0x` is stripped as well.