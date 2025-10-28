#!/bin/sh
set -eu

if command -v mise; then
    echo "mise already installed at $(command -v mise)"
    exit 0
fi

#region logging setup
if [ "${MISE_DEBUG-}" = "true" ] || [ "${MISE_DEBUG-}" = "1" ]; then
  debug() {
    echo "$@" >&2
  }
else
  debug() {
    :
  }
fi

if [ "${MISE_QUIET-}" = "1" ] || [ "${MISE_QUIET-}" = "true" ]; then
  info() {
    :
  }
else
  info() {
    echo "$@" >&2
  }
fi

error() {
  echo "$@" >&2
  exit 1
}
#endregion

#region environment setup
get_os() {
  os="$(uname -s)"
  if [ "$os" = Darwin ]; then
    echo "macos"
  elif [ "$os" = Linux ]; then
    echo "linux"
  else
    error "unsupported OS: $os"
  fi
}

get_arch() {
  musl=""
  if type ldd >/dev/null 2>/dev/null; then
    libc=$(ldd /bin/ls | grep 'musl' | head -1 | cut -d ' ' -f1)
    if [ -n "$libc" ]; then
      musl="-musl"
    fi
  fi
  arch="$(uname -m)"
  if [ "$arch" = x86_64 ]; then
    echo "x64$musl"
  elif [ "$arch" = aarch64 ] || [ "$arch" = arm64 ]; then
    echo "arm64$musl"
  elif [ "$arch" = armv7l ]; then
    echo "armv7$musl"
  else
    error "unsupported architecture: $arch"
  fi
}

get_ext() {
  if [ -n "${MISE_INSTALL_EXT:-}" ]; then
    echo "$MISE_INSTALL_EXT"
  elif [ -n "${MISE_VERSION:-}" ] && echo "$MISE_VERSION" | grep -q '^v2024'; then
    # 2024 versions don't have zstd tarballs
    echo "tar.gz"
  elif tar_supports_zstd; then
    echo "tar.zst"
  elif command -v zstd >/dev/null 2>&1; then
    echo "tar.zst"
  else
    echo "tar.gz"
  fi
}

tar_supports_zstd() {
  # tar is bsdtar or version is >= 1.31
  if tar --version | grep -q 'bsdtar' && command -v zstd >/dev/null 2>&1; then
    true
  elif tar --version | grep -q '1\.(3[1-9]|[4-9][0-9]'; then
    true
  else
    false
  fi
}

shasum_bin() {
  if command -v shasum >/dev/null 2>&1; then
    echo "shasum"
  elif command -v sha256sum >/dev/null 2>&1; then
    echo "sha256sum"
  else
    error "mise install requires shasum or sha256sum but neither is installed. Aborting."
  fi
}

