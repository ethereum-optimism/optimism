#!/bin/bash
export PATH="/usr/local/opt/llvm/bin"
for arg do
  shift
  [ "$arg" == "-g" ] && continue
  #[ "$arg" == "-lpthread" ] && continue
  [ "$arg" == "-Werror" ] && continue
  #[ "$arg" == "-msoft-float" ] && continue
  set -- "$@" "$arg"
done
#echo "ARGS" $@ 1>&2
exec clang -lc -target mips-linux-gnu --sysroot /Users/kafka/fun/mips/sysroot -fuse-ld=lld -Wno-error -nostdlib "$@"
