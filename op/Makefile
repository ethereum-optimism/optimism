PRESTATE ?= prestate-mt64.bin.gz
# "remote" in the sense of "not in this directory"
REMOTE_PRESTATE ?= ../op-program/bin/$(PRESTATE)
ARTIFACTS ?= ../op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tgz
BINARY ?= op

# TODO make circleci use this target.
# circleci uses this `build` target.
build: $(PRESTATE) $(ARTIFACTS)
	go build -o $(BINARY) .
.PHONY: build

$(PRESTATE): $(REMOTE_PRESTATE)
	cp $^ .

$(REMOTE_PRESTATE):
	make cannon-prestates -C ../

$(ARTIFACTS):
	just ../op-deployer/copy-contract-artifacts

clean:
	rm -f $(BINARY) $(PRESTATE)
.PHONY: clean
