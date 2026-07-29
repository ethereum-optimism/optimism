FROM ubuntu:latest

RUN apt-get -o Acquire::Retries=8 update -y && apt-get -o Acquire::Retries=8 upgrade -y && apt-get -o Acquire::Retries=8 install -y ca-certificates

WORKDIR /

COPY kona-node/kona/target/release/kona-node /usr/local/bin

RUN mkdir -p /11155420

COPY kona-node/bootstores/sepolia.json /11155420/bootstore.json
COPY jwttoken/jwt.hex /

ENTRYPOINT [ "kona-node" ]
