name: Feature request
description: Suggest a feature
labels: ["C-enhancement", "S-needs-triage"]
body:
  - type: markdown
    attributes:
      value: |
        Please ensure that the feature has not already been requested in the issue tracker.
  - type: textarea
    attributes:
      label: Describe the feature
      description: |
        Please describe the feature and what it is aiming to solve, if relevant.

    validations:
      required: true
  - type: textarea
    attributes:
      label: Additional context
      description: Add any other context to the feature (screenshots, resources, etc.)
