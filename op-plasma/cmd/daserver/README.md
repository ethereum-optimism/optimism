# Plasma DA Server

## Introduction

This simple DA server implementation support local storage via LevelDB and remove via S3.
LevelDB is only recommended for usage in local Devnets where connecting to S3 is not convenient.
See the [S3 doc](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/) for more information
on how to configure the S3 client.
