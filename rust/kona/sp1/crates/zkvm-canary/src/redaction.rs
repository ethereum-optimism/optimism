//! Redaction for diagnostic text emitted by the canary.

use url::Url;

const REDACTED: &str = "[redacted]";
const REDACTED_URL: &str = "[redacted-url]";
const SECRET_KEYS: [&str; 3] = ["token=", "api_key=", "apikey="];

/// Removes URL credentials, URL queries, and explicit secret assignments from a diagnostic.
pub fn redact_sensitive_detail(detail: &str) -> String {
    let mut redacted = String::with_capacity(detail.len());
    let mut token_start = 0;
    for (index, character) in detail.char_indices() {
        if character.is_whitespace() {
            redacted.push_str(&redact_token(&detail[token_start..index]));
            redacted.push(character);
            token_start = index + character.len_utf8();
        }
    }
    redacted.push_str(&redact_token(&detail[token_start..]));
    redacted
}

fn redact_token(token: &str) -> String {
    redact_secret_assignments(&redact_urls(token))
}

fn redact_urls(token: &str) -> String {
    let mut redacted = String::with_capacity(token.len());
    let mut cursor = 0;
    while let Some((scheme_start, marker)) = next_url(token, cursor) {
        redacted.push_str(&token[cursor..scheme_start]);
        let raw_end = next_url(token, marker + 3).map_or(token.len(), |(start, _)| start);
        let candidate_len = token[scheme_start..raw_end].trim_end_matches(is_url_suffix).len();
        let candidate_end = scheme_start + candidate_len;
        redacted.push_str(&redact_url(&token[scheme_start..candidate_end]));
        redacted.push_str(&token[candidate_end..raw_end]);
        cursor = raw_end;
    }
    redacted.push_str(&token[cursor..]);
    redacted
}

fn next_url(token: &str, from: usize) -> Option<(usize, usize)> {
    let marker = from + token[from..].find("://")?;
    let scheme_start = token[..marker]
        .char_indices()
        .rev()
        .find(|(_, character)| !is_scheme_character(*character))
        .map_or(0, |(index, character)| index + character.len_utf8());
    Some((scheme_start, marker))
}

fn redact_url(url_text: &str) -> String {
    let Ok(mut url) = Url::parse(url_text) else {
        return REDACTED_URL.to_string();
    };
    if url.set_password(None).is_err() || url.set_username("").is_err() {
        return REDACTED_URL.to_string();
    }
    url.set_query(None);
    url.set_fragment(None);
    url.to_string()
}

const fn is_scheme_character(character: char) -> bool {
    character.is_ascii_alphanumeric() || matches!(character, '+' | '-' | '.')
}

const fn is_url_suffix(character: char) -> bool {
    matches!(character, ',' | ';' | ')' | ']' | '}' | '\'' | '"')
}

fn redact_secret_assignments(value: &str) -> String {
    let lower = value.to_ascii_lowercase();
    let mut redacted = String::with_capacity(value.len());
    let mut cursor = 0;
    while let Some((start, key_len)) = next_secret_assignment(&lower, cursor) {
        let value_start = start + key_len;
        redacted.push_str(&value[cursor..value_start]);
        redacted.push_str(REDACTED);
        if value[value_start..].starts_with(REDACTED) {
            cursor = value_start + REDACTED.len();
            continue;
        }
        cursor = value[value_start..]
            .find(is_secret_value_delimiter)
            .map_or(value.len(), |offset| value_start + offset);
    }
    redacted.push_str(&value[cursor..]);
    redacted
}

fn next_secret_assignment(lower: &str, cursor: usize) -> Option<(usize, usize)> {
    SECRET_KEYS
        .iter()
        .filter_map(|key| {
            lower[cursor..].match_indices(key).find_map(|(offset, _)| {
                let start = cursor + offset;
                let boundary = start == 0 || !lower.as_bytes()[start - 1].is_ascii_alphanumeric();
                boundary.then_some((start, key.len()))
            })
        })
        .min_by_key(|(start, _)| *start)
}

const fn is_secret_value_delimiter(character: char) -> bool {
    matches!(character, '&' | ',' | ';' | ')' | ']' | '}' | '\'' | '"')
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn malformed_url_redacts_only_the_url_token() {
        assert_eq!(
            redact_sensitive_detail("request failed for https://user:secret@; retrying"),
            "request failed for [redacted-url]; retrying",
        );
    }

    #[test]
    fn non_ascii_context_before_a_url_is_preserved() {
        assert_eq!(
            redact_sensitive_detail("request→https://user:secret@example.test/path?key=hidden"),
            "request→https://example.test/path",
        );
    }

    #[test]
    fn redaction_is_idempotent() {
        let once = redact_sensitive_detail("guest rejected with token=hidden");
        assert_eq!(redact_sensitive_detail(&once), once);
    }

    #[test]
    fn every_url_in_one_token_is_redacted() {
        assert_eq!(
            redact_sensitive_detail(
                "requests failed: https://one:secret@a.test/x?token=first,https://two:hidden@b.test/y?api_key=second"
            ),
            "requests failed: https://a.test/x,https://b.test/y",
        );
    }
}