get_checksum() {
  version=$1
  os="$(get_os)"
  arch="$(get_arch)"
  ext="$(get_ext)"
  url="https://github.com/jdx/mise/releases/download/v${version}/SHASUMS256.txt"

  # For current version use static checksum otherwise
  # use checksum from releases
  if [ "$version" = "v2025.3.2" ]; then
    checksum_linux_x86_64="5d2e40270c32c906385c1b35c8dac0f995c4765b429aff5fe67e9efc9957aed5  ./mise-v2025.3.2-linux-x64.tar.gz"
    checksum_linux_x86_64_musl="d8312658bd020b717a90bc2c3729625656c97b37c70edda645ce8bcbd68edf56  ./mise-v2025.3.2-linux-x64-musl.tar.gz"
    checksum_linux_arm64="8dd66426b9f36f5c1bed161d3d79b2e12294c1c8a9801b1f654311e008787a5b  ./mise-v2025.3.2-linux-arm64.tar.gz"
    checksum_linux_arm64_musl="7746d2ab653caa73128936dd2bc7704fe4f106fa58597d65f5020843767f3ceb  ./mise-v2025.3.2-linux-arm64-musl.tar.gz"
    checksum_linux_armv7="a2a626b3e1d393e19fe600103e97659acec8de18f6868014928a2a3f68301307  ./mise-v2025.3.2-linux-armv7.tar.gz"
    checksum_linux_armv7_musl="e3dc41cdf7ce857c322b8713fc619708ad7c8f1da8d5b00b7a6fb62f215819f6  ./mise-v2025.3.2-linux-armv7-musl.tar.gz"
    checksum_macos_x86_64="c1f58a5772b80e2d7fe64f6c0dd832c38c5146deb763130f71f09984a5328acb  ./mise-v2025.3.2-macos-x64.tar.gz"
    checksum_macos_arm64="d5f836d2e946201147be4ca3467c547ab847a14f93b26c6a4f736804fc2107ef  ./mise-v2025.3.2-macos-arm64.tar.gz"
    checksum_linux_x86_64_zstd="ea4f403772ec034bc11f654f410aeaade40fcffc1cee78fd02340f8bae3e9db7  ./mise-v2025.3.2-linux-x64.tar.zst"
    checksum_linux_x86_64_musl_zstd="c7c6f96421658eb7cb8f7fd8cb37e36fc91b025d0564899828376d07789a80b4  ./mise-v2025.3.2-linux-x64-musl.tar.zst"
    checksum_linux_arm64_zstd="5696404d02481cd8bd9d9d8a721ec80141f3f74cb0f77f3eee0bbf3fc0f9b20b  ./mise-v2025.3.2-linux-arm64.tar.zst"
    checksum_linux_arm64_musl_zstd="68695587360d52bd6889e11ff79ff1b18a533fc81c56dadf213db9364b1ca082  ./mise-v2025.3.2-linux-arm64-musl.tar.zst"
    checksum_linux_armv7_zstd="6411d630c4515a49ab903865647a1da0fa3fe97c4a990ccd078364abcb6b2d7b  ./mise-v2025.3.2-linux-armv7.tar.zst"
    checksum_linux_armv7_musl_zstd="fbcbd945b7700f9c54e7dea423010889df1a51ca3edf25b20a01766e706619d5  ./mise-v2025.3.2-linux-armv7-musl.tar.zst"
    checksum_macos_x86_64_zstd="af0af814ad8aa8828bd4cef455f3edc4059984ea3490bec4827778ca9c73951b  ./mise-v2025.3.2-macos-x64.tar.zst"
    checksum_macos_arm64_zstd="30763db42007598d8c6042d88bf6045bb3834e8d12a0844a5aa0005d609beb8b  ./mise-v2025.3.2-macos-arm64.tar.zst"

    # TODO: refactor this, it's a bit messy
    if [ "$(get_ext)" = "tar.zst" ]; then
      if [ "$os" = "linux" ]; then
        if [ "$arch" = "x64" ]; then
          echo "$checksum_linux_x86_64_zstd"
        elif [ "$arch" = "x64-musl" ]; then
          echo "$checksum_linux_x86_64_musl_zstd"
        elif [ "$arch" = "arm64" ]; then
          echo "$checksum_linux_arm64_zstd"
        elif [ "$arch" = "arm64-musl" ]; then
          echo "$checksum_linux_arm64_musl_zstd"
        elif [ "$arch" = "armv7" ]; then
          echo "$checksum_linux_armv7_zstd"
        elif [ "$arch" = "armv7-musl" ]; then
          echo "$checksum_linux_armv7_musl_zstd"
        else
          warn "no checksum for $os-$arch"
        fi
      elif [ "$os" = "macos" ]; then
        if [ "$arch" = "x64" ]; then
          echo "$checksum_macos_x86_64_zstd"
        elif [ "$arch" = "arm64" ]; then
          echo "$checksum_macos_arm64_zstd"
        else
          warn "no checksum for $os-$arch"
        fi
      else
        warn "no checksum for $os-$arch"
      fi
    else
      if [ "$os" = "linux" ]; then
        if [ "$arch" = "x64" ]; then
          echo "$checksum_linux_x86_64"
        elif [ "$arch" = "x64-musl" ]; then
          echo "$checksum_linux_x86_64_musl"
        elif [ "$arch" = "arm64" ]; then
          echo "$checksum_linux_arm64"
        elif [ "$arch" = "arm64-musl" ]; then
          echo "$checksum_linux_arm64_musl"
        elif [ "$arch" = "armv7" ]; then
          echo "$checksum_linux_armv7"
        elif [ "$arch" = "armv7-musl" ]; then
          echo "$checksum_linux_armv7_musl"
        else
          warn "no checksum for $os-$arch"
        fi
      elif [ "$os" = "macos" ]; then
        if [ "$arch" = "x64" ]; then
          echo "$checksum_macos_x86_64"
        elif [ "$arch" = "arm64" ]; then
          echo "$checksum_macos_arm64"
        else
          warn "no checksum for $os-$arch"
        fi
      else
        warn "no checksum for $os-$arch"
      fi
    fi
  else
    if command -v curl >/dev/null 2>&1; then
      debug ">" curl -fsSL "$url"
      checksums="$(curl --compressed -fsSL "$url")"
    else
      if command -v wget >/dev/null 2>&1; then
        debug ">" wget -qO - "$url"
        stderr=$(mktemp)
        checksums="$(wget -qO - "$url")"
      else
        error "mise standalone install specific version requires curl or wget but neither is installed. Aborting."
      fi
    fi
    # TODO: verify with minisign or gpg if available

    checksum="$(echo "$checksums" | grep "$os-$arch.$ext")"
    if ! echo "$checksum" | grep -Eq "^([0-9a-f]{32}|[0-9a-f]{64})"; then
      warn "no checksum for mise $version and $os-$arch"
    else
      echo "$checksum"
    fi
  fi
}

