# FastSSZ

The FastSSZ project in this reposity is a combination of two things: a high performant low level library to work with SSZ encodings (root of this project) and the ([sszgen](./sszgen)) code generator that generates the SSZ encodings for Go structs using the SSZ library. By combining both, this library achieves peak Go native performance and zero memory allocation. The repository uses as test the official Ethereum SSZ tests ([spectests](./spectests/)) for the Consensus Spec data structures.

If you are only looking for the Consensus data structures and types with the SSZ support, it is recommended to use [go-eth-consensus](https://github.com/umbracle/go-eth-consensus) instead, since it is already integrated with other parts of the Consensus stack, like the Beacon http API.

Clone:

```
$ git clone git@github.com:ferranbt/fastssz.git
```

Download the eth2.0 spec tests

```
$ make get-spec-tests
```

Regenerate the test spec encodings:

```
$ make build-spec-tests
```

Generate encodings for a specific package:

```
$ go run sszgen/*.go --path ./ethereumapis/eth/v1alpha1 [--objs BeaconBlock,Eth1Data]
```

Optionally, you can specify the objs you want to generate. Otherwise, it will generate encodings for all structs in the package. Note that if a struct does not have 'ssz' tags when required (i.e size of arrays), the generator will fail.

By default, it generates a file with the prefix '\_encoding.go' for each file that contains a generated struct. Optionally, you can combine all the outputs in a single file with the 'output' flag.

```
$ go run sszgen/*.go --path ./ethereumapis/eth/v1alpha1 --output ./ethereumapis/eth/v1alpha1/encoding.go
```

Test the spectests:

```
$ go test -v ./spectests/... -run TestSpec
```

Run the fuzzer:

```
$ FUZZ_TESTS=True go test -v ./spectests/... -run TestFuzz
```

To install the generator run:

```
$ go get github.com/ferranbt/fastssz/sszgen
```

Benchmark (BeaconBlock):

```
$ go test -v ./spectests/... -run=XXX -bench=.
goos: linux
goarch: amd64
pkg: github.com/ferranbt/fastssz/spectests
cpu: AMD Ryzen 5 2400G with Radeon Vega Graphics
BenchmarkMarshal_Fast
BenchmarkMarshal_Fast-8             	  291054	      4088 ns/op	    8192 B/op	       1 allocs/op
BenchmarkMarshal_SuperFast
BenchmarkMarshal_SuperFast-8        	  798883	      1354 ns/op	       0 B/op	       0 allocs/op
BenchmarkUnMarshal_Fast
BenchmarkUnMarshal_Fast-8           	   64065	     17614 ns/op	   11900 B/op	     210 allocs/op
BenchmarkHashTreeRoot_Fast
BenchmarkHashTreeRoot_Fast-8        	   25863	     45932 ns/op	       0 B/op	       0 allocs/op
BenchmarkHashTreeRoot_SuperFast
BenchmarkHashTreeRoot_SuperFast-8   	   54078	     21999 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/ferranbt/fastssz/spectests	5.501s
```

## Package reference

To reference a struct from another package use the '--include' flag to point to that package.

Example:

```
$ go run sszgen/*.go --path ./example2
$ go run sszgen/*.go --path ./example
[ERR]: could not find struct with name 'Checkpoint'
$ go run sszgen/*.go --path ./example --include ./example2
```

There are some caveats required to use this functionality.

- If multiple input paths import the same package, all of them need to import it with the same alias if any.
- If the folder of the package is not the same as the name of the package, any input file that imports this package needs to do it with an alias.

## Fast HashTreeRoot

`Fastssz` integrates with Prysm [gohashtree](https://github.com/prysmaticlabs/gohashtree) library to do high performance and concurrent Sha256 hashing. It achieves a 2x performance improvement with respect to the normal sequential hashing. As of now, this feature is not yet enabled by default since it does not use the `gohashtree` main branch. You can track the updates on [this](https://github.com/prysmaticlabs/gohashtree/issues/4) issue.

In order to use this feature, enable manually the hash function in the Hasher like in the benchmark example.
