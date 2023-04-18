
bin:
	mkdir bin

bin/minimal.elf: bin
	cd minimal && GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o ../bin/minimal.elf .
	# verify output with: readelf -h bin/minimal.elf
	# result is mips32, big endian, R3000

bin/minimal.dump: bin/minimal.elf
	# TODO: currently have the little-endian toolchain, but should use the big-endian one. The -EB compat flag works though.
	mipsel-linux-gnu-objdump -D --disassembler-options=no-aliases --wide --source -m mips:3000 -EB bin/minimal.elf > bin/minimal.dump
