#!/bin/bash
export PATH="/usr/local/opt/llvm/bin:$PATH"
echo "ARGS" $@ 1>&2
clang $@ -lc -target mips-linux-gnu --sysroot /Users/kafka/fun/mips/sysroot -fuse-ld=lld -Wno-error -nostdlib

