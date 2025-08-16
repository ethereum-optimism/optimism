#!/bin/sh

# Credit where it's due: much of this script is copied from or inspired by [rustup] and [foundryup].
#
# [rustup]: https://sh.rustup.rs
# [foundrup]: https://foundry.paradigm.xyz

# All configs are here.
# If you modify the configs in any way, please also update the help text below.
OP_UP_VERSION="${OP_UP_VERSION:-0.1.1}" # The default version is hardcoded for now.
OP_UP_REPO="${OP_UP_REPO:-ethereum-optimism/optimism}"
OP_UP_DIR="${OP_UP_DIR:-"${HOME}/.op-up"}"

if [ "$#" != 0 ]; then
  echo "The op-up installer.

When no parameters are passed, the op-up command is installed from GitHub.
Anything else causes this help text to be printed.

The installation is configured via environment variables:

OP_UP_REPO:
  The GitHub repo from which to download op-up.
  (default: ${OP_UP_REPO})

OP_UP_VERSION:
  The semver-formatted version of the op-up command to install.
  (default: ${OP_UP_VERSION})

OP_UP_DIR:
  The main directory for the op-up command. The install directory is \"\${OP_UP_DIR}/bin\".
  (default: ${OP_UP_DIR})

NO_COLOR:
  Disables pretty text when set and nonempty. https://no-color.org/

The script only understands the GitHub releases API.
On error, the script exits with status code 1."
  exit
fi

# Use pretty text when the user's environment most likely supports it.
_text_bold=''
_text_red=''
_text_reset=''
# Ensure the script is running in a terminal, NO_COLOR is unset or empty, and tput is available.
if [ -t 1 ] && [ -z "${NO_COLOR-}" ] && command -v tput >/dev/null 2>&1; then
  ncolors=$(tput colors 2>/dev/null || printf 0)
  # Checking for 8 colors helps avoid weird edge cases on legacy or misconfigured systems.
  if [ "$ncolors" -ge 8 ]; then
    _text_bold=$(tput bold)
    _text_red=$(tput setaf 1)
    _text_reset=$(tput sgr0)
  fi
fi

say() {
  printf "op-up-installer: %s\n" "$1"
}

shout() {
  printf '%s' "${_text_bold}${_text_red}"
  say "${1}${_text_reset}"
}

err() {
  say "error: ${1}" >&2
  exit 1
}

ensure() {
  if ! "$@"; then err "command failed: ${*}"; fi
}

# Get the os type.

_ostype="$(uname -s)"

case "$_ostype" in
    Linux)
        _ostype=linux
        ;;
    Darwin)
        _ostype=darwin
        ;;
    *)
        err "os type is not Linux or Darwin: $_ostype"
        ;;
esac

# Get the cpu type.

_cputype="$(uname -m)"

# Working around Mac idiosyncrasies.
if [ "$_ostype" = Darwin ]; then
    # Darwin `uname -m` can lie due to Rosetta shenanigans. If you manage to
    # invoke a native shell binary and then a native uname binary, you can
    # get the real answer, but that's hard to ensure, so instead we use
    # `sysctl` (which doesn't lie) to check for the actual architecture.
    if [ "$_cputype" = i386 ]; then
        # Handling i386 compatibility mode in older macOS versions (<10.15)
        # running on x86_64-based Macs.
        # Starting from 10.15, macOS explicitly bans all i386 binaries from running.
        # See: <https://support.apple.com/en-us/HT208436>

        # Avoid `sysctl: unknown oid` stderr output and/or non-zero exit code.
        if sysctl hw.optional.x86_64 2> /dev/null || true | grep -q ': 1'; then
            _cputype=amd64
        fi
    elif [ "$_cputype" = x86_64 ]; then
        # Handling x86-64 compatibility mode (a.k.a. Rosetta 2)
        # in newer macOS versions (>=11) running on arm64-based Macs.
        # Rosetta 2 is built exclusively for x86-64 and cannot run i386 binaries.

        # Avoid `sysctl: unknown oid` stderr output and/or non-zero exit code.
        if sysctl hw.optional.arm64 2> /dev/null || true | grep -q ': 1'; then
            _cputype=arm64
        fi
    fi
fi

case "$_cputype" in
    aarch64 | arm64)
        _cputype=arm64
        ;;
    x86_64 | x86-64 | x64 | amd64)
        _cputype=amd64
        ;;
    *)
        err "unsupported cpu type: $_cputype"
esac

# Download the binary.

_binary_name="op-up"

_target="${_ostype}-${_cputype}"
say "downloading for target ${_target}..."
_file_without_ext="${_binary_name}-${OP_UP_VERSION}-${_target}"
_url="https://github.com/${OP_UP_REPO}/releases/download/${_binary_name}/v${OP_UP_VERSION}/${_file_without_ext}.tar.gz"
_archive=$(mktemp) || err "create temporary file"
ensure curl --location --proto '=https' --tlsv1.2 --silent --show-error --fail "$_url" --output "$_archive"
say 'downloaded'

# Extract to the destination.

say "installing..."
_install_dir="${OP_UP_DIR}/bin"
mkdir -p "$_install_dir"
ensure tar --verbose --extract --file "$_archive" --directory "$_install_dir" --strip-components 1
ensure chmod +x "${_install_dir}/${_binary_name}"
say 'installed'

# Update the PATH if necessary.

case ":${PATH}:" in

  *":${_install_dir}:"*)
    ;;

  *)

    say 'updating PATH...'
    say "finding shell profile for shell ${SHELL}..."
    case "$SHELL" in
      */zsh)
          _profile="${ZDOTDIR-"$HOME"}/.zshenv"
          ;;
      */bash)
          _profile="${HOME}/.bashrc"
          ;;
      */fish)
          _profile="${HOME}/.config/fish/config.fish"
          ;;
      */ash)
          _profile="${HOME}/.profile"
          ;;
      *)
          err "could not detect shell, manually add ${_install_dir} to your PATH."
    esac
    say "shell profile found at ${_profile}"

    echo >> "$_profile"
    if [ "$SHELL" = fish ]; then
        echo "fish_add_path -a ${_install_dir}" >> "$_profile"
    else
        echo "export PATH=\"\${PATH}:${_install_dir}\"" >> "$_profile"
    fi
    say 'updated PATH'
    shout "run 'source ${_profile}' or start a new terminal session to use op-up"

esac
