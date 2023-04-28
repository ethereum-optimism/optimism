
bin:
	mkdir bin

bin/hello.elf: bin
	cd hello && GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o ../bin/hello.elf .
	# verify output with: readelf -h bin/hello.elf
	# result is mips32, big endian, R3000

bin/hello.dump: bin/hello.elf
	# TODO: currently have the little-endian toolchain, but should use the big-endian one. The -EB compat flag works though.
	mipsel-linux-gnu-objdump -D --disassembler-options=no-aliases --wide --source -m mips:3000 -EB bin/hello.elf > bin/hello.dump
