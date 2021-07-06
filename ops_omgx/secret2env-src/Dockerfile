FROM golang:1.14-alpine AS build

WORKDIR /src/
COPY . /src/
RUN env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o secret2env
