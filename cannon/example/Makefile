all: elf

.PHONY: elf
elf: $(patsubst %/go.mod,bin/%.elf,$(wildcard */go.mod))

bin:
	mkdir bin

# take any directory with a go mod, and build an ELF
# verify output with: readelf -h bin/<name>.elf
# result is mips32, big endian, R3000
bin/%.elf: bin
	cd $(@:bin/%.elf=%) && GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o ../$@ .
