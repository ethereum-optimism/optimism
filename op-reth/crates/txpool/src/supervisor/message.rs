//! Interop message primitives.
// Source: https://github.com/op-rs/kona
// Copyright © 2023 kona contributors Copyright © 2024 Optimism
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
// associated documentation files (the “Software”), to deal in the Software without restriction,
// including without limitation the rights to use, copy, modify, merge, publish, distribute,
// sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or
// substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
// NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
/// An [`ExecutingDescriptor`] is a part of the payload to `supervisor_checkAccessList`
/// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#executingdescriptor>
#[derive(Default, Debug, PartialEq, Eq, Clone, serde::Serialize, serde::Deserialize)]
pub struct ExecutingDescriptor {
    /// The timestamp used to enforce timestamp [invariant](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/derivation.md#invariants)
    #[serde(with = "alloy_serde::quantity")]
    timestamp: u64,
    /// The timeout that requests verification to still hold at `timestamp+timeout`
    /// (message expiry may drop previously valid messages).
    #[serde(skip_serializing_if = "Option::is_none", with = "alloy_serde::quantity::opt")]
    timeout: Option<u64>,
}

impl ExecutingDescriptor {
    /// Create a new [`ExecutingDescriptor`] from the timestamp and timeout
    pub const fn new(timestamp: u64, timeout: Option<u64>) -> Self {
        Self { timestamp, timeout }
    }
}
