# Vendored SP1 Groth16 test artifacts

These files are consumed only by tests. They pin the SP1 **v6.0.0** Groth16 circuit, which is not
the production circuit (`vk/groth16_vk_v6.1.0.bin`). Tests call `Verify` with
`Circuit{VK: groth16_vk_v6.0.0.bin, VKRoot: 0x008cd56e10c2fe24795cff1e1d1f40d3a324528d315674da45d26afb376e8670}`.

| file | upstream | upstream revision | sha256 |
|---|---|---|---|
| `groth16-fixture-v6.0.0.json` | `succinctlabs/sp1-project-template`, `contracts/src/fixtures/groth16-fixture.json` | commit `d1b0924587efd7f4b6c2797d6fdd8282da71d088` (main, 2026-02-20) | `ee35af16543ee044e6a9600251cc0767ebc268d2bd7530b82fbdb2dbca4de799` |
| `groth16_vk_v6.0.0.bin` | crates.io `sp1-verifier` 6.0.0, `vk-artifacts/groth16_vk.bin` (`https://static.crates.io/crates/sp1-verifier/sp1-verifier-6.0.0.crate`, crate sha256 `09466f72acb4d511e5c896ea6dd8a792641e6eb202d77db323bc38ac60a3c53b`) | `succinctlabs/sp1` commit `f87f8d6ff005d542db22e241928319f5e96a4609` (`.cargo_vcs_info.json`) | `0e78f4db7a6771a3a6a7d9c3b0de6fe73d58781368967a7fe84d87aefffec896` (= `VERIFIER_HASH()` of `sp1-contracts` `v6.0.0/SP1VerifierGroth16.sol`) |

The fixture is Fibonacci n=20: `vkey 0x004a55ed3c7a07d0233a027278a8b7ff8681ffbd5d1ec4795c18966f6e693090`,
public values `abi.encode(20, 6765, 10946)`, 356-byte proof with prefix `0x0e78f4db`.

Production constant (`../vk/groth16_vk_v6.1.0.bin`): copied byte-for-byte from crates.io
`sp1-verifier` 6.8.0 `vk-artifacts/groth16_vk.bin` (`succinctlabs/sp1` commit
`58c4aeadbc504c274dd9fb82ed8130d0939ab756`), 492 bytes, sha256
`4388a21c687fdd5f218d7e3d13190cac4c5355818d3605fd5fb811df468ee696` (= `VERIFIER_HASH()` of
`sp1-contracts` `v6.1.0/SP1VerifierGroth16.sol`), `VK_ROOT = 0x002f850ee998974d6cc00e50cd0814b098c05bfade466d28573240d057f25352`
(`sp1-verifier-6.8.0/src/lib.rs` `VK_ROOT_BYTES`).
