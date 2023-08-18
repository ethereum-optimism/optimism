## Labels

Labels are divided into categories with their descriptions annotated as `<Category Name>: <description>`.

The following are a comprehensive list of label categories.

- **Area labels** ([`A-`][area]): Denote the general area for the related issue or PR changes.
- **Category labels** ([`C-`][category]): Contextualize the type of issue or change.
- **Meta labels** ([`M-`][meta]): These add context to the issues or prs themselves primarily relating to process.
- **Difficulty labels** ([`D-`][difficulty]): Describe the associated implementation's difficulty level.
- **Status labels** ([`S-`][status]): Specify the status of an issue or pr.

Labels also provide a versatile filter for finding tickets that need help or are open for assignment.
This makes them a great tool for contributors!

[area]: https://github.com/ethereum-optimism/optimism/labels?q=a-
[category]: https://github.com/ethereum-optimism/optimism/labels?q=c-
[meta]: https://github.com/ethereum-optimism/optimism/labels?q=m-
[difficulty]: https://github.com/ethereum-optimism/optimism/labels?q=d-
[status]: https://github.com/ethereum-optimism/optimism/labels?q=s-

#### Filtering for Work

To find tickets available for external contribution, take a look at the https://github.com/ethereum-optimism/optimism/labels/M-community label.

You can filter by the https://github.com/ethereum-optimism/optimism/labels/D-good-first-issue
label to find issues that are intended to be easy to implement or fix.

Also, all labels can be seen by visiting the [labels page][labels]

[labels]: https://github.com/ethereum-optimism/optimism/labels

#### Modifying Labels

When altering label names or deleting labels there are a few things you must be aware of.

- This may affect the mergify bot's use of labels. See the [mergify config](.github/mergify.yml).
- If the https://github.com/ethereum-optimism/labels/S-stale label is altered, the [close-stale](.github/workflows/close-stale.yml) workflow should be updated.
- If the https://github.com/ethereum-optimism/labels/M-dependabot label is altered, the [dependabot config](.github/dependabot.yml) file should be adjusted.
- Saved label filters for project boards will not automatically update. These should be updated if label names change.
