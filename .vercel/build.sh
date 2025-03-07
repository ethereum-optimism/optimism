#!/bin/bash
set -e

mkdir -p $HOME/bin
export PATH=$HOME/bin:$PATH
REPO_ROOT=$(pwd)

echo "Installing mdbook..."
curl -sSL https://github.com/rust-lang/mdBook/releases/download/v0.4.14/mdbook-v0.4.14-x86_64-unknown-linux-gnu.tar.gz | tar -xz
chmod +x ./mdbook
cp ./mdbook $HOME/bin/

echo "Installing mdbook-template..."
curl -sSL https://github.com/sgoudham/mdbook-template/releases/latest/download/mdbook-template-x86_64-unknown-linux-gnu.tar.gz | tar -xz
chmod +x ./mdbook-template
cp ./mdbook-template $HOME/bin/

echo "Installing Rust..."
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | RUSTUP_HOME=/tmp/rustup HOME=/tmp sh -s -- -y
export PATH="/tmp/.cargo/bin:$PATH"
export RUSTUP_HOME=/tmp/rustup
export CARGO_HOME=/tmp/.cargo

rustup toolchain install nightly
rustup default nightly

echo "Verifying installations..."
which mdbook
which mdbook-template
rustc --version

echo "Building the book..."
cd "${REPO_ROOT}/book"
mdbook build

echo "Building Rust documentation..."
cd "${REPO_ROOT}"
export RUSTDOCFLAGS="--cfg docsrs --show-type-layout --generate-link-to-definition --enable-index-page -Zunstable-options"
cargo doc --all-features --no-deps || {
  echo "Cargo doc failed, but continuing with mdbook only"
}

# Try to copy the API docs if they were generated
if [ -d "${REPO_ROOT}/target/doc" ]; then
  echo "Copying API documentation..."
  mkdir -p "${REPO_ROOT}/book/book/api"
  cp -r "${REPO_ROOT}/target/doc"/* "${REPO_ROOT}/book/book/api/"
else
  echo "API documentation not generated, continuing with mdbook only"
fi

# Create a final output directory for Vercel
mkdir -p "${REPO_ROOT}/vercel-output"
cp -r "${REPO_ROOT}/book/book"/* "${REPO_ROOT}/vercel-output/"

echo "Build completed successfully!"
