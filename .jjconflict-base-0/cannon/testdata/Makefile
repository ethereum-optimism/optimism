all: elf

go1-24:
	make -C ./go-1-24 elf
.PHONY: go1-24

go1-25:
	make -C ./go-1-25 elf
.PHONY: go1-25

.PHONY: elf
elf: go1-24 go1-25

.PHONY: clean
clean:
	make -C ./go-1-24 clean
	make -C ./go-1-25 clean
