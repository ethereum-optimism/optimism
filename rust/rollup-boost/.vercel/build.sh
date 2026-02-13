#!/bin/bash
set -e

mkdir -p $HOME/bin
export PATH=$HOME/bin:$PATH
REPO_ROOT=$(pwd)

echo "Installing system dependencies..."
apt-get update && apt-get install -y clang libclang-dev

echo "Installing Rust..."
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | RUSTUP_HOME=/tmp/rustup HOME=/tmp sh -s -- -y
export PATH="/tmp/.cargo/bin:$PATH"
export RUSTUP_HOME=/tmp/rustup
export CARGO_HOME=/tmp/.cargo

rustup toolchain install nightly
rustup default nightly

echo "Installing mdbook..."
cargo install mdbook
cargo install mdbook-template
cargo install mdbook-mermaid

echo "Verifying installations..."
which mdbook
which mdbook-template
rustc --version

echo "Building the book..."
cd "${REPO_ROOT}/book"
mdbook build

echo "Skipping Rust documentation build to avoid libclang dependency..."
echo "Building mdbook documentation only..."

# Create a final output directory for Vercel
mkdir -p "${REPO_ROOT}/vercel-output"
cp -r "${REPO_ROOT}/book/book"/* "${REPO_ROOT}/vercel-output/"

echo "Build completed successfully!"
