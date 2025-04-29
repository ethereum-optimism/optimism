all: elf

go1-22:
	make -C ./go-1-22 elf
.PHONY: go1-22

go1-23:
	make -C ./go-1-23 elf
.PHONY: go1-23

.PHONY: elf
elf: go1-22 go1-23

