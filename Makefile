opnode:
	go build -o ./bin/op ./opnode/cmd
.PHONY: opnode

clean:
	rm -rf ./bin
.PHONY: clean
