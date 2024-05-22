# go-ds-leveldb

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/ipfs/go-ds-leveldb?status.svg)](https://godoc.org/github.com/ipfs/go-ds-leveldb)
[![Build Status](https://travis-ci.org/ipfs/go-ds-leveldb.svg?branch=master)](https://travis-ci.org/ipfs/go-ds-leveldb)

> A go-datastore implementation using LevelDB

`go-ds-leveldb` implements the [go-datastore](https://github.com/ipfs/go-datastore) interface using a LevelDB backend.

## Lead Maintainer

[Steven Allen](https://github.com/Stebalien)

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Install

This module can be installed like a regular go module:

```
go get github.com/ipfs/go-ds-leveldb
```

It uses [Gx](https://github.com/whyrusleeping/gx) to manage dependencies. You can use `make deps` to rewrite imports to the gx-specified versions.

## Usage

```
import "github.com/ipfs/go-ds-leveldb"
```

Check the [GoDoc documentation](https://godoc.org/github.com/ipfs/go-ds-leveldb)


## Contribute

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Protocol Labs, Inc.
