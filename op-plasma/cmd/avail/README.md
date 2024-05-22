# Plasma Avail DA Server

## Introduction

This introduces a DA Server that interacts with Avail DA for posting and retrieving data.

## Usage

- You will need to get a seed phrase, funded with Avail tokens, and an App ID. Steps to generate them can be found [here](https://docs.availproject.org/docs/end-user-guide)
- To run the da server, run:

```
go run ./cmd/avail  --addr=localhost --port=8000 --avail.rpc=<Avail RPC URL> --avail.seed="<seed phrase>" --avail.appid=<APP ID> --avail.timeout=<Timeout>
```
