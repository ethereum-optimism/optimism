# Linting

See

- [markdownlint rule reference](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md)
- [exemple .markdownlint.json file](https://github.com/DavidAnson/markdownlint/blob/main/schema/.markdownlint.jsonc)

Justification for linting rules in [.markdownlint.json](/.markdownlint.json):

- *line_length* (`!strict && stern`): don't trip up on url lines
- *no-blanks-blockquote*: enable multiple consecutive blockquotes separated by white lines
- *single-title*: enable reusing `<h1>` for content
- *no-emphasis-as-heading*: enable emphasized paragraphs
