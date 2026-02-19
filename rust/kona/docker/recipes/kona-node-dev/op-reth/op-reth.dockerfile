FROM ghcr.io/paradigmxyz/op-reth:nightly AS reth

FROM ubuntu:latest

COPY --from=reth /usr/local/bin/op-reth /usr/local/bin/op-reth

WORKDIR /

COPY jwttoken/jwt.hex /

ENTRYPOINT [ "op-reth" ]
