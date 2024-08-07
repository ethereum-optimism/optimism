#!/bin/bash

set -e

function maketest() {
  local src="$1"
  local out="$2"
  
  printf "building %s -> %s" "$src" "$out"

  # Create a temporary file
  full_bin=$(mktemp)

  # Assemble the full test vector
  mips-linux-gnu-as -defsym big_endian=1 -march=mips64 -o "$full_bin" "$src"

  # Copy the `.test` section data to a temporary file
  section_data=$(mktemp)
  mips-linux-gnu-objcopy --dump-section .test="$section_data" "$full_bin"
  
  # Write the .test section data to the output file
  cp "$section_data" "$out"

  # Clean up the temporary files
  rm "$full_bin" "$section_data"

  printf " ✅\n"
}

mkdir -p /tmp/mips

if [ "$#" -gt 0 ]; then
  maketest "$1" "test/bin/$(basename "$1" .asm).bin" 
else
  for d in test/*.asm; 
  do
    [ -e "$d" ] || continue
    maketest "$d" "test/bin/$(basename "$d" .asm).bin"
  done

  echo "[🧙] All tests built successfully. God speed, space cowboy."
fi
