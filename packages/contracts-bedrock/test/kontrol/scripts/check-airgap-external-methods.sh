#!/usr/bin/env bash
set -euo pipefail

# These hashes commit to the complete ABI of each production artifact, including function
# mutability. A new selector, or a getter changed from view to state-mutating, must update this
# manifest and the transition-closure proof model in the same review; otherwise the Kontrol job
# fails before symbolic execution.
contracts=(
  OptimismPortal2
  FaultDisputeGame
  PermissionedDisputeGame
  SuperFaultDisputeGame
  ZKDisputeGame
  SuperPermissionedDisputeGame
)

expected=(
  0x3d0a8839b5a6b6b2709b9ec069af205eae5fbb823a0df26a0c32c2d2489eb751
  0x8d1898ac97d75ec217565ccd4c4d15236447dce6c4e421921010d79f504f3659
  0x9e7cb2389ae65b3df79b75693568549fa298ef259d7771a1734012829b5a062c
  0x325694a70f85077d99b822772c932450ab1f1700927ae15942ba7618b787d620
  0xc8e237b16f1b06f721de86dd8a3305a10b4f044e02386b77f4b70b38a141dac9
  0xa849b277c92693bbc42047354b93371a8f0cbbd07fb1f35c8f18330602a21947
)

for i in "${!contracts[@]}"; do
  actual=$(forge inspect "${contracts[$i]}" abi --json | cast keccak)
  if [[ "$actual" != "${expected[$i]}" ]]; then
    echo "External ABI coverage changed for ${contracts[$i]}" >&2
    echo "expected: ${expected[$i]}" >&2
    echo "actual:   $actual" >&2
    echo "Review the ABI change and extend the transition-closure proofs before updating this hash." >&2
    exit 1
  fi
done

echo "Air-gap external method coverage passed"
