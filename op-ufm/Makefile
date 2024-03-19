LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

ufm:
	go build -v $(LDFLAGS) -o ./bin/ufm ./cmd/ufm
.PHONY: ufm

fmt:
	go mod tidy
	gofmt -w .
.PHONY: fmt

test:
	go test -race -v ./...
.PHONY: test

lint:
	go vet ./...
.PHONY: lint

tls:
	kubectl get secrets op-ufm-client-tls -o yaml | yq '.data."tls.key"' | base64 --decode > tls/tls.key
	kubectl get secrets op-ufm-client-tls -o yaml | yq '.data."tls.crt"' | base64 --decode > tls/tls.crt
	kubectl get secrets op-ufm-client-tls -o yaml | yq '.data."ca.crt"' | base64 --decode > tls/ca.crt
.PHONY: tls
