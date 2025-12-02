FROM ubuntu:latest

RUN apt-get update -y && apt-get upgrade -y && apt install -y ca-certificates

WORKDIR /

COPY kona-node/kona/target/release/kona-node /usr/local/bin

RUN mkdir -p /11155420

COPY kona-node/bootstores/sepolia.json /11155420/bootstore.json
COPY jwttoken/jwt.hex /

ENTRYPOINT [ "kona-node" ]
