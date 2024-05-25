all: elf dump

.PHONY: elf
elf: $(patsubst %/go.mod,bin/%.elf,$(wildcard */go.mod))

.PHONY: dump
dump: $(patsubst %/go.mod,bin/%.dump,$(wildcard */go.mod))

bin:
	mkdir bin

# take any directory with a go mod, and build an ELF
# verify output with: readelf -h bin/<name>.elf
# result is mips32, big endian, R3000
bin/%.elf: bin
	cd $(@:bin/%.elf=%) && GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o ../$@ .

# take any ELF and dump it
# TODO: currently have the little-endian toolchain, but should use the big-endian one. The -EB compat flag works though.
bin/%.dump: bin/%.elf
	mipsel-linux-gnu-objdump -D --disassembler-options=no-aliases --wide --source -m mips:3000 -EB $(@:%.dump=%.elf) > $@
