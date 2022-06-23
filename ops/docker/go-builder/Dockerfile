FROM ethereum/client-go:alltools-v1.10.17 as geth

FROM golang:1.18.0-alpine3.15

COPY --from=geth /usr/local/bin/abigen /usr/local/bin/abigen

RUN apk add --no-cache make gcc musl-dev linux-headers git jq curl bash gzip ca-certificates openssh && \
	go install gotest.tools/gotestsum@latest && \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.46.2

CMD ["bash"]
