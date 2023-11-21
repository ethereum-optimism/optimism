arch=$(echo -n "$TARGETPLATFORM" | cut -d '/' -f2)

ver="v4.0.8"

download_bin() {
  curl -L -o "/usr/local/bin/$1" "https://github.com/prysmaticlabs/prysm/releases/download/$ver/$1-$ver-linux-$arch"
  chmod +x "/usr/local/bin/$1"
}

download_bin prysmctl
download_bin validator
download_bin beacon-chain