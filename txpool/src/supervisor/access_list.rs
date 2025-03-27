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
use crate::supervisor::CROSS_L2_INBOX_ADDRESS;
use alloy_eips::eip2930::AccessListItem;
use alloy_primitives::B256;

/// Parses [`AccessListItem`]s to inbox entries.
///
/// Return flattened iterator with all inbox entries.
pub fn parse_access_list_items_to_inbox_entries<'a>(
    access_list_items: impl Iterator<Item = &'a AccessListItem>,
) -> impl Iterator<Item = &'a B256> {
    access_list_items.filter_map(parse_access_list_item_to_inbox_entries).flatten()
}

/// Parse [`AccessListItem`] to inbox entries, if any.
/// Max 3 inbox entries can exist per [`AccessListItem`] that points to [`CROSS_L2_INBOX_ADDRESS`].
///
/// Returns `Vec::new()` if [`AccessListItem`] address doesn't point to [`CROSS_L2_INBOX_ADDRESS`].
// TODO: add url to spec once [pr](https://github.com/ethereum-optimism/specs/pull/612) is merged
fn parse_access_list_item_to_inbox_entries(
    access_list_item: &AccessListItem,
) -> Option<impl Iterator<Item = &B256>> {
    (access_list_item.address == CROSS_L2_INBOX_ADDRESS)
        .then(|| access_list_item.storage_keys.iter())
}
