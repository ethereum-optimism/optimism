all: elf

go1-23:
	make -C ./go-1-23 elf
.PHONY: go1-23

go1-24:
	make -C ./go-1-24 elf
.PHONY: go1-24

.PHONY: elf
elf: go1-23 go1-24

.PHONY: clean
clean:
	make -C ./go-1-23 clean
	make -C ./go-1-24 clean
