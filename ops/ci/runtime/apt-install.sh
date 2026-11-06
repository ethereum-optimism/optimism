#!/usr/bin/env bash
# apt-get update + install. Arguments are passed to `apt-get install`.
set -euo pipefail

# Timeouts, so a mirror that stops responding errors out instead of being waited on.
APT_OPTS=(-o Acquire::Retries=3 -o Acquire::http::Timeout=20 -o Acquire::https::Timeout=20)

export NEEDRESTART_MODE=a
export DEBIAN_FRONTEND=noninteractive

# Copy of each list as the machine image shipped it. The suffix is not one apt
# reads (it only picks up *.list and *.sources), so the copies sit inertly next
# to the rewritten files until the fallback below needs them. Written once: some
# jobs call this script more than once, and re-copying would capture the
# already-rewritten file and make the fallback a no-op.
BACKUP_SUFFIX=".shipped-mirrors"
# Set once archive.ubuntu.com has been found unusable, so later calls in the
# same job don't rewrite back onto it.
FALLBACK_MARKER="/tmp/.apt-install-mirror-fallback"

SOURCE_FILES=()
for f in /etc/apt/sources.list /etc/apt/sources.list.d/*.sources /etc/apt/sources.list.d/*.list; do
  if [ -f "$f" ]; then
    SOURCE_FILES+=("$f")
  fi
done

# The machine images point apt at the single-region EC2 mirror
# (us-east-1.ec2.archive.ubuntu.com), which is one set of AWS hosts with no
# routing of its own: when it degrades, apt has nowhere else to go, and it is only
# nearby when the runner happens to be in that region. archive.ubuntu.com is
# Cloudflare anycast, so redundancy, geo-routing and caching are the CDN's job.
# Third-party lists keep their own URIs.
if [ ! -f "$FALLBACK_MARKER" ]; then
  for f in "${SOURCE_FILES[@]}"; do
    if [ ! -f "$f$BACKUP_SUFFIX" ]; then
      sudo cp -p -- "$f" "$f$BACKUP_SUFFIX"
    fi
    sudo sed -i -E "s#https?://[A-Za-z0-9.-]+\.ec2\.archive\.ubuntu\.com/ubuntu#http://archive.ubuntu.com/ubuntu#g" "$f"
  done
fi

# The CDN is a single point of failure of its own: Canonical has had
# archive.ubuntu.com outages that fail every job installing a package through it.
# The EC2 mirror the image shipped is usually still serving when that happens, so
# put it back and try once more instead of failing the job outright.
restore_shipped_mirrors() {
  for f in "${SOURCE_FILES[@]}"; do
    if [ -f "$f$BACKUP_SUFFIX" ]; then
      sudo cp -p -- "$f$BACKUP_SUFFIX" "$f"
    fi
  done
  touch "$FALLBACK_MARKER"
}

apt_update_and_install() {
  sudo -E apt-get "${APT_OPTS[@]}" update --error-on=any
  sudo -E apt-get "${APT_OPTS[@]}" install -y "$@"
}

if apt_update_and_install "$@"; then
  exit 0
fi

echo "apt failed against archive.ubuntu.com; retrying with the mirrors the machine image shipped." >&2
restore_shipped_mirrors
apt_update_and_install "$@"
