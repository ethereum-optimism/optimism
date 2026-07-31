#!/bin/bash
#
# Build a customized benchmark report directory (with HTML + images), using the original criterion report as input.
#
set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <input-dir> <output-dir>"
    exit 1
fi

script_dir=$( dirname -- "$0"; )
criterion_dir=$( realpath -s $1 )
output_dir=$( realpath -s $2 )

source "${script_dir}/env-vars.sh"

echo "Input directory:  $criterion_dir"
echo "Output directory: $output_dir"

# Error if input_dir doesn't exist
if [ ! -d "$criterion_dir" ]; then
    echo "Error: input directory '$criterion_dir' does not exist."
    exit 1
fi

cmd="${script_dir}/criterion-get-changes.py $criterion_dir --output-format html --only-significant"
export BENCH_CHANGES_HTML=$( $cmd )

# Clean up the output directory and create a copy of the input directory
rm -rf "$output_dir"
cp -r "$criterion_dir" "$output_dir"

# Iterate over all html files in the input directory, recursively
find $output_dir -type f -name "*.html" | while read file; do
    echo "- Updating $file ..."
    python3 "${script_dir}/criterion-update-html.py" "$file" "$file"
done