#endregion

download_file() {
  url="$1"
  filename="$(basename "$url")"
  cache_dir="$(mktemp -d)"
  file="$cache_dir/$filename"

  info "mise: installing mise..."

  if command -v curl >/dev/null 2>&1; then
    debug ">" curl -#fLo "$file" "$url"
    curl -#fLo "$file" "$url"
  else
    if command -v wget >/dev/null 2>&1; then
      debug ">" wget -qO "$file" "$url"
      stderr=$(mktemp)
      wget -O "$file" "$url" >"$stderr" 2>&1 || error "wget failed: $(cat "$stderr")"
    else
      error "mise standalone install requires curl or wget but neither is installed. Aborting."
    fi
  fi

  echo "$file"
}

install_mise() {
  version="${MISE_VERSION:-v2025.3.2}"
  version="${version#v}"
  os="$(get_os)"
  arch="$(get_arch)"
  ext="$(get_ext)"
  install_path="${MISE_INSTALL_PATH:-$HOME/.local/bin/mise}"

  if [ -x "$install_path" ]; then
    debug "mise-setup: already installed"
    exit 0
  fi

  install_dir="$(dirname "$install_path")"
  tarball_url="https://github.com/jdx/mise/releases/download/v${version}/mise-v${version}-${os}-${arch}.${ext}"

  cache_file=$(download_file "$tarball_url")
  debug "mise-setup: tarball=$cache_file"

  debug "validating checksum"
  cd "$(dirname "$cache_file")" && get_checksum "$version" | "$(shasum_bin)" -c >/dev/null

  # extract tarball
  mkdir -p "$install_dir"
  rm -rf "$install_path"
  cd "$(mktemp -d)"
  if [ "$(get_ext)" = "tar.zst" ] && ! tar_supports_zstd; then
    zstd -d -c "$cache_file" | tar -xf -
  else
    tar -xf "$cache_file"
  fi
  mv mise/bin/mise "$install_path"
  info "mise: installed successfully to $install_path"
}

after_finish_help() {
  case "${SHELL:-}" in
  */zsh)
    info "mise: run the following to activate mise in your shell:"
    info "echo \"eval \\\"\\\$($install_path activate zsh)\\\"\" >> \"${ZDOTDIR-$HOME}/.zshrc\""
    info ""
    info "mise: run \`mise doctor\` to verify this is setup correctly"
    ;;
  */bash)
    info "mise: run the following to activate mise in your shell:"
    info "echo \"eval \\\"\\\$($install_path activate bash)\\\"\" >> ~/.bashrc"
    info ""
    info "mise: run \`mise doctor\` to verify this is setup correctly"
    ;;
  */fish)
    info "mise: run the following to activate mise in your shell:"
    info "echo \"$install_path activate fish | source\" >> ~/.config/fish/config.fish"
    info ""
    info "mise: run \`mise doctor\` to verify this is setup correctly"
    ;;
  *)
    info "mise: run \`$install_path --help\` to get started"
    ;;
  esac
}

install_mise
if [ "${MISE_INSTALL_HELP-}" != 0 ]; then
  after_finish_help
fi
