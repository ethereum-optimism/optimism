#!/bin/bash
export PATH="/usr/local/opt/llvm/bin"
for arg do
  shift
  [ "$arg" == "-g" ] && continue
  #[ "$arg" == "-lpthread" ] && set -- "$@" "/Users/kafka/fun/mips/build/sysroot/lib/crt1.o"
  #[ "$arg" == "-lpthread" ] && continue
  [ "$arg" == "-Werror" ] && continue
  [ "$arg" == "-fno-caret-diagnostics" ] && continue
  #[ "$arg" == "-msoft-float" ] && continue
  set -- "$@" "$arg"
done
echo "ARGS" $@ 1>&2
exec clang -g0 -lc -target mips-linux-gnu --sysroot /Users/kafka/fun/mips/build/sysroot -fuse-ld=lld -Wno-error -nostdlib /Users/kafka/fun/mips/build/sysroot/lib/crt1.o "$@"
