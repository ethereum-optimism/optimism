FROM golang:1.18

RUN apt-get -y update && apt-get -y upgrade && apt-get -y install wget && apt-get install ca-certificates -y

RUN go version

RUN mkdir -p /go-substrate-rpc-client
WORKDIR /go-substrate-rpc-client

# Reset parent entrypoint
ENTRYPOINT []
CMD ["make", "test-cover"]
