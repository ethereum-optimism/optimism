#!/usr/bin/env bash
# apt-get update + install. Arguments are passed to `apt-get install`.
set -euo pipefail

# The machine images point apt at the single-region EC2 mirror
# (us-east-1.ec2.archive.ubuntu.com), which is one set of AWS hosts with no
# routing of its own: when it degrades, apt has nowhere else to go, and it is only
# nearby when the runner happens to be in that region. archive.ubuntu.com is
# Cloudflare anycast, so redundancy, geo-routing and caching are the CDN's job.
# Third-party lists keep their own URIs.
for f in /etc/apt/sources.list /etc/apt/sources.list.d/*.sources /etc/apt/sources.list.d/*.list; do
  if [ -f "$f" ]; then
    sudo sed -i -E "s#https?://[A-Za-z0-9.-]+\.ec2\.archive\.ubuntu\.com/ubuntu#http://archive.ubuntu.com/ubuntu#g" "$f"
  fi
done

# Timeouts, so a mirror that stops responding errors out instead of being waited on.
APT_OPTS=(-o Acquire::Retries=3 -o Acquire::http::Timeout=20 -o Acquire::https::Timeout=20)

export NEEDRESTART_MODE=a
export DEBIAN_FRONTEND=noninteractive

sudo -E apt-get "${APT_OPTS[@]}" update --error-on=any
sudo -E apt-get "${APT_OPTS[@]}" install -y "$@"
