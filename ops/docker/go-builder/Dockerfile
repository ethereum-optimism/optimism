FROM golang:1.17.8-alpine3.15

RUN apk add --no-cache make gcc musl-dev linux-headers git jq curl bash gzip ca-certificates openssh && \
	go install gotest.tools/gotestsum@latest

CMD ["bash"]